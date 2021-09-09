package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path"
	"skynet/plugin/monitor/msg"
	"skynet/plugin/monitor/shared"
	"skynet/sn/utils"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
)

var agentInstance shared.AgentMap

// xxxHandler will be run synchronized to avoid data struct race
// Thus any possible heavy workload such as RecvMsgRet should be run in go routine and
// be careful about race because no message will be processed when the handler run.

func msgInfoHandler(c *shared.Websocket, agent *shared.AgentInfo, m *msg.CommonMsg) error {
	var data msg.InfoMsg
	err := msg.Unmarshal(m.Data, &data)
	if err != nil || len(data.Host) > 256 || len(data.Machine) > 32 || len(data.System) > 128 {
		return errors.New("Message format error: " + err.Error())
	}
	agent.HostName = data.Host
	agent.Machine = data.Machine
	agent.System = data.System

	var rec shared.PluginMonitorAgent
	if err = utils.GetDB().First(&rec, agent.ID).Error; err != nil {
		return err
	}
	rec.Hostname = data.Host
	rec.Machine = data.Machine
	rec.System = data.System
	err = utils.GetDB().Save(&rec).Error
	if err != nil {
		return err
	}

	// check version
	v1 := version.Must(version.NewVersion(shared.AgentVersion))
	v2, err := version.NewVersion(data.Version)
	if err != nil {
		return err
	}
	if v1.GreaterThan(v2) {
		go func() {
			logf := log.WithFields(defaultField).WithFields(log.Fields{
				"ip": agent.IP,
				"id": agent.ID,
			})
			agent.Updating = true
			defer func() {
				agent.Updating = false
			}()
			name := "agent-"
			if strings.Contains(strings.ToLower(agent.System), "linux") {
				name += "linux-"
			} else {
				logf.Error("Platform not supported")
				return
			}
			if agent.Machine == "x86_64" {
				name += "amd64"
			} else {
				logf.Error("Platform not supported")
				return
			}
			f, err := os.ReadFile(path.Join(Instance.Path, name))
			if err != nil {
				logf.Error(err)
				return
			}
			id, err := msg.SendMsgByte(c, uuid.Nil, msg.OPFile, msg.Marshal(msg.FileMsg{
				Path:     name,
				File:     f,
				Override: true,
				Perm:     0755,
			}))
			if err != nil {
				logf.Error(err)
				return
			}
			ret, err := msg.RecvMsgRet(id, agent.MsgRetChan, 3600*time.Second)
			if err != nil {
				logf.Error(err)
				return
			}
			if ret.Code != 0 {
				logf.Error(ret.Data)
				return
			}
			_, err = msg.SendMsgByte(c, uuid.Nil, msg.OPRestart, []byte{})
			if err != nil {
				logf.Error(err)
				return
			}
		}()
	}
	return nil
}

func msgStatHandler(c *shared.Websocket, agent *shared.AgentInfo, m *msg.CommonMsg) error {
	var data msg.StatMsg
	if err := msg.Unmarshal(m.Data, &data); err != nil {
		return err
	}
	agent.CPU = data.CPU
	agent.Mem = data.Mem
	agent.TotalMem = data.TotalMem
	agent.Disk = data.Disk
	agent.TotalDisk = data.TotalDisk
	agent.Load1 = data.Load1
	agent.Latency = time.Since(data.Time).Milliseconds() / 2 // round-trip
	current := time.Since(agent.LastRsp).Seconds()
	agent.NetUp = uint64(float64(data.BandUp-agent.BandUp) / current)
	agent.NetDown = uint64(float64(data.BandDown-agent.BandDown) / current)
	agent.BandUp = data.BandUp
	agent.BandDown = data.BandDown
	agent.LastRsp = time.Now()
	return nil
}

func msgCMDResHandler(c *shared.Websocket, agent *shared.AgentInfo, m *msg.CommonMsg) error {
	var data msg.CMDResMsg
	if err := msg.Unmarshal(m.Data, &data); err != nil {
		return err
	}
	cmd, _ := agent.CMDRes.SetIfAbsent(data.CID, &shared.CMDRes{
		End:      false,
		DataChan: make(chan string, 60),
	})
	cmd.Data += data.Data
	cmd.Code = data.Code
	cmd.End = data.End
	cmd.Complete = data.Complete
	if len(cmd.DataChan) < 60 {
		cmd.DataChan <- data.Data
	}
	if data.End {
		close(cmd.DataChan)
	}
	return nil
}

func msgReturnHandler(c *shared.Websocket, agent *shared.AgentInfo, m *msg.CommonMsg) error {
	var data msg.RetMsg
	if err := msg.Unmarshal(m.Data, &data); err != nil {
		return err
	}
	agent.MsgRetChan.SetIfAbsent(m.ID)
	agent.MsgRetChan.Push(m.ID, &data)
	return nil
}

func msgShellHandler(c *shared.Websocket, agent *shared.AgentInfo, m *msg.CommonMsg) error {
	var data msg.ShellMsg
	err := msg.Unmarshal(m.Data, &data)
	if err != nil || data.OPCode != msg.ShellOutput {
		return errors.New("Message format error: " + err.Error())
	}
	return msg.SendShellMsgByte(agent.ShellConn.MustGet(data.SID), uuid.Nil, msg.ShellOutput, data.Data)
}

func ReqStat(ctx context.Context, c *shared.Websocket) {
	ticker := time.NewTicker(1 * time.Second)
	var id uint64 = 0
	for {
		select {
		case <-ticker.C:
			_, err := msg.SendMsgStr(c, uuid.Nil, msg.OPReqStat, strconv.FormatInt(time.Now().UnixNano(), 10))
			if err != nil {
				log.Warn(err)
			}
			id++
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func WSHandler(ip string, w http.ResponseWriter, r *http.Request) {
	// current connect agent id
	var id int = 0

	logf := log.WithFields(defaultField).WithFields(log.Fields{
		"ip": ip,
		"id": id,
	})

	// upgrade to ws
	logf.Debug("Request agent websocket")
	conn, err := shared.NewWebsocket(&websocket.Upgrader{
		HandshakeTimeout:  3 * time.Second,
		EnableCompression: true,
	}, w, r, nil)
	if err != nil {
		logf.Error(err)
		return
	}
	defer func() {
		conn.WriteMessage(websocket.CloseMessage, nil)
		conn.Close()
		logf.Debug("Close agent websocket")
	}()

	for {
		res, msgRead, err := msg.RecvMsg(conn)
		if err != nil {
			if msgRead == nil {
				logf.Error("Connection lost: ", ip)
				return
			} else {
				logf.Warn(err)
				continue
			}
		}
		if id == 0 {
			if res.OPCode == msg.OPLogin {
				var data msg.LoginMsg
				err = msg.Unmarshal(res.Data, &data)
				if err != nil || len(data.UID) != 32 {
					logf.Warn("Message format error: ", err)
					continue
				}
				if data.Token == token {
					utils.GetDB().Create(&shared.PluginMonitorAgent{
						UID:  data.UID,
						Name: data.UID[:6],
					})
					var rec shared.PluginMonitorAgent
					err = utils.GetDB().Where("uid = ?", data.UID).First(&rec).Error
					if err != nil {
						logf.Error(err)
						return
					}
					id = int(rec.ID)

					if v, ok := agentInstance.Get(id); ok && v.Online {
						msg.SendMsgRet(conn, res.ID, 2, "Agent already online")
						logf.Warn("Multiple agent login")
						return
					}

					connTime := time.Now()
					rec.LastIP = ip
					rec.LastLogin = connTime
					err = utils.GetDB().Save(&rec).Error
					if err != nil {
						logf.Error(err)
						return
					}

					agentInstance.Set(id, &shared.AgentInfo{
						ID:         id,
						IP:         ip,
						Name:       rec.Name,
						Conn:       conn,
						LastLogin:  connTime,
						Updating:   false,
						Online:     true,
						CMDRes:     &shared.CMDResMap{},
						MsgRetChan: &shared.ChanMap{},
						ShellConn:  &shared.SocketMap{},
					})
					// agent cleanup
					defer func() {
						agent := agentInstance.MustGet(id)
						agent.Online = false
						agent.Updating = false
						agent.MsgRetChan.Range(func(k uuid.UUID, v interface{}) bool {
							close(v.(chan interface{}))
							return true
						})
					}()

					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					go ReqStat(ctx, conn)

					msg.SendMsgRet(conn, res.ID, 0, "Login success")
					logf.Info("Login success")
				} else {
					msg.SendMsgRet(conn, res.ID, 1, "Token invalid")
					logf.Warn("Token invalid")
				}
			} else {
				msg.SendMsgRet(conn, res.ID, -2, "Need login")
			}
		} else {
			var err error = nil
			agent := agentInstance.MustGet(id)
			switch res.OPCode {
			case msg.OPInfo:
				err = msgInfoHandler(conn, agent, res)
			case msg.OPStat:
				err = msgStatHandler(conn, agent, res)
			case msg.OPCMDRes:
				err = msgCMDResHandler(conn, agent, res)
			case msg.OPRet:
				err = msgReturnHandler(conn, agent, res)
			case msg.OPShell:
				err = msgShellHandler(conn, agent, res)
			default:
				logf.Warn("Unknown opcode ", res.OPCode)
			}
			if err != nil {
				logf.Warn(err)
			}
		}
	}
}

// ShellHandler handles shell websocket request.
func ShellHandler(ip string, w http.ResponseWriter, r *http.Request) {
	// current connect agent id
	var id int = 0

	logf := log.WithFields(defaultField).WithFields(log.Fields{
		"ip": ip,
		"id": id,
	})

	// upgrade to ws
	logf.Debug("Request shell websocket")
	conn, err := shared.NewWebsocket(&websocket.Upgrader{
		HandshakeTimeout:  3 * time.Second,
		EnableCompression: true,
	}, w, r, nil)
	if err != nil {
		logf.Error(err)
		return
	}
	defer func() {
		conn.WriteMessage(websocket.CloseMessage, nil)
		conn.Close()
		logf.Debug("Close shell websocket")
	}()

	for {
		// get next msg
		res, msgRead, err := msg.RecvShellMsg(conn)
		if err != nil {
			if msgRead == nil {
				logf.Info("Shell closed")
				return
			} else {
				logf.Warn(err)
				continue
			}
		}
		if id != 0 && !agentInstance.MustGet(id).Online {
			logf.Info("Agent closed")
			msg.SendShellMsgStr(conn, uuid.Nil, msg.ShellError, "Agent offline")
			id = 0 // not send exit to agent
			return
		}
		if id == 0 { // not connect shell
			if res.OPCode == msg.ShellConnect {
				var data msg.ShellConnectMsg
				if err := msg.Unmarshal(res.Data, &data); err != nil {
					logf.Warn(err)
					continue
				}
				agent, ok := agentInstance.Get(data.ID)
				if !ok {
					msg.SendShellMsgStr(conn, uuid.Nil, msg.ShellError, "Agent not found")
					logf.Warn("Agent not found")
					continue
				}
				if !agent.Online {
					msg.SendShellMsgStr(conn, uuid.Nil, msg.ShellError, "Agent not online")
					logf.Warn("Agent not online")
					continue
				}

				msgID, err := msg.SendMsgByte(agent.Conn, uuid.Nil, msg.OPShell, msgRead)
				if err != nil {
					logf.Error(err)
					continue
				}
				id = data.ID
				rec, err := msg.RecvMsgRet(msgID, agent.MsgRetChan, time.Second*3)
				if err != nil {
					logf.Error(err)
					return
				}
				err = msg.SendShellMsgStr(conn, msgID, msg.ShellReturn, rec.Data)
				if err != nil {
					logf.Error(err)
					return
				}
				sid, err := uuid.Parse(rec.Data)
				if err != nil {
					logf.Error(err)
					return
				}
				agent.ShellConn.Set(sid, conn)
				defer func() {
					_, err = msg.SendMsgByte(agentInstance.MustGet(id).Conn, uuid.Nil, msg.OPShell, msg.Marshal(msg.ShellMsg{
						SID:    sid,
						OPCode: msg.ShellDisconnect,
					}))
					if err != nil {
						logf.Warn(err)
					}
					logf.Debug("Shell disconnected")
				}()
				defer agent.ShellConn.Delete(sid)
				logf.Info("Get shell success")
			} else {
				msg.SendShellMsgStr(conn, uuid.Nil, msg.ShellError, "Need login")
				logf.Warn("Need login")
			}
		} else {
			agent := agentInstance.MustGet(id)
			switch res.OPCode {
			case msg.ShellInput:
				fallthrough
			case msg.ShellSize:
				_, err := msg.SendMsgByte(agent.Conn, uuid.Nil, msg.OPShell, msgRead)
				if err != nil {
					logf.Warn(err)
					msg.SendShellMsgStr(conn, uuid.Nil, msg.ShellError, err.Error())
				}
			case msg.ShellDisconnect:
				return
			default:
				logf.Warn("Unknown opcode ", res.OPCode)
			}
		}
	}
}
