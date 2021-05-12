package shared

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var (
	AgentNotExistError  = errors.New("Agent not exist")
	AgentNotOnlineError = errors.New("Agent not online")
	UIDNotFoundError    = errors.New("Command UID not found")
)

type CMDRes struct {
	Data string
	End  bool
}

type AgentInfo struct {
	ID        int
	IP        string
	Name      string
	HostName  string
	LastLogin time.Time
	System    string
	Machine   string
	Conn      *websocket.Conn       `json:"-"`
	CMDRes    map[uuid.UUID]*CMDRes `json:"-"`
	Online    bool

	LastRsp   time.Time
	CPU       float64 // percent
	Mem       uint64  // bytes
	TotalMem  uint64  // bytes
	Disk      uint64  // bytes
	TotalDisk uint64  // bytes
	Load1     float64
	Latency   int64  // ms
	NetUp     uint64 // bytes/s
	NetDown   uint64 // bytes/s
	BandUp    uint64 // bytes
	BandDown  uint64 // bytes
}

type PluginShared interface {
	GetCMDRes(id int, uid uuid.UUID) (*CMDRes, error)
	RunCMD(id int, cmd string) (uuid.UUID, error)
	GetAgents() map[int]*AgentInfo
}
