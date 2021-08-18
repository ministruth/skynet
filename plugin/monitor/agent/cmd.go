package main

import (
	"context"
	"errors"
	"fmt"
	"skynet/plugin/monitor/msg"
	"skynet/plugin/monitor/shared"
	"skynet/sn/utils"
	"time"

	"github.com/go-cmd/cmd"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

var cmdInstance utils.UUIDMap

var (
	CMDNotRunningError = errors.New("Command not running")
)

func KillCommand(cid uuid.UUID) error {
	if c, exist := cmdInstance.Get(cid); exist {
		return c.(*cmd.Cmd).Stop()
	}
	return CMDNotRunningError
}

func RunCommandSync(c *shared.Websocket, mid uuid.UUID, cid uuid.UUID, name string, args ...string) {
	payload := cmd.NewCmd(name, args...)
	cmdInstance.Set(cid, payload)
	defer cmdInstance.Delete(cid)
	status := <-payload.Start()
	if !status.Complete {
		return
	}
	sendmsg := ""
	for _, v := range status.Stdout {
		sendmsg += v + "\n"
	}
	err := msg.SendMsgRet(c, mid, 0, string(msg.Marshal(msg.CMDResMsg{
		CID:      cid,
		Data:     sendmsg + fmt.Sprintf("\nTask exit with code: %v", status.Exit),
		Code:     status.Exit,
		Complete: status.Complete,
		End:      true,
	})))
	if err != nil {
		log.Warn("Could not send cmd result")
	}
}

func RunCommandAsync(c *shared.Websocket, cid uuid.UUID, name string, args ...string) {
	payload := cmd.NewCmd(name, args...)
	cmdInstance.Set(cid, payload)
	s := payload.Start()
	t := time.NewTicker(1 * time.Second)
	ctx, cancel := context.WithCancel(context.Background())

	send := func(m string, code int, complete bool, end bool) {
		_, err := msg.SendMsgByte(c, uuid.Nil, msg.OPCMDRes, msg.Marshal(msg.CMDResMsg{
			CID:      cid,
			Data:     m,
			Code:     code,
			Complete: complete,
			End:      end,
		}))
		if err != nil {
			log.Warn("Could not send cmd result")
		}
	}

	go func() {
		line := 0
		consume := func() string {
			status := payload.Status()
			n := len(status.Stdout)
			ret := ""
			if n > line {
				for i := line; i < n; i++ {
					ret += status.Stdout[i] + "\n"
				}
				line = n
			}
			return ret
		}
		for {
			select {
			case <-t.C:
				sendmsg := consume()
				if sendmsg != "" {
					send(sendmsg, 0, false, false)
				}
			case <-ctx.Done():
				sendmsg := consume()
				send(sendmsg+fmt.Sprintf("\nTask exit with code: %v", payload.Status().Exit), payload.Status().Exit, payload.Status().Complete, true)
				return
			}
		}
	}()
	go func() {
		select {
		case <-s:
			t.Stop()
			cancel()
			cmdInstance.Delete(cid)
		}
	}()
}
