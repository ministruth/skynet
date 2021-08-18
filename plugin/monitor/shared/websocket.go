package shared

import (
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Websocket struct {
	websocket.Conn
	rMutex sync.Mutex
	wMutex sync.Mutex
}

func NewWebsocket(up *websocket.Upgrader, w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*Websocket, error) {
	var ret Websocket
	connTemp, err := up.Upgrade(w, r, responseHeader)
	if err != nil {
		return nil, err
	}
	ret.Conn = *connTemp
	return &ret, nil
}

func DialWebsocket(d *websocket.Dialer, urlStr string, requestHeader http.Header) (*Websocket, *http.Response, error) {
	var ret Websocket
	connTemp, resp, err := d.Dial(urlStr, requestHeader)
	if err != nil {
		return nil, resp, err
	}
	ret.Conn = *connTemp
	return &ret, resp, nil
}

func (c *Websocket) NextReader() (messageType int, r io.Reader, err error) {
	c.rMutex.Lock()
	defer c.rMutex.Unlock()
	return c.Conn.NextReader()
}

func (c *Websocket) NextWriter(messageType int) (io.WriteCloser, error) {
	c.wMutex.Lock()
	defer c.wMutex.Unlock()
	return c.Conn.NextWriter(messageType)
}

func (c *Websocket) SetReadDeadline(t time.Time) error {
	c.rMutex.Lock()
	defer c.rMutex.Unlock()
	return c.Conn.SetReadDeadline(t)
}

func (c *Websocket) SetWriteDeadline(t time.Time) error {
	c.wMutex.Lock()
	defer c.wMutex.Unlock()
	return c.Conn.SetWriteDeadline(t)
}

func (c *Websocket) WriteJSON(v interface{}) error {
	c.wMutex.Lock()
	defer c.wMutex.Unlock()
	return c.Conn.WriteJSON(v)
}

func (c *Websocket) WriteMessage(messageType int, data []byte) error {
	c.wMutex.Lock()
	defer c.wMutex.Unlock()
	return c.Conn.WriteMessage(messageType, data)
}

func (c *Websocket) ReadJSON(v interface{}) error {
	c.rMutex.Lock()
	defer c.rMutex.Unlock()
	return c.Conn.ReadJSON(v)
}

func (c *Websocket) ReadMessage() (messageType int, p []byte, err error) {
	c.rMutex.Lock()
	defer c.rMutex.Unlock()
	return c.Conn.ReadMessage()
}
