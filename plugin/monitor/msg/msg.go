package msg

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"skynet/plugin/monitor/shared"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// OPCode is agent message operation.
type OPCode int

const (
	OPLogin   OPCode = iota + 1 // agent login
	OPRet                       // return value message
	OPInfo                      // agent info
	OPStat                      // agent status
	OPCMDRes                    // agent command result
	OPCMD                       // run command
	OPCMDKill                   // kill command
	OPFile                      // send file
	OPShell                     // shell operation
	OPRestart                   // restart
)

// ShellOP is shell message operation.
type ShellOP int

const (
	ShellError      ShellOP = iota - 1 // error message
	ShellReturn                        // return value message
	ShellInput                         // input message
	ShellOutput                        // output message
	ShellSize                          // resize message
	ShellConnect                       // shell connect message
	ShellDisconnect                    // shell disconnect message
)

var (
	MsgFormatError = errors.New("Msg format error")
)

type CommonMsg struct {
	ID     uuid.UUID // msg id
	OPCode OPCode    // msg opcode
	Data   []byte    // msg data
}

type ShellMsg struct {
	ID     uuid.UUID // shell msg id
	SID    uuid.UUID // shell agent id
	OPCode ShellOP   // shell msg opcode
	Data   []byte    // shell msg data
}

type RetMsg struct {
	Code int    // return code
	Data string // return data
}

type LoginMsg struct {
	UID   string // agent uid
	Token string // login token
}

type FileMsg struct {
	Path      string      // file save path
	File      []byte      // file data
	Recursive bool        // recursive create
	Override  bool        // override path
	Perm      os.FileMode // permission
}

type CMDKillMsg struct {
	CID uuid.UUID // command cid to kill
}

type InfoMsg struct {
	Version string // version
	Host    string // host name
	Machine string // machine name
	System  string // system name
}

type CMDMsg struct {
	CID     uuid.UUID // command cid
	Payload string    // command payload
	Sync    bool      // sync run
}

type CMDResMsg struct {
	CID      uuid.UUID // command cid
	Data     string    // command result
	Code     int       // return code
	Complete bool      // command complete or killed
	End      bool      // is result end
}

type ShellSizeMsg struct {
	Rows uint16 // row
	Cols uint16 // col
	X    uint16 // x
	Y    uint16 // y
}

type ShellConnectMsg struct {
	ID int // agent id
	ShellSizeMsg
}

type StatMsg struct {
	CPU       float64   // unit percent
	Mem       uint64    // unit bytes
	TotalMem  uint64    // unit bytes
	Disk      uint64    // unit bytes
	TotalDisk uint64    // unit bytes
	Load1     float64   // cpu load1
	Time      time.Time // collect time
	BandUp    uint64    // unit bytes
	BandDown  uint64    // unit bytes
}

// Marshal uses json marshal to convert v to bytes, exit when facing any error.
func Marshal(v interface{}) []byte {
	d, err := json.Marshal(v)
	if err != nil {
		log.Fatal(err)
	}
	return d
}

// Unmarshal just wrapper for json.Unmarshal.
func Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// SendMsgStr send string message s to c, return message uuid and error.
// If id == uuid.Nil random new id.
func SendMsgStr(c *shared.Websocket, id uuid.UUID, o OPCode, s string) (uuid.UUID, error) {
	return SendMsgByte(c, id, o, []byte(s))
}

// SendMsg send byte message b to c, return message uuid and error.
// If id == uuid.Nil random new id.
func SendMsgByte(c *shared.Websocket, id uuid.UUID, o OPCode, b []byte) (uuid.UUID, error) {
	if id == uuid.Nil {
		id = uuid.New()
	}
	return id, c.WriteMessage(websocket.TextMessage, Marshal(CommonMsg{
		ID:     id,
		OPCode: o,
		Data:   b,
	}))
}

// SendMsgRet send return message d to c.
func SendMsgRet(c *shared.Websocket, id uuid.UUID, code int, d string) error {
	_, err := SendMsgByte(c, id, OPRet, Marshal(RetMsg{
		Code: code,
		Data: d,
	}))
	return err
}

func ctxChan(ctx context.Context, c chan interface{}) (*RetMsg, bool) {
	select {
	case <-ctx.Done():
		return nil, false
	case ret, ok := <-c:
		if !ok {
			return nil, false
		}
		return ret.(*RetMsg), true
	}
}

// RecvMsgRet receive return value from recvChan matching message id with timeout.
func RecvMsgRet(id uuid.UUID, recvChan *shared.ChanMap, timeout time.Duration) (*RetMsg, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	c, _ := recvChan.SetIfAbsent(id)
	defer recvChan.Delete(id)
	ret, ok := ctxChan(ctx, c)
	if !ok {
		return nil, shared.OPTimeoutError
	}
	if ret.Code == -1 {
		return ret, errors.New(ret.Data)
	}
	return ret, nil
}

// SendShellMsgByte sends byte message b to shell client c.
func SendShellMsgByte(c *shared.Websocket, id uuid.UUID, code ShellOP, b []byte) error {
	return c.WriteMessage(websocket.TextMessage, Marshal(ShellMsg{
		ID:     id,
		OPCode: code,
		Data:   b,
	}))
}

// SendShellMsgStr sends string message s to shell client c.
func SendShellMsgStr(c *shared.Websocket, id uuid.UUID, code ShellOP, s string) error {
	return SendShellMsgByte(c, id, code, []byte(s))
}

// RecvShellMsg receives message from shell client c.
func RecvShellMsg(c *shared.Websocket) (*ShellMsg, []byte, error) {
	_, msgRead, err := c.ReadMessage()
	if err != nil {
		return nil, nil, err
	}
	var res ShellMsg
	err = Unmarshal(msgRead, &res)
	if err != nil {
		return nil, msgRead, err
	}
	return &res, msgRead, nil
}

// RecvMsg receives message from agent c.
func RecvMsg(c *shared.Websocket) (*CommonMsg, []byte, error) {
	_, msgRead, err := c.ReadMessage()
	if err != nil {
		return nil, nil, err
	}
	var res CommonMsg
	err = Unmarshal(msgRead, &res)
	if err != nil {
		return nil, msgRead, err
	}
	return &res, msgRead, nil
}
