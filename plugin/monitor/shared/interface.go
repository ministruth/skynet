package shared

import (
	"errors"
	"os"
	plugins "skynet/plugin"
	"skynet/sn"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var (
	AgentNotExistError  = errors.New("Agent not exist")
	AgentNotOnlineError = errors.New("Agent not online")
	UIDNotFoundError    = errors.New("Command UID not found")
	OPTimeoutError      = errors.New("Operation timeout")
)

type PluginMonitorAgent struct {
	ID        int32  `gorm:"primaryKey;not null"`
	UID       string `gorm:"uniqueIndex;type:char(32);not null"`
	Name      string `gorm:"type:varchar(32);not null"`
	Hostname  string `gorm:"type:varchar(256)"`
	LastIP    string `gorm:"type:varchar(64)"`
	System    string `gorm:"type:varchar(128)"`
	Machine   string `gorm:"type:varchar(32)"`
	LastLogin time.Time
	Track     sn.Track `gorm:"embedded"`
}

type PluginMonitorAgentSetting struct {
	ID      int32    `gorm:"primaryKey;not null"`
	AgentID int32    `gorm:"not null"`
	Name    string   `gorm:"type:varchar(256);not null"`
	Value   string   `gorm:"type:varchar(1024);not null"`
	Track   sn.Track `gorm:"embedded"`
}

type CMDRes struct {
	Data     string
	Code     int
	Complete bool
	End      bool
	DataChan chan string
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
	GetPluginPath(c *plugins.PluginConfig, p string) string
	DeleteAllSetting(id int) error
	DeleteSetting(id int, name string) error
	GetAllSetting(id int) ([]*PluginMonitorAgentSetting, error)
	GetSetting(id int, name string) (*PluginMonitorAgentSetting, error)
	NewSetting(id int, name string, value string) error
	UpdateSetting(id int, name string, value string) error
	WriteFile(id int, path string, file string, recursive bool, override bool, perm os.FileMode, timeout time.Duration) error
	KillCMD(id int, uid uuid.UUID, isReturn bool) error
	GetCMDRes(id int, uid uuid.UUID) (*CMDRes, error)
	RunCMDAsync(id int, cmd string) (uuid.UUID, chan string, error)
	RunCMDSync(id int, cmd string, timeout time.Duration) (uuid.UUID, string, error)
	GetAgents() map[int]*AgentInfo
}
