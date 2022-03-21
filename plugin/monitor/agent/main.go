package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"skynet/plugin/monitor/msg"
	"skynet/plugin/monitor/shared"
	"skynet/sn/utils"
	"strings"
	"syscall"
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
	"github.com/ztrue/tracerr"
	"golang.org/x/sys/unix"
)

var ErrRestart = tracerr.New("restart triggered")

var (
	token      string
	ssl        bool
	insecure   bool
	logfile    string
	maxTime    int
	netConfig  string
	diskConfig string
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
		sleepTime = utils.Min(sleepTime*2, maxTime)
	}
}

func loginHandler(c *shared.Websocket) error {
	id, err := machineid.ID()
	if err != nil {
		utils.WithTrace(tracerr.Wrap(err)).Fatal(err)
	}

	mid, err := msg.SendMsg(c, uuid.Nil, msg.AgentMessage_LOGIN, &msg.AgentMessage_Login{
		Login: &msg.LoginMessage{
			Uid:   utils.MD5(id),
			Token: token,
		},
	})
	if err != nil {
		return err
	}

	r, _, err := msg.RecvMsg(c)
	if err != nil {
		return err
	}
	if r.Id != mid.String() || r.Type != msg.AgentMessage_RETURN {
		log.Fatal("Invalid server response")
	}
	ret := r.GetReturn()
	if ret == nil {
		return msg.ErrFormat
	}
	if ret.Code != msg.ReturnMessage_OK {
		log.Fatal(ret.Data)
	}
	log.WithFields(log.Fields{
		"id": utils.MD5(id),
	}).Info("Login success")
	return nil
}

func parseUname() (string, string, error) {
	uts := &unix.Utsname{}
	if err := unix.Uname(uts); err != nil {
		return "", "", err
	}
	return string(bytes.TrimRight(uts.Machine[:], "\x00")),
		string(bytes.TrimRight(uts.Sysname[:], "\x00")) + " " +
			string(bytes.TrimRight(uts.Nodename[:], "\x00")) + " " +
			string(bytes.TrimRight(uts.Release[:], "\x00")), nil
}

func sendInfoHandler(c *shared.Websocket) error {
	hostname, err := os.Hostname()
	if err != nil {
		utils.WithTrace(tracerr.Wrap(err)).Warn(err)
	}
	machine, system, _ := parseUname()
	_, err = msg.SendMsg(c, uuid.Nil, msg.AgentMessage_INFO, &msg.AgentMessage_Info{
		Info: &msg.InfoMessage{
			Version:  shared.AgentVersion,
			OS:       runtime.GOOS,
			Hostname: hostname,
			Machine:  machine,
			System:   system,
		},
	})
	return err
}

func msgReturnHandler(c *shared.Websocket, m *msg.AgentMessage) error {
	data := m.GetReturn()
	if data == nil {
		return msg.ErrFormat
	}
	recvChan.Push(uuid.MustParse(m.Id), data)
	return nil
}

func msgCMDHandler(c *shared.Websocket, m *msg.AgentMessage) error {
	cmd := m.GetCommand()
	if cmd == nil {
		return msg.ErrFormat
	}
	switch cmd.Type {
	case msg.CommandMessage_RUN:
		data := cmd.GetRun()
		if data == nil {
			return msg.ErrFormat
		}
		cid, err := uuid.Parse(data.Cid)
		if err != nil {
			return msg.ErrFormat
		}
		w, err := shellquote.Split(data.Payload)
		if err != nil {
			if data.Sync {
				err = msg.SendMsgRet(c, uuid.MustParse(m.Id), -1, err.Error())
			} else {
				_, err = msg.SendMsg(c, uuid.Nil, msg.AgentMessage_COMMAND, &msg.AgentMessage_Command{
					Command: &msg.CommandMessage{
						Type: msg.CommandMessage_RESULT,
						Data: &msg.CommandMessage_Res{
							Res: &msg.CMDResMessage{
								Cid:  data.Cid,
								Data: err.Error(),
								End:  true,
							},
						},
					},
				})
			}
			return err
		} else {
			log.Info("Run command: ", data.Payload)
			if data.Sync {
				go RunCommandSync(c, uuid.MustParse(m.Id), cid, w[0], w[1:]...)
			} else {
				RunCommandAsync(c, cid, w[0], w[1:]...)
			}
		}
	case msg.CommandMessage_KILL:
		data := cmd.GetKill()
		if data == nil {
			return msg.ErrFormat
		}
		cid, err := uuid.Parse(data.Cid)
		if err != nil {
			return msg.ErrFormat
		}
		log.Warn("Cancel command: ", data.Cid)
		return KillCommand(cid)
	}
	return nil
}

func msgFileHandler(c *shared.Websocket, m *msg.AgentMessage) error {
	data := m.GetFile()
	if data == nil {
		return msg.ErrFormat
	}
	time.Sleep(3 * time.Second)
	log.Info("Receive file: ", data.Path)
	if data.Recursive {
		dir, _ := filepath.Split(data.Path)
		os.MkdirAll(dir, fs.FileMode(data.Perm))
	}
	if !data.Override && utils.FileExist(data.Path) {
		return msg.SendMsgRet(c, uuid.MustParse(m.Id), 0, "File not change")
	}
	os.Remove(data.Path) // support override running program
	err := ioutil.WriteFile(data.Path, data.File, fs.FileMode(data.Perm))
	if err != nil {
		utils.WithTrace(tracerr.Wrap(err)).Warn(err)
		return msg.SendMsgRet(c, uuid.MustParse(m.Id), -1, err.Error())
	} else {
		return msg.SendMsgRet(c, uuid.MustParse(m.Id), 0, "Write file success")
	}
}

func msgShellHandler(c *shared.Websocket, m *msg.AgentMessage) error {
	data := m.GetShell()
	if data == nil {
		return msg.ErrFormat
	}
	var sid uuid.UUID
	var err error
	if data.Sid != "" {
		sid, err = uuid.Parse(data.Sid)
		if err != nil {
			return msg.ErrFormat
		}
	}
	switch data.Type {
	case msg.ShellMessage_CONNECT:
		cmsg := data.GetConnect()
		if cmsg == nil {
			return msg.ErrFormat
		}
		id, err := CreateShell(cmsg.Size)
		if err != nil {
			return msg.SendMsgRet(c, uuid.MustParse(m.Id), -1, err.Error())
		}
		go HandleShellOutput(c, id)
		msg.SendMsgRet(c, uuid.MustParse(m.Id), 0, id.String())
		log.Info("Shell connected: ", id)
	case msg.ShellMessage_DISCONNECT:
		CloseShell(sid)
		log.Info("Shell disconnect: ", sid)
	case msg.ShellMessage_SIZE:
		size := data.GetSize()
		if size == nil {
			return msg.ErrFormat
		}
		SetShellSize(sid, size)
	case msg.ShellMessage_INPUT:
		HandleShellInput(sid, string(data.GetPutdata()))
	default:
		log.Warn("Unknown shell opcode ", data.Type)
	}
	return nil
}

func msgReqStatHandler(c *shared.Websocket, m *msg.AgentMessage) error {
	cpuUsage, err := cpu.Percent(0, false)
	if err != nil {
		utils.WithTrace(tracerr.Wrap(err)).Warn(err)
		cpuUsage = []float64{0}
	}
	memUsage, err := mem.VirtualMemory()
	if err != nil {
		utils.WithTrace(tracerr.Wrap(err)).Warn(err)
		memUsage = &mem.VirtualMemoryStat{}
	}
	partionUsage, err := disk.Partitions(false)
	if err != nil {
		utils.WithTrace(tracerr.Wrap(err)).Warn(err)
	}
	var diskUsage, disktotUsage uint64
	for _, v := range partionUsage {
		usage, err := disk.Usage(v.Mountpoint)
		if err != nil {
			utils.WithTrace(tracerr.Wrap(err)).Warn(err)
			continue
		}
		if diskConfig != "" {
			for _, nv := range strings.Split(diskConfig, ",") {
				if nv == v.Device {
					diskUsage += usage.Used
					disktotUsage += usage.Total
					break
				}
			}
		} else {
			diskUsage += usage.Used
			disktotUsage += usage.Total
		}
	}
	loadUsage, err := load.Avg()
	if err != nil {
		utils.WithTrace(tracerr.Wrap(err)).Warn(err)
		loadUsage = &load.AvgStat{}
	}
	netUsage, err := net.IOCounters(true)
	var txByte, rxByte uint64
	if err != nil {
		utils.WithTrace(tracerr.Wrap(err)).Warn(err)
	}
	for _, v := range netUsage {
		if netConfig != "" {
			for _, nv := range strings.Split(netConfig, ",") {
				if nv == v.Name {
					txByte += v.BytesSent
					rxByte += v.BytesRecv
					break
				}
			}
		} else {
			if strings.HasPrefix(v.Name, "br-") ||
				strings.HasPrefix(v.Name, "docker") ||
				strings.HasPrefix(v.Name, "lo") ||
				strings.HasPrefix(v.Name, "veth") {
				// except virtual interface and loop
				continue
			} else {
				txByte += v.BytesSent
				rxByte += v.BytesRecv
			}
		}
	}

	req := m.GetStatusReq()
	if req == nil {
		return msg.ErrFormat
	}
	_, err = msg.SendMsg(c, uuid.Nil, msg.AgentMessage_STATUS, &msg.AgentMessage_StatusRsp{
		StatusRsp: &msg.StatusRspMessage{
			Time:      req.Time,
			CPU:       cpuUsage[0],
			Mem:       memUsage.Used,
			TotalMem:  memUsage.Total,
			Disk:      diskUsage,
			TotalDisk: disktotUsage,
			Load1:     loadUsage.Load1,
			BandUp:    txByte,
			BandDown:  rxByte,
		},
	})
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

	errWrap := func(conn *shared.Websocket, res *msg.AgentMessage,
		f func(c *shared.Websocket, m *msg.AgentMessage) error) {
		if e := f(conn, res); e != nil {
			utils.WithTrace(e).Warn(err)
		}
	}

	for {
		res, msgRead, err := msg.RecvMsg(conn)
		if err != nil {
			if msgRead == nil {
				break
			} else {
				utils.WithTrace(err).Warn(err)
				continue
			}
		}
		switch res.Type {
		case msg.AgentMessage_RETURN:
			err = msgReturnHandler(conn, res)
		case msg.AgentMessage_COMMAND:
			err = msgCMDHandler(conn, res)
		case msg.AgentMessage_FILE:
			go errWrap(conn, res, msgFileHandler)
		case msg.AgentMessage_SHELL:
			err = msgShellHandler(conn, res)
		case msg.AgentMessage_STATUS:
			err = msgReqStatHandler(conn, res)
		case msg.AgentMessage_RESTART:
			conn.WriteMessage(websocket.CloseMessage, nil)
			return ErrRestart
		default:
			log.Warn("Unknown opcode ", res.Type)
		}
		if err != nil {
			utils.WithTrace(err).Warn(err)
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

	log.Info("Running pid ", syscall.Getpid())

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
				log.Warn("Trigger restart")
				return
			} else {
				utils.WithTrace(err).Error(err)
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
	rootCmd.Flags().StringVar(&netConfig, "net", "", "custom network device, seperated by ,(e.g. en0,en1)")
	rootCmd.Flags().StringVar(&diskConfig, "disk", "", "custom disk device, seperated by ,(e.g. /dev/disk1,/dev/disk2)")
	rootCmd.Flags().IntVar(&maxTime, "maxtime", 16, "max wait time when retrying")
	rootCmd.MarkFlagRequired("token")

	rootCmd.Execute()
}
