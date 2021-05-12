package main

import (
	"skynet/sn"
	"time"
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
