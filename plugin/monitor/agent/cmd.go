package main

import (
	"context"
	"fmt"
	"skynet/plugin/monitor/msg"
	"skynet/plugin/monitor/shared"
	"skynet/sn/tpl"
	"skynet/sn/utils"
	"time"

	"github.com/go-cmd/cmd"
	"github.com/google/uuid"
	"github.com/ztrue/tracerr"
	"google.golang.org/protobuf/proto"
)

var cmdInstance tpl.SafeMap[uuid.UUID, *cmd.Cmd]

var (
	ErrCMDNotRunning = tracerr.New("command not running")
)

func KillCommand(cid uuid.UUID) error {
	if c, exist := cmdInstance.Get(cid); exist {
		return c.Stop()
	}
	return ErrCMDNotRunning
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
	ret, err := proto.Marshal(&msg.CMDResMessage{
		Cid:      cid.String(),
		Data:     sendmsg + fmt.Sprintf("\nTask exit with code: %v", status.Exit),
		Code:     int32(status.Exit),
		Complete: status.Complete,
		End:      true,
	})
	if err != nil {
		utils.WithTrace(err).Warn(err)
	}
	if err := msg.SendMsgRet(c, mid, 0, string(ret)); err != nil {
		utils.WithTrace(err).Warn(err)
	}
}

func RunCommandAsync(c *shared.Websocket, cid uuid.UUID, name string, args ...string) {
	payload := cmd.NewCmd(name, args...)
	cmdInstance.Set(cid, payload)
	s := payload.Start()
	t := time.NewTicker(1 * time.Second)
	ctx, cancel := context.WithCancel(context.Background())

	send := func(m string, code int, complete bool, end bool) {
		_, err := msg.SendMsg(c, uuid.Nil, msg.AgentMessage_COMMAND, &msg.AgentMessage_Command{
			Command: &msg.CommandMessage{
				Type: msg.CommandMessage_RESULT,
				Data: &msg.CommandMessage_Res{
					Res: &msg.CMDResMessage{
						Cid:      cid.String(),
						Data:     m,
						Code:     int32(code),
						Complete: complete,
						End:      end,
					},
				},
			},
		})
		if err != nil {
			utils.WithTrace(err).Warn(err)
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
		<-s
		t.Stop()
		cancel()
		cmdInstance.Delete(cid)
	}()
}
