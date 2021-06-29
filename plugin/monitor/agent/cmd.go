package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"skynet/plugin/monitor/msg"
	"time"

	"github.com/go-cmd/cmd"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

var cmdInstance = make(map[uuid.UUID]*cmd.Cmd)

var (
	CMDNotRunningError = errors.New("Command not running")
)

func KillCommand(uid uuid.UUID) error {
	if c, exist := cmdInstance[uid]; exist {
		return c.Stop()
	}
	return CMDNotRunningError
}

func RunCommandSync(c *websocket.Conn, mid uuid.UUID, uid uuid.UUID, name string, args ...string) {
	payload := cmd.NewCmd(name, args...)
	cmdInstance[uid] = payload
	defer delete(cmdInstance, uid)
	status := <-payload.Start()
	if !status.Complete {
		return
	}
	sendmsg := ""
	for _, v := range status.Stdout {
		sendmsg += v + "\n"
	}
	d, err := json.Marshal(msg.CMDResMsg{
		UID:      uid,
		Data:     sendmsg + fmt.Sprintf("\nTask exit with code: %v", status.Exit),
		Code:     status.Exit,
		Complete: status.Complete,
		End:      true,
	})
	if err != nil {
		log.Fatal(err)
	}
	err = msg.SendRsp(c, mid, 0, string(d))
	if err != nil {
		log.Warn("Could not send cmd result")
	}
}

func RunCommandAsync(c *websocket.Conn, uid uuid.UUID, name string, args ...string) {
	payload := cmd.NewCmd(name, args...)
	cmdInstance[uid] = payload
	s := payload.Start()
	t := time.NewTicker(1 * time.Second)
	ctx, cancel := context.WithCancel(context.Background())

	send := func(m string, code int, complete bool, end bool) {
		d, err := json.Marshal(msg.CMDResMsg{
			UID:      uid,
			Data:     m,
			Code:     code,
			Complete: complete,
			End:      end,
		})
		if err != nil {
			log.Fatal(err)
		}
		_, err = msg.SendReq(c, msg.OPCMDRes, string(d))
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
			delete(cmdInstance, uid)
		}
	}()
}
