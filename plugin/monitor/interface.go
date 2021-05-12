package main

import (
	"encoding/json"
	"skynet/plugin/monitor/msg"
	"skynet/plugin/monitor/shared"

	"github.com/google/uuid"
)

type pluginShared struct{}

func (s pluginShared) GetCMDRes(id int, uid uuid.UUID) (*shared.CMDRes, error) {
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

func (s pluginShared) RunCMD(id int, cmd string) (uuid.UUID, error) {
	v, ok := agents[id]
	if !ok {
		return uuid.Nil, shared.AgentNotExistError
	}
	if !v.Online {
		return uuid.Nil, shared.AgentNotOnlineError
	}

	uid := uuid.New()
	d, _ := json.Marshal(msg.CMDMsg{
		UID:  uid,
		Data: cmd,
	})
	err := msg.SendReq(v.Conn, msg.OPCMD, string(d))
	if err != nil {
		return uuid.Nil, err
	}

	return uid, nil
}

func (s pluginShared) GetAgents() map[int]*shared.AgentInfo {
	return agents
}
