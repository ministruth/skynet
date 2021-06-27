package main

import (
	"encoding/json"
	"skynet/plugin/monitor/msg"
	"skynet/plugin/monitor/shared"

	"github.com/google/uuid"
)

func NewShared() shared.PluginShared {
	return &pluginShared{}
}

type pluginShared struct{}

func (s *pluginShared) KillCMD(id int, uid uuid.UUID) error {
	v, ok := agents[id]
	if !ok {
		return shared.AgentNotExistError
	}
	if !v.Online {
		return shared.AgentNotOnlineError
	}

	err := msg.SendReq(v.Conn, msg.OPCMDKill, uid.String())
	if err != nil {
		return err
	}

	return nil
}

func (s *pluginShared) GetCMDRes(id int, uid uuid.UUID) (*shared.CMDRes, error) {
	v, ok := agents[id]
	if !ok {
		return nil, shared.AgentNotExistError
	}
	if !v.Online {
		return nil, shared.AgentNotOnlineError
	}
	if v.CMDRes == nil || v.CMDRes[uid] == nil {
		return nil, shared.UIDNotFoundError
	}

	return v.CMDRes[uid], nil
}

func (s *pluginShared) RunCMD(id int, cmd string) (uuid.UUID, chan string, error) {
	v, ok := agents[id]
	if !ok {
		return uuid.Nil, nil, shared.AgentNotExistError
	}
	if !v.Online {
		return uuid.Nil, nil, shared.AgentNotOnlineError
	}

	uid := uuid.New()

	if agents[id].CMDRes == nil {
		agents[id].CMDRes = make(map[uuid.UUID]*shared.CMDRes)
	}
	if agents[id].CMDRes[uid] == nil {
		agents[id].CMDRes[uid] = &shared.CMDRes{
			End:      false,
			DataChan: make(chan string),
		}
	}

	d, _ := json.Marshal(msg.CMDMsg{
		UID:  uid,
		Data: cmd,
	})
	err := msg.SendReq(v.Conn, msg.OPCMD, string(d))
	if err != nil {
		return uuid.Nil, nil, err
	}

	return uid, agents[id].CMDRes[uid].DataChan, nil
}

func (s *pluginShared) GetAgents() map[int]*shared.AgentInfo {
	return agents
}
