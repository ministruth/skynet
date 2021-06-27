package main

import (
	"context"
	"encoding/json"
	"fmt"
	"skynet/plugin/monitor/msg"
	"time"

	"github.com/go-cmd/cmd"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

var cmdInstance = make(map[uuid.UUID]*cmd.Cmd)

func KillCommand(uid uuid.UUID) {
	if c, exist := cmdInstance[uid]; exist {
		c.Stop()
	}
}

func RunCommand(c *websocket.Conn, uid uuid.UUID, name string, args ...string) {
	payload := cmd.NewCmd(name, args...)
	cmdInstance[uid] = payload
	s := payload.Start()
	t := time.NewTicker(1 * time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	force := false

	send := func(m string, code int, complete bool, end bool) {
		d, _ := json.Marshal(msg.CMDMsg{
			UID:      uid,
			Data:     m,
			Code:     code,
			Complete: complete,
			End:      end,
		})
		err := msg.SendReq(c, msg.OPCMDRes, string(d))
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
				if force {
					send(sendmsg+"\nKilled after 3600s.", payload.Status().Exit, false, true)
					return
				}
				send(sendmsg+fmt.Sprintf("\nTask exit with code: %v", payload.Status().Exit), payload.Status().Exit, payload.Status().Complete, true)
				return
			}
		}
	}()
	go func() {
		select {
		case <-time.After(1 * time.Hour):
			payload.Stop()
			t.Stop()
			force = true
			cancel()
			delete(cmdInstance, uid)
		case <-s:
			t.Stop()
			cancel()
			delete(cmdInstance, uid)
		}
	}()
}
