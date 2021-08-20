package shared

import (
	"errors"
	"os"
	"skynet/sn"
	"time"

	plugins "skynet/plugin"

	"github.com/google/uuid"
)

const AgentVersion = "0.1.1"

var (
	// AgentNotExistError represents agent not exist.
	AgentNotExistError = errors.New("Agent not exist")
	// AgentNotOnlineError represents agent not online.
	AgentNotOnlineError = errors.New("Agent not online")
	// CMDIDNotFoundError represents command id not found.
	CMDIDNotFoundError = errors.New("Command ID not found")
	// OPTimeoutError represents operation timeout.
	OPTimeoutError = errors.New("Operation timeout")
)

// PluginMonitorAgent is agent database record.
type PluginMonitorAgent struct {
	ID        int32     `gorm:"primaryKey;not null"`                // agent id
	UID       string    `gorm:"uniqueIndex;type:char(32);not null"` // agent uid, md5(machineid)
	Name      string    `gorm:"type:varchar(32);not null"`          // agent recongnized name, default uid[:6]
	Hostname  string    `gorm:"type:varchar(256)"`                  // agent hostname
	LastIP    string    `gorm:"type:varchar(64)"`                   // agent last connect ip
	System    string    `gorm:"type:varchar(128)"`                  // agent system(Linux manjaro 5.4.131-1-MANJARO)
	Machine   string    `gorm:"type:varchar(32)"`                   // agent machine type(x86_64)
	LastLogin time.Time // agent last connect time
	Track     sn.Track  `gorm:"embedded"`
}

// PluginMonitorAgentSetting is agent setting database record.
type PluginMonitorAgentSetting struct {
	ID      int32    `gorm:"primaryKey;not null"`         // setting id
	AgentID int32    `gorm:"not null"`                    // agent id
	Name    string   `gorm:"type:varchar(256);not null"`  // setting name
	Value   string   `gorm:"type:varchar(1024);not null"` // setting value
	Track   sn.Track `gorm:"embedded"`
}

// CMDRes is command result struct.
type CMDRes struct {
	Data     string      // command full result data, appended when command running
	Code     int         // command return code
	Complete bool        // is command complete, false when terminated
	End      bool        // is command end, false when running
	DataChan chan string // command result data channel, pushed new output when receiving new one, closed when command end
}

// AgentInfo is agent information struct.
type AgentInfo struct {
	ID         int        // agent id
	IP         string     // agent ip
	Name       string     // agent name
	HostName   string     // agent hostname
	LastLogin  time.Time  // agent last login time
	System     string     // agent system(Linux manjaro 5.4.131-1-MANJARO)
	Machine    string     // agent machine type(x86_64)
	Conn       *Websocket `json:"-"` // agent websocket
	CMDRes     *CMDResMap `json:"-"` // agent command result
	MsgRetChan *ChanMap   `json:"-"` // message return value channel
	ShellConn  *SocketMap `json:"-"` // agent shell client websocket
	Online     bool       // agent online status
	Updating   bool       // agent updating

	LastRsp   time.Time // agent last response time
	CPU       float64   // cpu status, unit percent
	Mem       uint64    // memory status, unit bytes
	TotalMem  uint64    // total memory, unit bytes
	Disk      uint64    // disk status, unit bytes
	TotalDisk uint64    // total disk, unit bytes
	Load1     float64   // cpu load1
	Latency   int64     // agent latency, unit ms
	NetUp     uint64    // network upload, unit bytes/s
	NetDown   uint64    // network download, unit bytes/s
	BandUp    uint64    // bandwidth upload, unit bytes
	BandDown  uint64    // bandwidth download, unit bytes
}

// PluginShared is monitor shared API.
type PluginShared interface {
	// GetConfig return plugin config.
	GetConfig() *plugins.PluginConfig

	// DeleteAllSetting deletes all agent id setting, return affected rows and error.
	DeleteAllSetting(id int) (int64, error)

	// DeleteSetting delete agent id setting[name].
	DeleteSetting(id int, name string) error

	// GetAllSetting returns all agent id setting.
	GetAllSetting(id int) ([]*PluginMonitorAgentSetting, error)

	// GetSetting returns agent id setting[name].
	GetSetting(id int, name string) (*PluginMonitorAgentSetting, error)

	// NewSetting creates agent id setting[name] = value, return setting id and error.
	NewSetting(id int, name string, value string) (int, error)

	// UpdateSetting updates agent id setting[name] = value.
	UpdateSetting(id int, name string, value string) error

	// WriteFile writes localPath file to agent id's remotePath with perm and timeout.
	//
	// When recursive is true, create folder to satisfy remotePath. When override is true,
	// override remotePath existed file.
	WriteFile(id int, remotePath string, localPath string, recursive bool, override bool, perm os.FileMode, timeout time.Duration) error

	// KillCMD kills agent id command cid.
	KillCMD(id int, cid uuid.UUID) error

	// GetCMDRes returns agent id command cid result.
	GetCMDRes(id int, cid uuid.UUID) (*CMDRes, error)

	// RunCMDAsync run asynchronous cmd in agent id, return command uid, output data channel and error immediately.
	//
	// Warning: You must consume data channel ASAP for there are only 60 buffer in it, not consuming data will lead to data loss.
	RunCMDAsync(id int, cmd string) (uuid.UUID, chan string, error)

	// RunCMDSync run synchronous cmd in agent id with timeout, return command uid, output string and error.
	RunCMDSync(id int, cmd string, timeout time.Duration) (uuid.UUID, string, error)

	// GetAgents returns all agent map.
	GetAgents() map[int]*AgentInfo
}
