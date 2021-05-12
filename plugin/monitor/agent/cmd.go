package main

import (
	"context"
	"encoding/json"
	"skynet/plugin/monitor/msg"
	"time"

	"github.com/go-cmd/cmd"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

func RunCommand(c *websocket.Conn, uid uuid.UUID, name string, args ...string) {
	payload := cmd.NewCmd(name, args...)
	s := payload.Start()
	t := time.NewTicker(1 * time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	f := func(m string, end bool) {
		status := payload.Status()
		n := len(status.Stdout)
		d, _ := json.Marshal(msg.CMDMsg{
			UID:  uid,
			Data: status.Stdout[n-1] + m,
			End:  end,
		})
		err := msg.SendReq(c, msg.OPCMDRes, string(d))
		if err != nil {
			log.Warn("Could not send cmd result")
		}
	}
	force := false

	go func() {
		for {
			select {
			case <-t.C:
				f("", false)
			case <-ctx.Done():
				if force {
					f("\n\nKilled after 3600s.", true)
					return
				}
				f("", true)
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
		case <-s:
			payload.Stop()
			t.Stop()
			cancel()
		}
	}()
}
