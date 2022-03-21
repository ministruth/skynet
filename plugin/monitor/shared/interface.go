package shared

import (
	"os"
	"skynet/sn"
	"skynet/sn/tpl"
	"time"

	plugins "skynet/plugin"

	"github.com/google/uuid"
	"github.com/ztrue/tracerr"
)

const AgentVersion = "0.2.0"

var (
	// ErrAgentNotExist represents agent not exist.
	ErrAgentNotExist = tracerr.New("agent not exist")
	// ErrAgentNotOnline represents agent not online.
	ErrAgentNotOnline = tracerr.New("agent not online")
	// ErrCMDIDNotFound represents command id not found.
	ErrCMDIDNotFound = tracerr.New("command ID not found")
	// ErrOPTimeout represents operation timeout.
	ErrOPTimeout = tracerr.New("operation timeout")
)

// PluginMonitorAgent is agent database record.
type PluginMonitorAgent struct {
	sn.GeneralFields
	UID       string    `gorm:"column:uid;uniqueIndex;type:char(32);not null"` // agent uid, md5(machineid)
	Name      string    `gorm:"type:varchar(32);not null"`                     // agent recongnized name, default uid[:6]
	OS        string    `gorm:"column:os;type:varchar(32)"`                    // agent os
	Hostname  string    `gorm:"type:varchar(256)"`                             // agent hostname
	LastIP    string    `gorm:"type:varchar(64)"`                              // agent last connect ip
	System    string    `gorm:"type:varchar(128)"`                             // agent system(Linux manjaro 5.4.131-1-MANJARO)
	Machine   string    `gorm:"type:varchar(32)"`                              // agent machine type(x86_64)
	LastLogin time.Time // agent last connect time
}

// PluginMonitorAgentSetting is agent setting database record.
type PluginMonitorAgentSetting struct {
	sn.GeneralFields
	AgentID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uas_link"`         // agent id
	Name    string    `gorm:"type:varchar(256);not null;uniqueIndex:uas_link"` // setting name
	Value   string    `gorm:"type:string"`                                     // setting value
}

// CMDRes is command result struct.
type CMDRes struct {
	Data     string      // command full result data, appended when command running
	Code     int         // command return code
	Complete bool        // is command complete, false when terminated
	End      bool        // is command end, false when running
	DataChan chan string // command result data channel, pushed new output when receiving new one, closed when command end
}

// AgentStatus is status of agent.
type AgentStatus = int32

const (
	AgentOffline AgentStatus = iota
	AgentOnline
	AgentUpdating
)

// AgentInfo is agent information struct.
type AgentInfo struct {
	ID         uuid.UUID                           `json:"id"`         // agent id
	IP         string                              `json:"ip"`         // agent ip
	Name       string                              `json:"name"`       // agent name
	Hostname   string                              `json:"hostname"`   // agent hostname
	LastLogin  time.Time                           `json:"last_login"` // agent last login time
	OS         string                              `json:"os"`         // agent operating system
	System     string                              `json:"system"`     // agent system(Linux manjaro 5.4.131-1-MANJARO)
	Machine    string                              `json:"machine"`    // agent machine type(x86_64)
	Conn       *Websocket                          `json:"-"`          // agent websocket
	CMDRes     *tpl.SafeMap[uuid.UUID, *CMDRes]    `json:"-"`          // agent command result
	MsgRetChan *ChanMap                            `json:"-"`          // message return value channel
	ShellConn  *tpl.SafeMap[uuid.UUID, *Websocket] `json:"-"`          // agent shell client websocket
	Status     AgentStatus                         `json:"status"`     // agent status

	LastRsp   time.Time `json:"last_rsp"`   // agent last response time
	CPU       float64   `json:"cpu"`        // cpu status, unit percent
	Mem       uint64    `json:"mem"`        // memory status, unit bytes
	TotalMem  uint64    `json:"total_mem"`  // total memory, unit bytes
	Disk      uint64    `json:"disk"`       // disk status, unit bytes
	TotalDisk uint64    `json:"total_disk"` // total disk, unit bytes
	Load1     float64   `json:"load1"`      // cpu load1
	Latency   int64     `json:"latency"`    // agent latency, unit ms
	NetUp     uint64    `json:"net_up"`     // network upload, unit bytes/s
	NetDown   uint64    `json:"net_down"`   // network download, unit bytes/s
	BandUp    uint64    `json:"band_up"`    // bandwidth upload, unit bytes
	BandDown  uint64    `json:"band_down"`  // bandwidth download, unit bytes
}

// PluginShared is monitor shared API.
type PluginShared interface {
	sn.SNHandler[PluginShared]

	// GetInstance return plugin instance.
	GetInstance() *plugins.PluginInfo

	// DeleteAllSetting deletes all agent id setting, return affected rows and error.
	DeleteAllSetting(id uuid.UUID) (int64, error)

	// DeleteSetting delete agent id setting[name].
	DeleteSetting(id uuid.UUID, name string) (bool, error)

	// GetAllSetting returns all agent id setting.
	GetAllSetting(id uuid.UUID) ([]*PluginMonitorAgentSetting, error)

	// GetSetting returns agent id setting[name].
	GetSetting(id uuid.UUID, name string) (*PluginMonitorAgentSetting, error)

	// NewSetting creates agent id setting[name] = value, return setting and error.
	NewSetting(id uuid.UUID, name string, value string) (*PluginMonitorAgentSetting, error)

	// UpdateSetting updates agent id setting[name] = value.
	UpdateSetting(id uuid.UUID, name string, value string) error

	// GetAgent returns id agent.
	GetAgent(id uuid.UUID) (*PluginMonitorAgent, error)

	// GetAllAgent returns all agent.
	GetAllAgent(cond *sn.SNCondition) ([]*PluginMonitorAgent, error)

	// WriteFile writes localPath file to agent id's remotePath with perm and timeout.
	//
	// When recursive is true, create folder to satisfy remotePath. When override is true,
	// override remotePath existed file.
	WriteFile(id uuid.UUID, remotePath string, localPath string, recursive bool, override bool,
		perm os.FileMode, timeout time.Duration) error

	// KillCMD kills agent id command cid.
	KillCMD(id uuid.UUID, cid uuid.UUID) error

	// GetCMDRes returns agent id command cid result.
	GetCMDRes(id uuid.UUID, cid uuid.UUID) (*CMDRes, error)

	// RunCMDAsync run asynchronous cmd in agent id, return command uid, output data channel and error immediately.
	//
	// Warning: You must consume data channel ASAP for there are only 60 buffer in it, not consuming data will lead to data loss.
	RunCMDAsync(id uuid.UUID, cmd string) (uuid.UUID, chan string, error)

	// RunCMDSync run synchronous cmd in agent id with timeout, return command uid, output string and error.
	RunCMDSync(id uuid.UUID, cmd string, timeout time.Duration) (uuid.UUID, string, error)

	// GetAgentInstance returns all agent map, changes will not be saved.
	GetAgentInstance() map[uuid.UUID]*AgentInfo
}
