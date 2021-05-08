package main

import (
	"encoding/json"
	"net/http"
	"skynet/plugin/monitor/msg"
	"skynet/sn/utils"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jinzhu/copier"
	log "github.com/sirupsen/logrus"
)

var wsupgrader = websocket.Upgrader{
	HandshakeTimeout:  3 * time.Second,
	EnableCompression: true,
}

type AgentInfo struct {
	ID        int
	IP        string
	Name      string
	HostName  string
	LastLogin time.Time
	System    string
	Machine   string
	Conn      *websocket.Conn `json:"-"`
	Online    bool

	LastRsp   time.Time
	CPU       float64 // percent
	Mem       uint64  // bytes
	TotalMem  uint64  // bytes
	Disk      uint64  // bytes
	TotalDisk uint64  // bytes
	Load1     float64
	Latency   int64  // ms
	NetUp     uint64 // bytes/s
	NetDown   uint64 // bytes/s
	BandUp    uint64 // bytes
	BandDown  uint64 // bytes
}

var agents map[int]*AgentInfo

func init() {
	agents = make(map[int]*AgentInfo)
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

	formatErr := func() {
		msg.SendRsp(conn, -1, "Message format error")
		log.WithFields(fields()).Warn("Message format error")
	}

	for {
		_, msgRead, err := conn.ReadMessage()
		if err != nil {
			log.WithFields(defaultField).WithField("ip", ip).Info("Connection lost")
			return
		}
		var res msg.CommonMsg
		err = json.Unmarshal(msgRead, &res)
		if err != nil {
			formatErr()
			continue
		}
		if id == 0 {
			if res.Opcode == msg.OPLogin {
				var data msg.LoginMsg
				err = json.Unmarshal([]byte(res.Data), &data)
				if err != nil || len(data.UID) != 32 {
					formatErr()
					continue
				}
				if data.Token == token {
					utils.GetDB().Create(&PluginMonitorAgent{
						UID:  data.UID,
						Name: data.UID,
					})
					var rec PluginMonitorAgent
					err = utils.GetDB().Where("uid = ?", data.UID).First(&rec).Error
					if err != nil {
						log.WithFields(defaultField).Error(err)
						return
					}
					id = int(rec.ID)

					if v, exist := agents[id]; exist && v.Online {
						msg.SendRsp(conn, 2, "Agent already online")
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

					agents[id] = &AgentInfo{
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

					msg.SendRsp(conn, 0, "Login success")
					log.WithFields(fields()).Info("Login success")
				} else {
					msg.SendRsp(conn, 1, "Token invalid")
					log.WithFields(fields()).Warn("Token invalid")
				}
			} else {
				msg.SendRsp(conn, -2, "Need login")
			}
		} else {
			switch res.Opcode {
			case msg.OPInfo:
				var data msg.InfoMsg
				err = json.Unmarshal([]byte(res.Data), &data)
				if err != nil || len(data.Host) > 256 || len(data.Machine) > 32 || len(data.System) > 128 {
					formatErr()
					continue
				}
				agents[id].HostName = data.Host
				agents[id].Machine = data.Machine
				agents[id].System = data.System
				var rec PluginMonitorAgent
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
					formatErr()
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
			default:
				log.Warn("Unknown opcode ", res.Opcode)
			}
		}
	}
}
