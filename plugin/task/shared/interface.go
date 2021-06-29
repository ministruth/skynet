package shared

import (
	"context"
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

type PluginTask struct {
	ID      int32      `gorm:"primaryKey;not null"`
	Name    string     `gorm:"type:varchar(64);not null"`
	Detail  string     `gorm:"type:varchar(1024)"`
	Output  string     `gorm:"type:text"`
	Status  TaskStatus `gorm:"default:0;not null"`
	Percent int32      `gorm:"default:0;not null"`
	Track   sn.Track   `gorm:"embedded"`
}

type PluginShared interface {
	New(name string, detail string, cancel func() error) (int, error)
	CancelByUser(id int, msg string) error
	Cancel(id int, msg string) error
	Get(id int) (*PluginTask, error)
	GetAll(order []interface{}, limit interface{}, offset interface{}, where interface{}, args ...interface{}) ([]*PluginTask, error)
	AppendOutput(id int, out string) error
	AppendOutputNewLine(id int, out string) error
	UpdateOutput(id int, out string) error
	UpdateStatus(id int, status TaskStatus) error
	AddPercent(id int, percent int) error
	UpdatePercent(id int, percent int) error
	Count() (int64, error)
	NewCommand(agentID int, cmd string, name string, detail string) (chan bool, error)
	NewCustom(agentID int, name string, detail string, c func() error, f func(ctx context.Context, agentID int, taskID int) error) error
}
