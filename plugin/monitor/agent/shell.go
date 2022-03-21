package main

import (
	"io"
	"os"
	"os/exec"
	"skynet/plugin/monitor/msg"
	"skynet/plugin/monitor/shared"
	"skynet/sn/tpl"
	"skynet/sn/utils"

	"github.com/creack/pty"
	"github.com/google/uuid"
	"github.com/ztrue/tracerr"
)

type shellAgent struct {
	SID uuid.UUID // shell agent id
	CMD *exec.Cmd // shell agent cmd
	TTY *os.File  // shell agent tty
}

var shellInstance tpl.SafeMap[uuid.UUID, *shellAgent]

var (
	ErrShellAgentNotExist = tracerr.New("shell agent not exist")
)

func getShellAgent(sid uuid.UUID) (*shellAgent, error) {
	item, ok := shellInstance.Get(sid)
	if ok {
		return item, nil
	}
	return nil, ErrShellAgentNotExist
}

// CreateShell creates new shell object.
func CreateShell(size *msg.ShellSizeMessage) (uuid.UUID, error) {
	// pty shell
	cmd := exec.Command("/bin/bash")
	cmd.Env = append(os.Environ(), "TERM=xterm")
	tty, err := pty.Start(cmd)
	if err != nil {
		return uuid.Nil, tracerr.Wrap(err)
	}
	pty.Setsize(tty, &pty.Winsize{
		Rows: uint16(size.Rows),
		Cols: uint16(size.Cols),
		X:    uint16(size.X),
		Y:    uint16(size.Y),
	})
	sid := uuid.New()
	item := &shellAgent{
		CMD: cmd,
		TTY: tty,
		SID: sid,
	}
	shellInstance.Set(sid, item)
	return sid, nil
}

// CloseShell closes and delete shell object.
func CloseShell(sid uuid.UUID) {
	item, err := getShellAgent(sid)
	if err != nil {
		return
	}
	item.CMD.Process.Kill()
	item.CMD.Process.Wait()
	item.TTY.Close()
	shellInstance.Delete(sid)
}

// HandleShellInput sends str to tty.
func HandleShellInput(sid uuid.UUID, str string) {
	item, err := getShellAgent(sid)
	if err != nil {
		utils.WithTrace(err).Warn(err)
		return
	}
	io.WriteString(item.TTY, str)
}

func SetShellSize(sid uuid.UUID, size *msg.ShellSizeMessage) {
	item, err := getShellAgent(sid)
	if err != nil {
		utils.WithTrace(err).Warn(err)
		return
	}
	pty.Setsize(item.TTY, &pty.Winsize{
		Rows: uint16(size.Rows),
		Cols: uint16(size.Cols),
		X:    uint16(size.X),
		Y:    uint16(size.Y),
	})
}

// HandleShellOutput sends shell output to conn, deadloop until closed.
func HandleShellOutput(conn *shared.Websocket, sid uuid.UUID) {
	item, err := getShellAgent(sid)
	if err != nil {
		utils.WithTrace(err).Warn(err)
		return
	}
	for {
		buf := make([]byte, 1024)
		bufLen, err := item.TTY.Read(buf)
		if err != nil {
			return
		}

		_, err = msg.SendMsg(conn, uuid.Nil, msg.AgentMessage_SHELL, &msg.AgentMessage_Shell{
			Shell: &msg.ShellMessage{
				Sid:  sid.String(),
				Type: msg.ShellMessage_OUTPUT,
				Data: &msg.ShellMessage_Putdata{
					Putdata: buf[:bufLen],
				},
			},
		})
		if err != nil {
			utils.WithTrace(err).Warn(err)
		}
	}
}
