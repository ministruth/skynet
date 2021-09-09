package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"skynet/plugin/monitor/msg"
	"skynet/plugin/monitor/shared"
	"skynet/sn/utils"
	"strconv"
	"strings"
	"time"

	logrus_stack "github.com/Gurpartap/logrus-stack"
	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/kballard/go-shellquote"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

var ErrRestart = errors.New("restart triggered")

var (
	token    string
	ssl      bool
	insecure bool
	logfile  string
	maxTime  int
)

var rootCmd = &cobra.Command{
	Use:   "agent HOST",
	Short: "Skynet monitor agent",
	Args:  cobra.ExactArgs(1),
	Run:   run,
}

const pluginPath = "/api/plugin/2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa/ws"

var sleepTime = 1
var recvChan shared.ChanMap

func sleep() {
	log.Warnf("Retry in %v seconds...", sleepTime)
	time.Sleep(time.Second * time.Duration(sleepTime))
	if sleepTime < maxTime {
		sleepTime = utils.IntMin(sleepTime*2, maxTime)
	}
}

func loginHandler(c *shared.Websocket) error {
	id, err := machineid.ID()
	if err != nil {
		log.Fatal(err)
	}

	mid, err := msg.SendMsgByte(c, uuid.Nil, msg.OPLogin, msg.Marshal(msg.LoginMsg{
		UID:   utils.MD5(id),
		Token: token,
	}))
	if err != nil {
		return err
	}

	r, _, err := msg.RecvMsg(c)
	if err != nil {
		return err
	}
	if r.ID != mid || r.OPCode != msg.OPRet {
		log.Fatal("Invalid server response")
	}
	var ret msg.RetMsg
	err = msg.Unmarshal(r.Data, &ret)
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

func sendInfoHandler(c *shared.Websocket) error {
	hostname, err := os.Hostname()
	if err != nil {
		log.Warn("Could not determine hostname")
	}
	uts := &unix.Utsname{}
	err = unix.Uname(uts)
	if err != nil {
		log.Warn("Could not determine uname")
	}
	_, err = msg.SendMsgByte(c, uuid.Nil, msg.OPInfo, msg.Marshal(msg.InfoMsg{
		Version: shared.AgentVersion,
		Host:    hostname,
		Machine: string(bytes.TrimRight(uts.Machine[:], "\x00")),
		System: string(bytes.TrimRight(uts.Sysname[:], "\x00")) + " " +
			string(bytes.TrimRight(uts.Nodename[:], "\x00")) + " " +
			string(bytes.TrimRight(uts.Release[:], "\x00")),
	}))
	return err
}

func msgReturnHandler(c *shared.Websocket, m *msg.CommonMsg) error {
	var data msg.RetMsg
	if err := msg.Unmarshal(m.Data, &data); err != nil {
		return err
	}
	recvChan.Push(m.ID, data)
	return nil
}

func msgCMDHandler(c *shared.Websocket, m *msg.CommonMsg) error {
	var data msg.CMDMsg
	if err := msg.Unmarshal(m.Data, &data); err != nil {
		return err
	}
	w, err := shellquote.Split(data.Payload)
	if err != nil {
		log.Warn(err)
		if data.Sync {
			err = msg.SendMsgRet(c, m.ID, -1, err.Error())
		} else {
			_, err = msg.SendMsgByte(c, uuid.Nil, msg.OPCMDRes, msg.Marshal(msg.CMDResMsg{
				CID:  data.CID,
				Data: err.Error(),
				End:  true,
			}))
		}
		return err
	} else {
		log.Info("Run command: ", data.Payload)
		if data.Sync {
			go RunCommandSync(c, m.ID, data.CID, w[0], w[1:]...)
		} else {
			RunCommandAsync(c, data.CID, w[0], w[1:]...)
		}
	}
	return nil
}

func msgCMDKillHandler(c *shared.Websocket, m *msg.CommonMsg) error {
	var data msg.CMDKillMsg
	if err := msg.Unmarshal(m.Data, &data); err != nil {
		return err
	}
	log.Warn("Cancel command: ", data.CID)
	return KillCommand(data.CID)
}

func msgFileHandler(c *shared.Websocket, m *msg.CommonMsg) error {
	var data msg.FileMsg
	if err := msg.Unmarshal(m.Data, &data); err != nil {
		return err
	}
	log.Info("Receive file: ", data.Path)
	if data.Recursive {
		dir, _ := filepath.Split(data.Path)
		os.MkdirAll(dir, data.Perm)
	}
	if !data.Override && utils.FileExist(data.Path) {
		return msg.SendMsgRet(c, m.ID, 0, "File not change")
	}
	os.Remove(data.Path) // support override running program
	err := ioutil.WriteFile(data.Path, data.File, data.Perm)
	if err != nil {
		log.Warn(err)
		return msg.SendMsgRet(c, m.ID, -1, err.Error())
	} else {
		return msg.SendMsgRet(c, m.ID, 0, "Write file success")
	}
}

func msgShellHandler(c *shared.Websocket, m *msg.CommonMsg) error {
	var data msg.ShellMsg
	if err := msg.Unmarshal(m.Data, &data); err != nil {
		return err
	}
	switch data.OPCode {
	case msg.ShellConnect:
		var cmsg msg.ShellConnectMsg
		if err := msg.Unmarshal(data.Data, &cmsg); err != nil {
			return err
		}
		id, err := CreateShell(&cmsg.ShellSizeMsg)
		if err != nil {
			log.Warn(err)
			return msg.SendMsgRet(c, m.ID, -1, err.Error())
		}
		go HandleShellOutput(c, id)
		msg.SendMsgRet(c, m.ID, 0, id.String())
		log.Info("Shell connected: ", id)
	case msg.ShellDisconnect:
		CloseShell(data.SID)
		log.Info("Shell disconnect: ", data.SID)
	case msg.ShellSize:
		var size msg.ShellSizeMsg
		if err := msg.Unmarshal(data.Data, &size); err != nil {
			return err
		}
		SetShellSize(data.SID, &size)
	case msg.ShellInput:
		HandleShellInput(data.SID, string(data.Data))
	default:
		log.Warn("Unknown shell opcode ", data.OPCode)
	}
	return nil
}

func msgReqStatHandler(c *shared.Websocket, m *msg.CommonMsg) error {
	cpuUsage, err := cpu.Percent(0, false)
	if err != nil {
		log.Warn("Could not determine cpu usage")
		cpuUsage = []float64{0}
	}
	memUsage, err := mem.VirtualMemory()
	if err != nil {
		log.Warn("Could not determine mem usage")
		memUsage = &mem.VirtualMemoryStat{}
	}
	partionUsage, err := disk.Partitions(false)
	if err != nil {
		log.Warn("Could not determine disk usage")
	}
	var diskUsage, disktotUsage uint64
	for _, v := range partionUsage {
		usage, err := disk.Usage(v.Mountpoint)
		if err != nil {
			log.Warn("Could not determine disk usage")
			continue
		}
		diskUsage += usage.Used
		disktotUsage += usage.Total
	}
	loadUsage, err := load.Avg()
	if err != nil {
		log.Warn("Could not determine load usage")
		loadUsage = &load.AvgStat{}
	}
	netUsage, err := net.IOCounters(true)
	var txByte, rxByte uint64
	if err != nil {
		log.Warn("Could not determine net usage")
	}
	for _, v := range netUsage {
		if strings.HasSuffix(v.Name, "br-") ||
			strings.HasSuffix(v.Name, "docker") ||
			strings.HasSuffix(v.Name, "lo") ||
			strings.HasSuffix(v.Name, "veth") {
			// except virtual interface and loop
			continue
		}
		txByte += v.BytesSent
		rxByte += v.BytesRecv
	}

	tm, err := strconv.ParseInt(string(m.Data), 10, 64)
	if err != nil {
		return err
	}
	_, err = msg.SendMsgByte(c, uuid.Nil, msg.OPStat, msg.Marshal(msg.StatMsg{
		Time:      time.Unix(0, tm),
		CPU:       cpuUsage[0],
		Mem:       memUsage.Used,
		TotalMem:  memUsage.Total,
		Disk:      diskUsage,
		TotalDisk: disktotUsage,
		Load1:     loadUsage.Load1,
		BandUp:    txByte,
		BandDown:  rxByte,
	}))
	return err
}

func deadloop(url string) error {
	conn, _, err := shared.DialWebsocket(&websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: insecure},
	}, url, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Info("Connected")
	sleepTime = 1 // reset sleeptime

	if err = loginHandler(conn); err != nil {
		return err
	}
	if err = sendInfoHandler(conn); err != nil {
		return err
	}

	defer func() {
		shells := shellInstance.Keys()
		for _, v := range shells {
			CloseShell(v)
		}
	}()

	for {
		res, msgRead, err := msg.RecvMsg(conn)
		if err != nil {
			if msgRead == nil {
				break
			} else {
				log.Warn(err)
				continue
			}
		}
		switch res.OPCode {
		case msg.OPRet:
			err = msgReturnHandler(conn, res)
		case msg.OPCMD:
			err = msgCMDHandler(conn, res)
		case msg.OPCMDKill:
			err = msgCMDKillHandler(conn, res)
		case msg.OPFile:
			err = msgFileHandler(conn, res)
		case msg.OPShell:
			err = msgShellHandler(conn, res)
		case msg.OPReqStat:
			err = msgReqStatHandler(conn, res)
		case msg.OPRestart:
			return ErrRestart
		default:
			log.Warn("Unknown opcode ", res.OPCode)
		}
		if err != nil {
			log.Warn(err)
		}
	}
	log.Error("lost connection")
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
			if errors.Is(err, ErrRestart) {
				path := os.Args[0]
				var args []string
				if len(os.Args) > 1 {
					args = os.Args[1:]
				}

				cmd := exec.Command(path, args...)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Stdin = os.Stdin

				if err = cmd.Start(); err != nil {
					log.Fatalf("Restart: Failed to launch, error: %v", err)
				}
				return
			} else {
				log.Error(err)
			}
		}
		sleep()
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
