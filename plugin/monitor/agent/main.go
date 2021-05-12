package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"math"
	"os"
	"skynet/plugin/monitor/msg"
	"skynet/sn/utils"
	"time"

	logrus_stack "github.com/Gurpartap/logrus-stack"
	"github.com/denisbrodbeck/machineid"
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

const pluginPath = "/v1/plugin/2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa"

var sleepTime = 1
var maxTime int

func sleep() {
	log.Warnf("Retry in %v seconds...", sleepTime)
	time.Sleep(time.Second * time.Duration(sleepTime))
	if sleepTime < maxTime {
		sleepTime = int(math.Min(float64(sleepTime*2), float64(maxTime)))
	}
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

	// login
	id, err := machineid.ID()
	if err != nil {
		log.Fatal(err)
	}

	d, _ := json.Marshal(msg.LoginMsg{
		UID:   utils.MD5(id),
		Token: token,
	})
	err = msg.SendReq(c, msg.OPLogin, string(d))
	if err != nil {
		return err
	}

	// ret
	r, err := msg.Recv(c)
	if err != nil {
		return err
	}
	if r.Code != 0 {
		log.Fatal(r.Msg)
	}
	log.WithFields(log.Fields{
		"id": id,
	}).Info("Login success")

	// info
	hostname, err := os.Hostname()
	if err != nil {
		log.Warn("Could not determine hostname")
	}
	uts := &unix.Utsname{}
	err = unix.Uname(uts)
	if err != nil {
		log.Warn("Could not determine uname")
	}
	d, _ = json.Marshal(msg.InfoMsg{
		Host:    hostname,
		Machine: string(bytes.TrimRight(uts.Machine[:], "\x00")),
		System: string(bytes.TrimRight(uts.Sysname[:], "\x00")) + " " +
			string(bytes.TrimRight(uts.Nodename[:], "\x00")) + " " +
			string(bytes.TrimRight(uts.Release[:], "\x00")),
	})
	err = msg.SendReq(c, msg.OPInfo, string(d))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go UploadStat(ctx, c)

	for {
		_, msgRead, err := c.ReadMessage()
		if err != nil {
			break
		}
		var res msg.CommonMsg
		err = json.Unmarshal(msgRead, &res)
		if err != nil {
			log.Warn("Msg format error")
			continue
		}
		switch res.Opcode {
		case msg.OPCMD:
			var data msg.CMDMsg
			err = json.Unmarshal([]byte(res.Data), &data)
			if err != nil {
				log.Warn("Msg format error")
				continue
			}
			w, err := shellquote.Split(data.Data)
			if err != nil {
				d, _ := json.Marshal(msg.CMDMsg{
					UID:  data.UID,
					Data: err.Error(),
					End:  true,
				})
				err = msg.SendReq(c, msg.OPCMDRes, string(d))
				if err != nil {
					log.Warn("Could not send cmd result")
				}
			} else {
				log.Info("Run command: ", data.Data)
				RunCommand(c, data.UID, w[0], w[1:]...)
			}
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
