package main

import (
	"encoding/json"
	"net/http"
	"skynet/plugin/monitor/msg"
	"skynet/plugin/monitor/shared"
	"skynet/sn/utils"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jinzhu/copier"
	log "github.com/sirupsen/logrus"
)

var wsupgrader = websocket.Upgrader{
	HandshakeTimeout:  3 * time.Second,
	EnableCompression: true,
}

var recvChan = make(map[uuid.UUID]chan msg.RetMsg)

var agents map[int]*shared.AgentInfo

type AgentSort []*shared.AgentInfo

func (s AgentSort) Len() int           { return len(s) }
func (s AgentSort) Less(i, j int) bool { return s[i].ID < s[j].ID }
func (s AgentSort) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func init() {
	agents = make(map[int]*shared.AgentInfo)
}

func WSHandler(ip string, w http.ResponseWriter, r *http.Request) {
	conn, err := wsupgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(err)
		return
	}
	defer func() {
		conn.WriteMessage(websocket.CloseMessage, nil)
		conn.Close()
	}()

	var id int = 0

	fields := func() log.Fields {
		ret := make(log.Fields)
		copier.CopyWithOption(&ret, &defaultField, copier.Option{DeepCopy: true})
		ret["ip"] = ip
		ret["id"] = id
		return ret
	}

	formatErr := func(id uuid.UUID) {
		msg.SendRsp(conn, id, -1, "Message format error")
		log.WithFields(fields()).Warn("Message format error")
	}

	for {
		res, msgRead, err := msg.Recv(conn)
		if err != nil {
			if msgRead != nil {
				formatErr(res.ID)
				continue
			} else {
				log.WithFields(defaultField).WithField("ip", ip).Info("Connection lost")
				return
			}
		}
		if id == 0 {
			if res.Opcode == msg.OPLogin {
				var data msg.LoginMsg
				err = json.Unmarshal([]byte(res.Data), &data)
				if err != nil || len(data.UID) != 32 {
					formatErr(res.ID)
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
						log.WithFields(defaultField).Error(err)
						return
					}
					id = int(rec.ID)

					if v, exist := agents[id]; exist && v.Online {
						msg.SendRsp(conn, res.ID, 2, "Agent already online")
						log.WithFields(fields()).Warn("Multiple agent login")
						return
					}

					connTime := time.Now()
					rec.LastIP = ip
					rec.LastLogin = connTime
					err = utils.GetDB().Save(&rec).Error
					if err != nil {
						log.WithFields(defaultField).Error(err)
						return
					}

					agents[id] = &shared.AgentInfo{
						ID:        id,
						IP:        ip,
						Name:      rec.Name,
						Conn:      conn,
						LastLogin: connTime,
						Online:    true,
					}
					defer func() {
						agents[id].Online = false
						agents[id].Conn = nil
					}()

					msg.SendRsp(conn, res.ID, 0, "Login success")
					log.WithFields(fields()).Info("Login success")
				} else {
					msg.SendRsp(conn, res.ID, 1, "Token invalid")
					log.WithFields(fields()).Warn("Token invalid")
				}
			} else {
				msg.SendRsp(conn, res.ID, -2, "Need login")
			}
		} else {
			switch res.Opcode {
			case msg.OPInfo:
				var data msg.InfoMsg
				err = json.Unmarshal([]byte(res.Data), &data)
				if err != nil || len(data.Host) > 256 || len(data.Machine) > 32 || len(data.System) > 128 {
					formatErr(res.ID)
					continue
				}
				agents[id].HostName = data.Host
				agents[id].Machine = data.Machine
				agents[id].System = data.System
				var rec shared.PluginMonitorAgent
				err = utils.GetDB().First(&rec, id).Error
				if err != nil {
					log.WithFields(defaultField).Error(err)
					return
				}
				rec.Hostname = data.Host
				rec.Machine = data.Machine
				rec.System = data.System
				err = utils.GetDB().Save(&rec).Error
				if err != nil {
					log.WithFields(defaultField).Error(err)
					return
				}
			case msg.OPStat:
				var data msg.StatMsg
				err = json.Unmarshal([]byte(res.Data), &data)
				if err != nil {
					formatErr(res.ID)
					continue
				}
				agents[id].CPU = data.CPU
				agents[id].Mem = data.Mem
				agents[id].TotalMem = data.TotalMem
				agents[id].Disk = data.Disk
				agents[id].TotalDisk = data.TotalDisk
				agents[id].Load1 = data.Load1
				agents[id].Latency = time.Since(data.Time).Milliseconds()
				current := time.Since(agents[id].LastRsp).Seconds()
				agents[id].NetUp = uint64(float64(data.BandUp-agents[id].BandUp) / current)
				agents[id].NetDown = uint64(float64(data.BandDown-agents[id].BandDown) / current)
				agents[id].BandUp = data.BandUp
				agents[id].BandDown = data.BandDown
				agents[id].LastRsp = time.Now()
			case msg.OPCMDRes:
				var data msg.CMDResMsg
				err = json.Unmarshal([]byte(res.Data), &data)
				if err != nil {
					formatErr(res.ID)
					continue
				}
				agents[id].CMDRes[data.UID].Data += data.Data
				agents[id].CMDRes[data.UID].Code = data.Code
				agents[id].CMDRes[data.UID].End = data.End
				agents[id].CMDRes[data.UID].Complete = data.Complete
				agents[id].CMDRes[data.UID].DataChan <- data.Data
				if data.End {
					close(agents[id].CMDRes[data.UID].DataChan)
				}
			case msg.OPRet:
				var data msg.RetMsg
				err = json.Unmarshal([]byte(res.Data), &data)
				if err != nil {
					formatErr(res.ID)
					continue
				}
				ch, ok := recvChan[res.ID]
				if !ok {
					log.Warn("Error response: " + res.Data)
				} else {
					ch <- data
				}
			default:
				log.Warn("Unknown opcode ", res.Opcode)
			}
		}
	}
}
