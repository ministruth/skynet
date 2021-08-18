package main

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"skynet/plugin/monitor/msg"
	"skynet/plugin/monitor/shared"
	"skynet/sn/utils"

	"github.com/creack/pty"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type shellAgent struct {
	SID uuid.UUID // shell agent id
	CMD *exec.Cmd // shell agent cmd
	TTY *os.File  // shell agent tty
}

var shellInstance utils.UUIDMap

var (
	ShellAgentNotExist = errors.New("Shell agent not exist")
)

func getShellAgent(sid uuid.UUID) (*shellAgent, error) {
	item, ok := shellInstance.Get(sid)
	if ok {
		return item.(*shellAgent), nil
	}
	return nil, ShellAgentNotExist
}

// CreateShell creates new shell object.
func CreateShell(size *msg.ShellSizeMsg) (uuid.UUID, error) {
	// pty shell
	cmd := exec.Command("/bin/bash")
	cmd.Env = append(os.Environ(), "TERM=xterm")
	tty, err := pty.Start(cmd)
	if err != nil {
		return uuid.Nil, err
	}
	pty.Setsize(tty, &pty.Winsize{
		Rows: size.Rows,
		Cols: size.Cols,
		X:    size.X,
		Y:    size.Y,
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
		log.Warn(err)
		return
	}
	io.WriteString(item.TTY, str)
}

func SetShellSize(sid uuid.UUID, size *msg.ShellSizeMsg) {
	item, err := getShellAgent(sid)
	if err != nil {
		log.Warn(err)
		return
	}
	pty.Setsize(item.TTY, &pty.Winsize{
		Rows: size.Rows,
		Cols: size.Cols,
		X:    size.X,
		Y:    size.Y,
	})
}

// HandleShellOutput sends shell output to conn, deadloop until closed.
func HandleShellOutput(conn *shared.Websocket, sid uuid.UUID) {
	item, err := getShellAgent(sid)
	if err != nil {
		log.Warn(err)
		return
	}
	for {
		buf := make([]byte, 1024)
		bufLen, err := item.TTY.Read(buf)
		if err != nil {
			return
		}

		_, err = msg.SendMsgByte(conn, uuid.Nil, msg.OPShell, msg.Marshal(msg.ShellMsg{
			ID:     uuid.New(),
			SID:    sid,
			OPCode: msg.ShellOutput,
			Data:   buf[:bufLen],
		}))
		if err != nil {
			log.Warn(err)
		}
	}
}
