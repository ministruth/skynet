package sn

import (
	"time"
)

type Track struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserRole int

const (
	RoleEmpty UserRole = iota
	RoleUser
	RoleAdmin
)

type Users struct {
	ID        int32    `gorm:"primaryKey;not null"`
	Username  string   `gorm:"uniqueIndex;type:varchar(32);not null"`
	Password  string   `gorm:"type:char(32);not null" json:"-"`
	Avatar    []byte   `gorm:"type:bytes;not null" json:"-"`
	Role      UserRole `gorm:"default:1;not null"`
	LastLogin time.Time
	LastIP    string `gorm:"type:varchar(64)"`
	Track     Track  `gorm:"embedded"`
}

type Settings struct {
	ID    int32  `gorm:"primaryKey;not null"`
	Name  string `gorm:"uniqueIndex;type:varchar(256);not null"`
	Value string `gorm:"type:varchar(1024);not null"`
	Track Track  `gorm:"embedded"`
}

type NotifyLevel int

const (
	NotifyInfo NotifyLevel = iota
	NotifySuccess
	NotifyWarning
	NotifyError
	NotifyFatal
)

type Notifications struct {
	ID      int32       `gorm:"primaryKey;not null"`
	Level   NotifyLevel `gorm:"default:0;not null"`
	Name    string      `gorm:"type:varchar(256)"`
	Message string      `gorm:"type:varchar(1024)"`
	Read    int32       `gorm:"default:0;not null"`
	Track   Track       `gorm:"embedded"`
}
