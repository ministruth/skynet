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

type OPCode int

const (
	OPLogin OPCode = iota + 1
	OPRet
	OPInfo
	OPStat
	OPCMDRes
	OPCMD
	OPCMDKill
	OPFile
)

type CommonMsg struct {
	ID     uuid.UUID
	Opcode OPCode
	Data   string
}

type RetMsg struct {
	Code int
	Data string
}

type LoginMsg struct {
	UID   string
	Token string
}

type FileMsg struct {
	Path      string
	File      []byte
	Recursive bool
	Override  bool
	Perm      os.FileMode
}

type CMDKillMsg struct {
	UID    uuid.UUID
	Return bool
}

type InfoMsg struct {
	Host    string
	Machine string
	System  string
}

type CMDMsg struct {
	UID     uuid.UUID
	Payload string
	Sync    bool
}

type CMDResMsg struct {
	UID      uuid.UUID
	Data     string
	Code     int
	Complete bool
	End      bool
}

type StatMsg struct {
	CPU       float64 // percent
	Mem       uint64  // bytes
	TotalMem  uint64  // bytes
	Disk      uint64  // bytes
	TotalDisk uint64  // bytes
	Load1     float64
	Time      time.Time
	BandUp    uint64 // bytes
	BandDown  uint64 // bytes
}

func SendReqWithID(c *websocket.Conn, id uuid.UUID, o OPCode, d string) error {
	data, err := json.Marshal(CommonMsg{
		ID:     id,
		Opcode: o,
		Data:   d,
	})
	if err != nil {
		log.Fatal(err)
	}
	return c.WriteMessage(websocket.TextMessage, data)
}

func SendReq(c *websocket.Conn, o OPCode, d string) (uuid.UUID, error) {
	id := uuid.New()
	data, err := json.Marshal(CommonMsg{
		ID:     id,
		Opcode: o,
		Data:   d,
	})
	if err != nil {
		log.Fatal(err)
	}
	return id, c.WriteMessage(websocket.TextMessage, data)
}

func SendRsp(c *websocket.Conn, id uuid.UUID, code int, d string) error {
	retData, err := json.Marshal(RetMsg{
		Code: code,
		Data: d,
	})
	if err != nil {
		log.Fatal(err)
	}
	data, err := json.Marshal(CommonMsg{
		ID:     id,
		Opcode: OPRet,
		Data:   string(retData),
	})
	if err != nil {
		log.Fatal(err)
	}
	return c.WriteMessage(websocket.TextMessage, data)
}

func Recv(c *websocket.Conn) (*CommonMsg, []byte, error) {
	_, msgRead, err := c.ReadMessage()
	if err != nil {
		return nil, nil, err
	}
	var res CommonMsg
	err = json.Unmarshal(msgRead, &res)
	if err != nil {
		return nil, msgRead, err
	}
	return &res, msgRead, nil
}

func CtxChan(ctx context.Context, c chan RetMsg) (*RetMsg, bool) {
	select {
	case <-ctx.Done():
		return nil, false
	case ret := <-c:
		return &ret, true
	}
}

func RecvRsp(id uuid.UUID, recvChan map[uuid.UUID]chan RetMsg, timeout time.Duration) (*RetMsg, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	recvChan[id] = make(chan RetMsg)
	defer delete(recvChan, id)
	ret, ok := CtxChan(ctx, recvChan[id])
	if !ok {
		return nil, shared.OPTimeoutError
	}
	if ret.Code == -1 {
		return ret, errors.New(ret.Data)
	}
	return ret, nil
}
