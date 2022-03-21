package main

import (
	"os"
	"skynet/plugin/monitor/msg"
	"skynet/plugin/monitor/shared"
	"skynet/sn"
	"skynet/sn/impl"
	"time"

	plugins "skynet/plugin"

	"github.com/google/uuid"
	"github.com/ztrue/tracerr"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// NewShared returns new shared API object.
func NewShared() shared.PluginShared {
	return &pluginShared{
		tx:      sn.Skynet.GetDB(),
		agent:   impl.NewORM[shared.PluginMonitorAgent](nil),
		setting: impl.NewORM[shared.PluginMonitorAgentSetting](nil),
	}
}

type pluginShared struct {
	tx      *gorm.DB
	agent   *impl.ORM[shared.PluginMonitorAgent]
	setting *impl.ORM[shared.PluginMonitorAgentSetting]
}

func withAgentOnline(id uuid.UUID) (*shared.AgentInfo, error) {
	v, ok := agentInstance.Get(id)
	if !ok {
		return nil, shared.ErrAgentNotExist
	}
	if v.Status == shared.AgentOffline {
		return nil, shared.ErrAgentNotOnline
	}
	return v, nil
}

func (s *pluginShared) WithTx(tx *gorm.DB) shared.PluginShared {
	return &pluginShared{
		tx:      tx,
		agent:   impl.NewORM[shared.PluginMonitorAgent](tx),
		setting: impl.NewORM[shared.PluginMonitorAgentSetting](tx),
	}
}

func (s *pluginShared) GetInstance() *plugins.PluginInfo {
	return Instance
}

func (s *pluginShared) DeleteAllSetting(id uuid.UUID) (int64, error) {
	return s.setting.Impl.Where("agent_id = ?", id).Delete()
}

func (s *pluginShared) DeleteSetting(id uuid.UUID, name string) (bool, error) {
	row, err := s.setting.Impl.Where("agent_id = ? and name = ?", id, name).Delete()
	return row == 1, err
}

func (s *pluginShared) GetAllSetting(id uuid.UUID) ([]*shared.PluginMonitorAgentSetting, error) {
	return s.setting.Impl.Where("agent_id = ?", id).Find()
}

func (s *pluginShared) GetSetting(id uuid.UUID, name string) (*shared.PluginMonitorAgentSetting, error) {
	return s.setting.Impl.Where("agent_id = ? and name = ?", id, name).Take()
}

func (s *pluginShared) NewSetting(id uuid.UUID, name string, value string) (rec *shared.PluginMonitorAgentSetting, err error) {
	rec = &shared.PluginMonitorAgentSetting{
		AgentID: id,
		Name:    name,
		Value:   value,
	}
	err = s.setting.Impl.Create(rec)
	return
}

func (s *pluginShared) UpdateSetting(id uuid.UUID, name string, value string) error {
	return s.setting.Impl.Where("agent_id = ? and name = ?", id, name).Update("value", value)
}

func (s *pluginShared) GetAgent(id uuid.UUID) (*shared.PluginMonitorAgent, error) {
	return s.agent.Get(id)
}

func (s *pluginShared) GetAllAgent(cond *sn.SNCondition) ([]*shared.PluginMonitorAgent, error) {
	return s.agent.GetAll(cond)
}

func (s *pluginShared) WriteFile(id uuid.UUID, remotePath string, localPath string,
	recursive bool, override bool, perm os.FileMode, timeout time.Duration) error {
	v, err := withAgentOnline(id)
	if err != nil {
		return err
	}

	fileData, err := os.ReadFile(localPath)
	if err != nil {
		return tracerr.Wrap(err)
	}

	msgID, err := msg.SendMsg(v.Conn, uuid.Nil, msg.AgentMessage_FILE, &msg.AgentMessage_File{
		File: &msg.FileMessage{
			Path:      remotePath,
			File:      fileData,
			Recursive: recursive,
			Perm:      int32(perm),
			Override:  override,
		}})
	if err != nil {
		return err
	}
	_, err = msg.RecvMsgRet(msgID, v.MsgRetChan, timeout)
	if err != nil {
		return err
	}
	return nil
}

func (s *pluginShared) KillCMD(id uuid.UUID, cid uuid.UUID) error {
	v, err := withAgentOnline(id)
	if err != nil {
		return err
	}

	_, err = msg.SendMsg(v.Conn, uuid.Nil, msg.AgentMessage_COMMAND, &msg.AgentMessage_Command{
		Command: &msg.CommandMessage{
			Type: msg.CommandMessage_KILL,
			Data: &msg.CommandMessage_Kill{
				Kill: &msg.CMDKillMessage{
					Cid: cid.String(),
				},
			},
		},
	})

	return err
}

func (s *pluginShared) GetCMDRes(id uuid.UUID, cid uuid.UUID) (*shared.CMDRes, error) {
	v, err := withAgentOnline(id)
	if err != nil {
		return nil, err
	}
	if v.CMDRes == nil || !v.CMDRes.Has(cid) {
		return nil, shared.ErrCMDIDNotFound
	}

	return v.CMDRes.MustGet(cid), nil
}

func (s *pluginShared) RunCMDSync(id uuid.UUID, cmd string, timeout time.Duration) (uuid.UUID, string, error) {
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
	_, err = msg.SendMsg(v.Conn, uuid.Nil, msg.AgentMessage_COMMAND, &msg.AgentMessage_Command{
		Command: &msg.CommandMessage{
			Type: msg.CommandMessage_RUN,
			Data: &msg.CommandMessage_Run{
				Run: &msg.CMDRunMessage{
					Cid:     cid.String(),
					Payload: cmd,
					Sync:    true,
				},
			},
		},
	})
	if err != nil {
		return uuid.Nil, "", err
	}

	ret, err := msg.RecvMsgRet(msgID, v.MsgRetChan, timeout)
	if err == shared.ErrOPTimeout {
		s.KillCMD(id, cid)
		return uuid.Nil, "", err
	} else if err != nil {
		return uuid.Nil, "", err
	}

	var data msg.CMDResMessage
	if err := proto.Unmarshal([]byte(ret.Data), &data); err != nil {
		return uuid.Nil, "", err
	}
	dataid, err := uuid.Parse(data.Cid)
	if err != nil {
		return uuid.Nil, "", err
	}
	res := v.CMDRes.MustGet(dataid)
	res.Data += data.Data
	res.Code = int(data.Code)
	res.End = data.End
	res.Complete = data.Complete
	res.DataChan <- data.Data
	if data.End {
		close(res.DataChan)
	}
	return cid, data.Data, nil
}

func (s *pluginShared) RunCMDAsync(id uuid.UUID, cmd string) (uuid.UUID, chan string, error) {
	v, err := withAgentOnline(id)
	if err != nil {
		return uuid.Nil, nil, err
	}

	cid := uuid.New()
	v.CMDRes.SetIfAbsent(cid, &shared.CMDRes{
		End:      false,
		DataChan: make(chan string, 60),
	})

	_, err = msg.SendMsg(v.Conn, uuid.Nil, msg.AgentMessage_COMMAND, &msg.AgentMessage_Command{
		Command: &msg.CommandMessage{
			Type: msg.CommandMessage_RUN,
			Data: &msg.CommandMessage_Run{
				Run: &msg.CMDRunMessage{
					Cid:     cid.String(),
					Payload: cmd,
					Sync:    false,
				},
			},
		},
	})
	if err != nil {
		return uuid.Nil, nil, err
	}

	return cid, v.CMDRes.MustGet(cid).DataChan, nil
}

func (s *pluginShared) GetAgentInstance() map[uuid.UUID]*shared.AgentInfo {
	return agentInstance.Map()
}
