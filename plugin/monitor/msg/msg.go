package msg

import (
	"encoding/json"

	"github.com/gorilla/websocket"
)

type OPCode int

const (
	OPLogin OPCode = iota + 1
	OPInfo
)

type CommonMsg struct {
	Opcode OPCode
	Data   string
}
type RetMsg struct {
	Code int
	Msg  string
}

type LoginMsg struct {
	UID   string
	Token string
}

type InfoMsg struct {
	Host    string
	Machine string
	System  string
}

func SendReq(c *websocket.Conn, o OPCode, d string) error {
	data, _ := json.Marshal(CommonMsg{
		Opcode: o,
		Data:   d,
	})
	return c.WriteMessage(websocket.TextMessage, data)
}

func SendRsp(c *websocket.Conn, code int, msg string) error {
	data, _ := json.Marshal(RetMsg{
		Code: code,
		Msg:  msg,
	})
	return c.WriteMessage(websocket.TextMessage, data)
}

func Recv(c *websocket.Conn) (*RetMsg, error) {
	_, m, err := c.ReadMessage()
	if err != nil {
		return nil, err
	}
	var ret RetMsg
	err = json.Unmarshal(m, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}
