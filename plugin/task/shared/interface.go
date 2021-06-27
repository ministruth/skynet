package shared

import (
	"skynet/sn"
)

type TaskStatus int

const (
	TaskNotStart TaskStatus = iota
	TaskRunning
	TaskStop
	TaskSuccess
	TaskFail
)

type PluginTasks struct {
	ID      int32      `gorm:"primaryKey;not null"`
	Name    string     `gorm:"type:varchar(64);not null"`
	Detail  string     `gorm:"type:varchar(1024)"`
	Output  string     `gorm:"type:text"`
	Status  TaskStatus `gorm:"default:0;not null"`
	Percent int32      `gorm:"default:0;not null"`
	Track   sn.Track   `gorm:"embedded"`
}

type PluginShared interface {
	New(name string, detail string, cancel func()) (int, error)
	Cancel(id int)
	Get(id int) (*PluginTasks, error)
	GetAll(order []interface{}, limit interface{}, offset interface{}, where interface{}, args ...interface{}) ([]*PluginTasks, error)
	AppendOutput(id int, out string) error
	AppendOutputNewLine(id int, out string) error
	UpdateOutput(id int, out string) error
	UpdateStatus(id int, status TaskStatus) error
	UpdatePercent(id int, percent int) error
	Count() (int64, error)
	NewCommand(agentID int, cmd string, name string, detail string) (chan bool, error)
}
