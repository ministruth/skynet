package main

import (
	"context"
	"net/http"
	"os"
	"path"
	"skynet/plugin/monitor/msg"
	"skynet/plugin/monitor/shared"
	"skynet/sn"
	"skynet/sn/tpl"
	"skynet/sn/utils"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
)

// xxxHandler will be run synchronized to avoid data struct race
// Thus any possible heavy workload such as RecvMsgRet should be run in go routine and
// be careful about race because no message will be processed when the handler run.

func msgInfoHandler(c *shared.Websocket, agent *shared.AgentInfo, m *msg.AgentMessage) error {
	data := m.GetInfo()
	if data == nil || len(data.Hostname) > 256 || len(data.Machine) > 32 || len(data.System) > 128 {
		return msg.ErrFormat
	}
	agent.OS = data.OS
	agent.Hostname = data.Hostname
	agent.Machine = data.Machine
	agent.System = data.System

	var rec shared.PluginMonitorAgent
	if err := tracerr.Wrap(sn.Skynet.GetDB().First(&rec, agent.ID).Error); err != nil {
		return err
	}
	rec.OS = data.OS
	rec.Hostname = data.Hostname
	rec.Machine = data.Machine
	rec.System = data.System
	if err := tracerr.Wrap(sn.Skynet.GetDB().Save(&rec).Error); err != nil {
		return err
	}

	// check version
	v1 := version.Must(version.NewVersion(shared.AgentVersion))
	v2, err := version.NewVersion(data.Version)
	if err != nil {
		return tracerr.Wrap(err)
	}
	if v1.GreaterThan(v2) {
		agent.Status = shared.AgentUpdating
		go func() {
			logf := Instance.Log().WithFields(log.Fields{
				"ip": agent.IP,
				"id": agent.ID,
			})
			defer func() {
				atomic.CompareAndSwapInt32(&agent.Status, shared.AgentUpdating, shared.AgentOnline)
			}()
			target := "agent"
			name := "agent-"
			if agent.OS == "linux" {
				name += "linux-"
				target += ".new"
			} else if agent.OS == "darwin" {
				name += "darwin-"
				target += ".new"
			} else {
				logf.Error("Platform not supported")
				return
			}
			if agent.Machine == "x86_64" {
				name += "amd64"
			} else if agent.Machine == "arm64" {
				name += "arm64"
			} else {
				logf.Error("Platform not supported")
				return
			}
			f, err := os.ReadFile(path.Join(Instance.GetPath(), name))
			if err != nil {
				utils.WithLogTrace(logf, tracerr.Wrap(err)).Error(err)
				return
			}
			id, err := msg.SendMsg(c, uuid.Nil, msg.AgentMessage_FILE, &msg.AgentMessage_File{
				File: &msg.FileMessage{
					Path:     target,
					File:     f,
					Override: true,
					Perm:     0755,
				}})
			if err != nil {
				utils.WithLogTrace(logf, err).Error(err)
				return
			}
			ret, err := msg.RecvMsgRet(id, agent.MsgRetChan, 600*time.Second)
			if err != nil {
				utils.WithLogTrace(logf, err).Error(err)
				return
			}
			if ret.Code != msg.ReturnMessage_OK {
				logf.Error(ret.Data)
				return
			}
			_, err = msg.SendMsg(c, uuid.Nil, msg.AgentMessage_RESTART, nil)
			if err != nil {
				utils.WithLogTrace(logf, err).Error(err)
				return
			}
			Instance.LogSuccess(logf, "Agent updated")
		}()
	}
	return nil
}

func msgStatHandler(c *shared.Websocket, agent *shared.AgentInfo, m *msg.AgentMessage) error {
	data := m.GetStatusRsp()
	if data == nil {
		return msg.ErrFormat
	}
	agent.CPU = data.CPU
	agent.Mem = data.Mem
	agent.TotalMem = data.TotalMem
	agent.Disk = data.Disk
	agent.TotalDisk = data.TotalDisk
	agent.Load1 = data.Load1
	tm := time.Unix(0, data.Time)
	agent.Latency = time.Since(tm).Milliseconds() / 2 // round-trip
	current := tm.Sub(agent.LastRsp).Seconds()
	agent.NetUp = uint64(float64(data.BandUp-agent.BandUp) / current)
	agent.NetDown = uint64(float64(data.BandDown-agent.BandDown) / current)
	agent.BandUp = data.BandUp
	agent.BandDown = data.BandDown
	agent.LastRsp = tm
	return nil
}

func msgCMDResHandler(c *shared.Websocket, agent *shared.AgentInfo, m *msg.AgentMessage) error {
	data := m.GetCommand()
	if data == nil || data.Type != msg.CommandMessage_RESULT {
		return msg.ErrFormat
	}
	res := data.GetRes()
	if res == nil {
		return msg.ErrFormat
	}
	if _, err := uuid.Parse(res.Cid); err != nil {
		return err
	}

	cmd, _ := agent.CMDRes.SetIfAbsent(uuid.MustParse(res.Cid), &shared.CMDRes{
		End:      false,
		DataChan: make(chan string, 60),
	})
	cmd.Data += res.Data
	cmd.Code = int(res.Code)
	cmd.End = res.End
	cmd.Complete = res.Complete
	if len(cmd.DataChan) < 60 {
		cmd.DataChan <- res.Data
	}
	if res.End {
		close(cmd.DataChan)
	}
	return nil
}

func msgReturnHandler(c *shared.Websocket, agent *shared.AgentInfo, m *msg.AgentMessage) error {
	data := m.GetReturn()
	if data == nil {
		return msg.ErrFormat
	}
	agent.MsgRetChan.Push(uuid.MustParse(m.Id), data)
	return nil
}

func msgShellHandler(c *shared.Websocket, agent *shared.AgentInfo, m *msg.AgentMessage) error {
	data := m.GetShell()
	if data == nil || data.Type != msg.ShellMessage_OUTPUT {
		return msg.ErrFormat
	}
	id, err := uuid.Parse(data.Sid)
	if err != nil {
		return msg.ErrFormat
	}
	return msg.SendShellMsg(agent.ShellConn.MustGet(id), uuid.Nil, msg.ShellMessage_OUTPUT, data.Data)
}

func ReqStat(ctx context.Context, c *shared.Websocket) {
	ticker := time.NewTicker(1 * time.Second)
	var id uint64 = 0
	for {
		select {
		case <-ticker.C:
			_, err := msg.SendMsg(c, uuid.Nil, msg.AgentMessage_STATUS, &msg.AgentMessage_StatusReq{
				StatusReq: &msg.StatusReqMessage{
					Time: time.Now().UnixNano(),
				},
			})
			if err != nil {
				utils.WithTrace(err).Warn(err)
			}
			id++
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func WSHandler(ip string, w http.ResponseWriter, r *http.Request) error {
	// current connect agent id
	var id uuid.UUID

	logf := Instance.Log().WithField("ip", ip)

	// upgrade to ws
	logf.Debug("Request agent websocket")
	conn, err := shared.NewWebsocket(&websocket.Upgrader{
		HandshakeTimeout:  3 * time.Second,
		EnableCompression: true,
	}, w, r, nil)
	if err != nil {
		return err
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
				utils.WithLogTrace(logf, err).Debug(err)
				return nil // not log connection lost
			} else {
				utils.WithLogTrace(logf, err).Warn(err)
				continue
			}
		}
		logf = logf.WithField("type", res.Type.String())
		// not login
		if id == uuid.Nil {
			if res.Type == msg.AgentMessage_LOGIN {
				data := res.GetLogin()
				if data == nil || len(data.Uid) != 32 {
					logf.Warn("Message format error")
					continue
				}
				if data.Token == token {
					var rec shared.PluginMonitorAgent
					err = tracerr.Wrap(sn.Skynet.GetDB().Where(&shared.PluginMonitorAgent{UID: data.Uid}).
						Attrs(&shared.PluginMonitorAgent{Name: data.Uid[:6]}).
						FirstOrCreate(&rec).Error)
					if err != nil {
						return err
					}
					id = rec.ID

					if v, ok := agentInstance.Get(id); ok && v.Status != shared.AgentOffline {
						msg.SendMsgRet(conn, uuid.MustParse(res.Id), msg.ReturnMessage_ONLINE, "Agent already online")
						logf.Warn("Multiple agent login")
						return nil
					}

					connTime := time.Now()
					rec.LastIP = ip
					rec.LastLogin = connTime
					if err := tracerr.Wrap(sn.Skynet.GetDB().Save(&rec).Error); err != nil {
						return err
					}

					agentInstance.Set(id, &shared.AgentInfo{
						ID:         id,
						IP:         ip,
						Name:       rec.Name,
						Conn:       conn,
						LastLogin:  connTime,
						Status:     shared.AgentOnline,
						CMDRes:     new(tpl.SafeMap[uuid.UUID, *shared.CMDRes]),
						MsgRetChan: new(shared.ChanMap),
						ShellConn:  new(tpl.SafeMap[uuid.UUID, *shared.Websocket]),
					})
					// agent cleanup
					defer func() {
						agent := agentInstance.MustGet(id)
						agent.Status = shared.AgentOffline
						agent.CMDRes.Range(func(k uuid.UUID, v *shared.CMDRes) bool {
							close(v.DataChan)
							return true
						})
						agent.MsgRetChan.Range(func(k uuid.UUID, v chan any) bool {
							close(v)
							return true
						})
						agent.ShellConn.Range(func(k uuid.UUID, v *shared.Websocket) bool {
							v.WriteMessage(websocket.CloseMessage, nil)
							v.Close()
							return true
						})
					}()

					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					go ReqStat(ctx, conn)

					msg.SendMsgRet(conn, uuid.MustParse(res.Id), msg.ReturnMessage_OK, "Login success")
					logf = logf.WithField("id", id)
					Instance.LogSuccess(logf, "Login success")
				} else {
					msg.SendMsgRet(conn, uuid.MustParse(res.Id), msg.ReturnMessage_ERROR, "Token invalid")
					logf.Warn("Token invalid")
				}
			} else {
				msg.SendMsgRet(conn, uuid.MustParse(res.Id), msg.ReturnMessage_NEED_LOGIN, "Need login")
			}
		} else {
			var err error = nil
			agent := agentInstance.MustGet(id)
			switch res.Type {
			case msg.AgentMessage_INFO:
				err = msgInfoHandler(conn, agent, res)
			case msg.AgentMessage_STATUS:
				err = msgStatHandler(conn, agent, res)
			case msg.AgentMessage_COMMAND:
				err = msgCMDResHandler(conn, agent, res)
			case msg.AgentMessage_RETURN:
				err = msgReturnHandler(conn, agent, res)
			case msg.AgentMessage_SHELL:
				err = msgShellHandler(conn, agent, res)
			default:
				logf.Warn("Unknown opcode ", res.Type)
			}
			if err != nil {
				utils.WithLogTrace(logf, err).Warn(err)
			}
		}
	}
}

// // ShellHandler handles shell websocket request.
// func ShellHandler(ip string, w http.ResponseWriter, r *http.Request) error {
// 	// current connect agent id
// 	var id uuid.UUID

// 	logf := Instance.Log().WithField("ip", ip)

// 	// upgrade to ws
// 	logf.Debug("Request shell websocket")
// 	conn, err := shared.NewWebsocket(&websocket.Upgrader{
// 		HandshakeTimeout:  3 * time.Second,
// 		EnableCompression: true,
// 	}, w, r, nil)
// 	if err != nil {
// 		return err
// 	}
// 	defer func() {
// 		conn.WriteMessage(websocket.CloseMessage, nil)
// 		conn.Close()
// 		logf.Debug("Close shell websocket")
// 	}()

// 	for {
// 		// get next msg
// 		res, msgRead, err := msg.RecvShellMsg(conn)
// 		if err != nil {
// 			if msgRead == nil {
// 				return err
// 			} else {
// 				utils.WithLogTrace(logf, err).Warn(err)
// 				continue
// 			}
// 		}
// 		if id != uuid.Nil && !agentInstance.MustGet(id).Online {
// 			logf.Info("Agent closed")
// 			msg.SendShellMsg(conn, uuid.Nil, msg.ShellMessage_ERROR, &msg.ShellMessage_Error{
// 				Error: "Agent offline",
// 			})
// 			id = uuid.Nil // not send exit to agent
// 			return nil
// 		}
// 		if id == uuid.Nil { // not connect shell
// 			if data.Type == msg.ShellMessage_CONNECT {
// 				var data msg.ShellConnectMsg
// 				if err := msg.Unmarshal(res.Data, &data); err != nil {
// 					utils.WithLogTrace(logf, err).Warn(err)
// 					continue
// 				}
// 				agent, ok := agentInstance.Get(data.ID)
// 				if !ok {
// 					msg.SendShellMsgStr(conn, uuid.Nil, msg.ShellError, "Agent not found")
// 					logf.Warn("Agent not found")
// 					continue
// 				}
// 				if !agent.Online {
// 					msg.SendShellMsgStr(conn, uuid.Nil, msg.ShellError, "Agent not online")
// 					logf.Warn("Agent not online")
// 					continue
// 				}

// 				msgID, err := msg.SendMsgByte(agent.Conn, uuid.Nil, msg.OPShell, msgRead)
// 				if err != nil {
// 					utils.WithLogTrace(logf, err).Error(err)
// 					continue
// 				}
// 				id = data.ID
// 				rec, err := msg.RecvMsgRet(msgID, agent.MsgRetChan, time.Second*3)
// 				if err != nil {
// 					utils.WithLogTrace(logf, err).Error(err)
// 					return
// 				}
// 				err = msg.SendShellMsgStr(conn, msgID, msg.ShellReturn, rec.Data)
// 				if err != nil {
// 					utils.WithLogTrace(logf, err).Error(err)
// 					return
// 				}
// 				sid, err := uuid.Parse(rec.Data)
// 				if err != nil {
// 					utils.WithLogTrace(logf, tracerr.Wrap(err)).Error(err)
// 					return
// 				}
// 				agent.ShellConn.Set(sid, conn)
// 				defer func() {
// 					_, err = msg.SendMsgByte(agentInstance.MustGet(id).Conn, uuid.Nil, msg.OPShell, msg.Marshal(msg.ShellMsg{
// 						SID:    sid,
// 						OPCode: msg.ShellDisconnect,
// 					}))
// 					if err != nil {
// 						utils.WithLogTrace(logf, err).Warn(err)
// 					}
// 					logf.Debug("Shell disconnected")
// 				}()
// 				defer agent.ShellConn.Delete(sid)
// 				logf.Info("Get shell success")
// 			} else {
// 				msg.SendShellMsgStr(conn, uuid.Nil, msg.ShellError, "Need login")
// 				logf.Warn("Need login")
// 			}
// 		} else {
// 			agent := agentInstance.MustGet(id)
// 			switch res.OPCode {
// 			case msg.ShellInput:
// 				fallthrough
// 			case msg.ShellSize:
// 				_, err := msg.SendMsgByte(agent.Conn, uuid.Nil, msg.OPShell, msgRead)
// 				if err != nil {
// 					utils.WithLogTrace(logf, err).Warn(err)
// 					msg.SendShellMsgStr(conn, uuid.Nil, msg.ShellError, err.Error())
// 				}
// 			case msg.ShellDisconnect:
// 				return
// 			default:
// 				logf.Warn("Unknown opcode ", res.OPCode)
// 			}
// 		}
// 	}
// }
