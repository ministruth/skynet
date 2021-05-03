package main

import (
	"encoding/json"
	"net/http"
	plugins "skynet/plugin"
	"skynet/plugin/monitor/msg"
	"skynet/sn"
	"skynet/sn/utils"
	"time"

	"github.com/gin-gonic/gin"
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
	Conn      *websocket.Conn
	Online    bool
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
					defer func() { agents[id].Online = false }()

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
			default:
				log.Warn("Unknown opcode ", res.Opcode)
			}
		}
	}
}

func PageSetting(c *gin.Context, u *sn.Users) {
	sn.Skynet.Page.Render(c, plugins.SPWithIDPrefix(&Config, "setting"), "Skynet | Monitor", "Monitor", "/plugin/monitor", u, gin.H{
		"saveAPI":   "/plugin/" + Config.ID.String(),
		"renameAPI": "/plugin/" + Config.ID.String() + "/agent",
		"token":     token,
		"agents":    agents,
		"_path": append(sn.SNDefaultPath, []*sn.SNPageItem{
			{
				Name: "Plugin",
				Link: "/plugin",
			},
			{
				Name:   "Monitor",
				Active: true,
			},
		}...),
	})
}
