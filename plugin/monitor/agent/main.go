package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"skynet/plugin/monitor/msg"
	"skynet/sn/utils"
	"time"

	logrus_stack "github.com/Gurpartap/logrus-stack"
	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/kballard/go-shellquote"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

var token string
var ssl bool
var insecure bool
var logfile string

var rootCmd = &cobra.Command{
	Use:   "agent HOST",
	Short: "Skynet monitor agent",
	Args:  cobra.ExactArgs(1),
	Run:   run,
}

const pluginPath = "/api/plugin/2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa/ws"

var sleepTime = 1
var maxTime int
var recvChan = make(map[uuid.UUID]chan msg.RetMsg)

func sleep() {
	log.Warnf("Retry in %v seconds...", sleepTime)
	time.Sleep(time.Second * time.Duration(sleepTime))
	if sleepTime < maxTime {
		sleepTime = utils.IntMin(sleepTime*2, maxTime)
	}
}

func login(c *websocket.Conn) error {
	// login
	id, err := machineid.ID()
	if err != nil {
		log.Fatal(err)
	}

	d, err := json.Marshal(msg.LoginMsg{
		UID:   utils.MD5(id),
		Token: token,
	})
	if err != nil {
		log.Fatal(err)
	}
	mid, err := msg.SendReq(c, msg.OPLogin, string(d))
	if err != nil {
		return err
	}

	r, _, err := msg.Recv(c)
	if err != nil {
		return err
	}
	if r.ID != mid || r.Opcode != msg.OPRet {
		log.Fatal("Invalid server response")
	}
	var ret msg.RetMsg
	err = json.Unmarshal([]byte(r.Data), &ret)
	if err != nil {
		return err
	}
	if ret.Code != 0 {
		log.Fatal(ret.Data)
	}
	log.WithFields(log.Fields{
		"id": id,
	}).Info("Login success")
	return nil
}

func sendInfo(c *websocket.Conn) error {
	hostname, err := os.Hostname()
	if err != nil {
		log.Warn("Could not determine hostname")
	}
	uts := &unix.Utsname{}
	err = unix.Uname(uts)
	if err != nil {
		log.Warn("Could not determine uname")
	}
	d, err := json.Marshal(msg.InfoMsg{
		Host:    hostname,
		Machine: string(bytes.TrimRight(uts.Machine[:], "\x00")),
		System: string(bytes.TrimRight(uts.Sysname[:], "\x00")) + " " +
			string(bytes.TrimRight(uts.Nodename[:], "\x00")) + " " +
			string(bytes.TrimRight(uts.Release[:], "\x00")),
	})
	if err != nil {
		log.Fatal(err)
	}
	_, err = msg.SendReq(c, msg.OPInfo, string(d))
	return err
}

func deadloop(u string) error {
	websocket.DefaultDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: insecure}
	c, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		return err
	}
	defer c.Close()

	log.Info("Connected")
	sleepTime = 1

	err = login(c)
	if err != nil {
		return err
	}

	err = sendInfo(c)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go UploadStat(ctx, c)

	for {
		res, msgRead, err := msg.Recv(c)
		if err != nil {
			if msgRead == nil {
				break
			} else {
				log.Warn("Msg format error")
				continue
			}
		}
		switch res.Opcode {
		case msg.OPRet:
			var data msg.RetMsg
			err = json.Unmarshal([]byte(res.Data), &data)
			if err != nil {
				log.Warn("Msg format error")
				continue
			}
			ch, ok := recvChan[res.ID]
			if !ok {
				log.Warn("Error response: " + res.Data)
			} else {
				ch <- data
			}
		case msg.OPCMD:
			var data msg.CMDMsg
			err = json.Unmarshal([]byte(res.Data), &data)
			if err != nil {
				log.Warn("Msg format error")
				continue
			}
			w, err := shellquote.Split(data.Payload)
			if err != nil {
				if data.Sync {
					err = msg.SendRsp(c, res.ID, -1, err.Error())
				} else {
					d, err := json.Marshal(msg.CMDResMsg{
						UID:  data.UID,
						Data: err.Error(),
						End:  true,
					})
					if err != nil {
						log.Fatal(err)
					}
					_, err = msg.SendReq(c, msg.OPCMDRes, string(d))
				}
				if err != nil {
					log.Warn("Could not send cmd result")
				}
			} else {
				log.Info("Run command: ", data.Payload)
				if data.Sync {
					go RunCommandSync(c, res.ID, data.UID, w[0], w[1:]...)
				} else {
					RunCommandAsync(c, data.UID, w[0], w[1:]...)
				}
			}
		case msg.OPCMDKill:
			var data msg.CMDKillMsg
			err = json.Unmarshal([]byte(res.Data), &data)
			if err != nil {
				log.Warn("Msg format error")
				continue
			}
			log.Warn("Cancel command: ", data.UID)
			err = KillCommand(data.UID)
			if data.Return {
				if err != nil {
					msg.SendRsp(c, res.ID, -1, err.Error())
				} else {
					msg.SendRsp(c, res.ID, 0, "Kill task success")
				}
			}
		case msg.OPFile:
			var data msg.FileMsg
			err = json.Unmarshal([]byte(res.Data), &data)
			if err != nil {
				log.Warn("Msg format error")
				continue
			}
			if data.Recursive {
				dir, _ := filepath.Split(data.Path)
				os.MkdirAll(dir, data.Perm)
			}
			if !data.Override && utils.FileExist(data.Path) {
				msg.SendRsp(c, res.ID, 0, "File not change")
			}
			err := ioutil.WriteFile(data.Path, data.File, data.Perm)
			if err != nil {
				msg.SendRsp(c, res.ID, -1, err.Error())
			} else {
				msg.SendRsp(c, res.ID, 0, "Write file success")
			}
		default:
			log.Warn("Unknown opcode ", res.Opcode)
		}
	}
	log.Warn("lost connection")
	sleep()
	return nil
}

func run(cmd *cobra.Command, args []string) {
	var err error
	var logFile *os.File
	if logfile != "" {
		log.SetFormatter(&log.JSONFormatter{})
		logFile, err = os.OpenFile(logfile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0755)
		if err != nil {
			panic(err)
		}
		mw := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(mw)
		log.AddHook(logrus_stack.StandardHook())
	}
	defer logFile.Close()

	var u string
	if ssl {
		u = "wss://" + args[0] + pluginPath
	} else {
		u = "ws://" + args[0] + pluginPath
	}
	log.Info("Connecting to ", u)

	os.Mkdir("plugin", 0755)

	for {
		err := deadloop(u)
		if err != nil {
			log.Error(err)
			sleep()
		}
	}
}

func main() {
	rootCmd.Flags().StringVarP(&logfile, "file", "f", "", "logfile path")
	rootCmd.Flags().BoolVarP(&ssl, "ssl", "s", false, "enable ssl")
	rootCmd.Flags().BoolVar(&insecure, "insecure", false, "do not verify ssl certificate")
	rootCmd.Flags().StringVarP(&token, "token", "t", "", "connect token")
	rootCmd.Flags().IntVar(&maxTime, "maxtime", 16, "max wait time when retrying")
	rootCmd.MarkFlagRequired("token")

	rootCmd.Execute()
}
