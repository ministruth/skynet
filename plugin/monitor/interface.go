package main

import (
	"errors"
	"os"
	"skynet/plugin/monitor/msg"
	"skynet/plugin/monitor/shared"
	"skynet/sn/utils"
	"time"

	plugins "skynet/plugin"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NewShared returns new shared API object.
func NewShared() shared.PluginShared {
	return &pluginShared{}
}

type pluginShared struct{}

func withAgentOnline(id int) (*shared.AgentInfo, error) {
	v, ok := agentInstance.Get(id)
	if !ok {
		return nil, shared.AgentNotExistError
	}
	if !v.Online {
		return nil, shared.AgentNotOnlineError
	}
	return v, nil
}

func (s *pluginShared) GetConfig() *plugins.PluginConfig {
	return Config
}

func (s *pluginShared) DeleteAllSetting(id int) (int64, error) {
	res := utils.GetDB().Where("agent_id = ?", id).Delete(&shared.PluginMonitorAgentSetting{})
	return res.RowsAffected, res.Error
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
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (s *pluginShared) NewSetting(id int, name string, value string) (int, error) {
	rec := shared.PluginMonitorAgentSetting{
		AgentID: int32(id),
		Name:    name,
		Value:   value,
	}
	res := utils.GetDB().Create(&rec)
	return int(rec.ID), res.Error
}

func (s *pluginShared) UpdateSetting(id int, name string, value string) error {
	return utils.GetDB().Model(&shared.PluginMonitorAgentSetting{}).Where("agent_id = ? and name = ?", id, name).Update("value", value).Error
}

func (s *pluginShared) WriteFile(id int, remotePath string, localPath string, recursive bool, override bool, perm os.FileMode, timeout time.Duration) error {
	v, err := withAgentOnline(id)
	if err != nil {
		return err
	}

	fileData, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}

	msgID, err := msg.SendMsgByte(v.Conn, uuid.Nil, msg.OPFile, msg.Marshal(msg.FileMsg{
		Path:      remotePath,
		File:      fileData,
		Recursive: recursive,
		Perm:      perm,
		Override:  override,
	}))
	if err != nil {
		return err
	}
	_, err = msg.RecvMsgRet(msgID, v.MsgRetChan, timeout)
	if err != nil {
		return err
	}
	return nil
}

func (s *pluginShared) KillCMD(id int, cid uuid.UUID) error {
	v, err := withAgentOnline(id)
	if err != nil {
		return err
	}

	_, err = msg.SendMsgByte(v.Conn, uuid.Nil, msg.OPCMDKill, msg.Marshal(msg.CMDKillMsg{
		CID: cid,
	}))

	return err
}

func (s *pluginShared) GetCMDRes(id int, cid uuid.UUID) (*shared.CMDRes, error) {
	v, err := withAgentOnline(id)
	if err != nil {
		return nil, err
	}
	if v.CMDRes == nil || !v.CMDRes.Has(cid) {
		return nil, shared.CMDIDNotFoundError
	}

	return v.CMDRes.MustGet(cid), nil
}

func (s *pluginShared) RunCMDSync(id int, cmd string, timeout time.Duration) (uuid.UUID, string, error) {
	v, err := withAgentOnline(id)
	if err != nil {
		return uuid.Nil, "", err
	}

	cid := uuid.New()
	v.CMDRes.SetIfAbsent(cid, &shared.CMDRes{
		End:      false,
		DataChan: make(chan string, 1),
	})

	msgID := uuid.New()
	_, err = msg.SendMsgByte(v.Conn, msgID, msg.OPCMD, msg.Marshal(msg.CMDMsg{
		CID:     cid,
		Payload: cmd,
		Sync:    true,
	}))
	if err != nil {
		return uuid.Nil, "", err
	}

	ret, err := msg.RecvMsgRet(msgID, v.MsgRetChan, timeout)
	if err == shared.OPTimeoutError {
		s.KillCMD(id, cid)
		return uuid.Nil, "", err
	} else if err != nil {
		return uuid.Nil, "", err
	}

	var data msg.CMDResMsg
	err = msg.Unmarshal([]byte(ret.Data), &data)
	if err != nil {
		return uuid.Nil, "", err
	}
	res := v.CMDRes.MustGet(data.CID)
	res.Data += data.Data
	res.Code = data.Code
	res.End = data.End
	res.Complete = data.Complete
	res.DataChan <- data.Data
	if data.End {
		close(res.DataChan)
	}
	return cid, data.Data, nil
}

func (s *pluginShared) RunCMDAsync(id int, cmd string) (uuid.UUID, chan string, error) {
	v, err := withAgentOnline(id)
	if err != nil {
		return uuid.Nil, nil, err
	}

	cid := uuid.New()

	v.CMDRes.SetIfAbsent(cid, &shared.CMDRes{
		End:      false,
		DataChan: make(chan string, 60),
	})

	_, err = msg.SendMsgByte(v.Conn, uuid.Nil, msg.OPCMD, msg.Marshal(msg.CMDMsg{
		CID:     cid,
		Payload: cmd,
		Sync:    false,
	}))
	if err != nil {
		return uuid.Nil, nil, err
	}

	return cid, v.CMDRes.MustGet(cid).DataChan, nil
}

func (s *pluginShared) GetAgents() map[int]*shared.AgentInfo {
	return agentInstance.Map()
}
