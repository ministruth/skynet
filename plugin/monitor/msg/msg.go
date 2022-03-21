package msg

import (
	"context"
	"skynet/plugin/monitor/shared"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/ztrue/tracerr"
	"google.golang.org/protobuf/proto"
)

var (
	ErrUnknownMsg = tracerr.New("unknown message")
	ErrFormat     = tracerr.New("message format error")
)

// RecvShellMsg receives message from client c.
func RecvShellMsg(c *shared.Websocket) (*ShellMessage, []byte, error) {
	_, msgRead, err := c.ReadMessage()
	if err != nil {
		return nil, nil, err
	}
	var res ShellMessage
	if err := proto.Unmarshal(msgRead, &res); err != nil {
		return nil, msgRead, err
	}
	if res.Type == ShellMessage_UNKNOWN {
		return nil, msgRead, ErrUnknownMsg
	}
	if _, err := uuid.Parse(res.Sid); err != nil {
		return nil, msgRead, err
	}

	return &res, msgRead, nil
}

// RecvMsg receives message from agent c.
func RecvMsg(c *shared.Websocket) (*AgentMessage, []byte, error) {
	_, msgRead, err := c.ReadMessage()
	if err != nil {
		return nil, nil, err
	}
	var res AgentMessage
	if err := proto.Unmarshal(msgRead, &res); err != nil {
		return nil, msgRead, err
	}
	if res.Type == AgentMessage_UNKNOWN {
		return nil, msgRead, ErrUnknownMsg
	}
	if _, err := uuid.Parse(res.Id); err != nil {
		return nil, msgRead, err
	}

	return &res, msgRead, nil
}

func ctxChan(ctx context.Context, c chan interface{}) (*ReturnMessage, bool) {
	select {
	case <-ctx.Done():
		return nil, false
	case ret, ok := <-c:
		if !ok {
			return nil, false
		}
		return ret.(*ReturnMessage), true
	}
}

// RecvMsgRet receive return value from recvChan matching message id with timeout.
func RecvMsgRet(id uuid.UUID, recvChan *shared.ChanMap, timeout time.Duration) (*ReturnMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	c, _ := recvChan.SetIfAbsent(id)
	defer recvChan.Delete(id)
	ret, ok := ctxChan(ctx, c)
	if !ok {
		return nil, shared.ErrOPTimeout
	}
	if ret.Code == -1 {
		return ret, tracerr.New(ret.Data)
	}
	return ret, nil
}

// SendShellMsg send byte message d to c.
func SendShellMsg(c *shared.Websocket, id uuid.UUID, t ShellMessage_MsgType, d isShellMessage_Data) error {
	data, err := proto.Marshal(&ShellMessage{
		Sid:  id.String(),
		Type: t,
		Data: d,
	})
	if err != nil {
		return err
	}
	return c.WriteMessage(websocket.TextMessage, data)
}

// SendMsg send byte message d to c, return message uuid and error.
// If id == uuid.Nil random new id.
func SendMsg(c *shared.Websocket, id uuid.UUID, t AgentMessage_MsgType, d isAgentMessage_Data) (uuid.UUID, error) {
	if id == uuid.Nil {
		id = uuid.New()
	}
	data, err := proto.Marshal(&AgentMessage{
		Id:   id.String(),
		Type: t,
		Data: d,
	})
	if err != nil {
		return uuid.Nil, err
	}
	return id, c.WriteMessage(websocket.TextMessage, data)
}

// SendMsgRet send return message d to c.
func SendMsgRet(c *shared.Websocket, id uuid.UUID, code ReturnMessage_ReturnCode, d string) error {
	_, err := SendMsg(c, id, AgentMessage_RETURN, &AgentMessage_Return{
		Return: &ReturnMessage{
			Code: code,
			Data: d,
		},
	})
	return err
}
