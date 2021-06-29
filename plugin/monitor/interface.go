package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	plugins "skynet/plugin"
	"skynet/plugin/monitor/msg"
	"skynet/plugin/monitor/shared"
	"skynet/sn/utils"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

func NewShared() shared.PluginShared {
	return &pluginShared{}
}

type pluginShared struct{}

func checkAgent(id int) (*shared.AgentInfo, error) {
	v, ok := agents[id]
	if !ok {
		return nil, shared.AgentNotExistError
	}
	if !v.Online {
		return nil, shared.AgentNotOnlineError
	}
	return v, nil
}

func (s *pluginShared) DeleteAllSetting(id int) error {
	return utils.GetDB().Where("agent_id = ?", id).Delete(&shared.PluginMonitorAgentSetting{}).Error
}

func (s *pluginShared) DeleteSetting(id int, name string) error {
	return utils.GetDB().Where("agent_id = ? and name = ?", id, name).Delete(&shared.PluginMonitorAgentSetting{}).Error
}

func (s *pluginShared) GetAllSetting(id int) ([]*shared.PluginMonitorAgentSetting, error) {
	var rec []*shared.PluginMonitorAgentSetting
	err := utils.GetDB().Where("agent_id = ?", id).Find(&rec).Error
	if err != nil {
		return nil, err
	}
	return rec, nil
}

func (s *pluginShared) GetSetting(id int, name string) (*shared.PluginMonitorAgentSetting, error) {
	var rec shared.PluginMonitorAgentSetting
	err := utils.GetDB().Where("agent_id = ? and name = ?", id, name).First(&rec).Error
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (s *pluginShared) NewSetting(id int, name string, value string) error {
	return utils.GetDB().Create(&shared.PluginMonitorAgentSetting{
		AgentID: int32(id),
		Name:    name,
		Value:   value,
	}).Error
}

func (s *pluginShared) UpdateSetting(id int, name string, value string) error {
	return utils.GetDB().Model(&shared.PluginMonitorAgentSetting{}).Where("agent_id = ? and name = ?", id, name).Update("value", value).Error
}

func (s *pluginShared) GetPluginPath(c *plugins.PluginConfig, p string) string {
	return "plugin/" + c.ID.String() + "/" + p
}

func (s *pluginShared) WriteFile(id int, path string, file string, recursive bool, override bool, perm os.FileMode, timeout time.Duration) error {
	v, ok := agents[id]
	if !ok {
		return shared.AgentNotExistError
	}
	if !v.Online {
		return shared.AgentNotOnlineError
	}

	fileData, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	d, err := json.Marshal(msg.FileMsg{
		Path:      path,
		File:      fileData,
		Recursive: recursive,
		Perm:      perm,
		Override:  override,
	})
	if err != nil {
		log.Fatal(err)
	}
	mid, err := msg.SendReq(v.Conn, msg.OPFile, string(d))
	if err != nil {
		return err
	}
	_, err = msg.RecvRsp(mid, recvChan, timeout)
	if err != nil {
		return err
	}
	return nil
}

func (s *pluginShared) KillCMD(id int, uid uuid.UUID, isReturn bool) error {
	v, err := checkAgent(id)
	if err != nil {
		return err
	}

	d, err := json.Marshal(msg.CMDKillMsg{
		UID:    uid,
		Return: isReturn,
	})
	if err != nil {
		log.Fatal(err)
	}
	mid, err := msg.SendReq(v.Conn, msg.OPCMDKill, string(d))
	if err != nil {
		return err
	}
	if !isReturn {
		return nil
	}
	_, err = msg.RecvRsp(mid, recvChan, time.Second*3)
	if err != nil {
		return err
	}
	return nil
}

func (s *pluginShared) GetCMDRes(id int, uid uuid.UUID) (*shared.CMDRes, error) {
	v, err := checkAgent(id)
	if err != nil {
		return nil, err
	}
	if v.CMDRes == nil || v.CMDRes[uid] == nil {
		return nil, shared.UIDNotFoundError
	}

	return v.CMDRes[uid], nil
}

func (s *pluginShared) RunCMDSync(id int, cmd string, timeout time.Duration) (uuid.UUID, string, error) {
	v, err := checkAgent(id)
	if err != nil {
		return uuid.Nil, "", err
	}

	uid := uuid.New()
	if agents[id].CMDRes == nil {
		agents[id].CMDRes = make(map[uuid.UUID]*shared.CMDRes)
	}
	if agents[id].CMDRes[uid] == nil {
		agents[id].CMDRes[uid] = &shared.CMDRes{
			End:      false,
			DataChan: make(chan string, 1),
		}
	}

	mid := uuid.New()
	d, err := json.Marshal(msg.CMDMsg{
		UID:     uid,
		Payload: cmd,
		Sync:    true,
	})
	if err != nil {
		log.Fatal(err)
	}
	err = msg.SendReqWithID(v.Conn, mid, msg.OPCMD, string(d))
	if err != nil {
		return uuid.Nil, "", err
	}

	ret, err := msg.RecvRsp(mid, recvChan, timeout)
	if err == shared.OPTimeoutError {
		s.KillCMD(id, uid, false)
		return uuid.Nil, "", err
	} else if err != nil {
		return uuid.Nil, "", err
	}

	var data msg.CMDResMsg
	err = json.Unmarshal([]byte(ret.Data), &data)
	if err != nil {
		return uuid.Nil, "", err
	}
	agents[id].CMDRes[data.UID].Data += data.Data
	agents[id].CMDRes[data.UID].Code = data.Code
	agents[id].CMDRes[data.UID].End = data.End
	agents[id].CMDRes[data.UID].Complete = data.Complete
	agents[id].CMDRes[data.UID].DataChan <- data.Data
	if data.End {
		close(agents[id].CMDRes[data.UID].DataChan)
	}
	return uid, data.Data, nil
}

func (s *pluginShared) RunCMDAsync(id int, cmd string) (uuid.UUID, chan string, error) {
	v, err := checkAgent(id)
	if err != nil {
		return uuid.Nil, nil, err
	}

	uid := uuid.New()

	if agents[id].CMDRes == nil {
		agents[id].CMDRes = make(map[uuid.UUID]*shared.CMDRes)
	}
	if agents[id].CMDRes[uid] == nil {
		agents[id].CMDRes[uid] = &shared.CMDRes{
			End:      false,
			DataChan: make(chan string, 60),
		}
	}

	d, err := json.Marshal(msg.CMDMsg{
		UID:     uid,
		Payload: cmd,
		Sync:    false,
	})
	if err != nil {
		log.Fatal(err)
	}
	_, err = msg.SendReq(v.Conn, msg.OPCMD, string(d))
	if err != nil {
		return uuid.Nil, nil, err
	}

	return uid, agents[id].CMDRes[uid].DataChan, nil
}

func (s *pluginShared) GetAgents() map[int]*shared.AgentInfo {
	return agents
}
