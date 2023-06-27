package sn

import (
	"github.com/google/uuid"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
)

type UserPerm = int32

const (
	PermNone    UserPerm = 0            // default, no permission
	PermExecute UserPerm = 1            // execute
	PermWrite   UserPerm = 1 << 1       // write database
	PermRead    UserPerm = 1 << 2       // read
	PermAll     UserPerm = (1 << 3) - 1 // all permission

	PermWriteExecute UserPerm = PermWrite | PermExecute // write and execute
)

type NotifyLevel = int32

const (
	NotifyInfo NotifyLevel = iota
	NotifySuccess
	NotifyWarning
	NotifyError
	NotifyFatal
)

// DBStruct identify database struct.
type DBStruct interface {
	ValidDBStruct()
}

type GeneralFields struct {
	ID        uuid.UUID `gorm:"type:char(36);primaryKey;not null" json:"id"`
	CreatedAt int64     `gorm:"autoCreateTime:milli" json:"created_at"` // create time
	UpdatedAt int64     `gorm:"autoUpdateTime:milli" json:"updated_at"` // update time
}

func (u *GeneralFields) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

func (u GeneralFields) ValidDBStruct() {}

type User struct {
	GeneralFields
	Username  string `gorm:"uniqueIndex;type:varchar(32);not null;default:null" json:"username"`
	Password  string `gorm:"type:char(32);not null;default:null" json:"-"`
	Avatar    string `gorm:"type:string" json:"avatar"`
	LastLogin int64  `json:"last_login"`
	LastIP    string `gorm:"type:varchar(64)" json:"last_ip"`
}

type Group struct {
	GeneralFields
	Name string `gorm:"type:varchar(32);uniqueIndex;not null;default:null" json:"name"`
	Note string `gorm:"type:varchar(256)" json:"note"`
}

type UserGroupLink struct {
	GeneralFields
	UID   uuid.UUID `gorm:"column:uid;type:char(36);uniqueIndex:ug_link;not null;default:null" json:"uid"`
	GID   uuid.UUID `gorm:"column:gid;type:char(36);uniqueIndex:ug_link;not null;default:null" json:"gid"`
	User  *User     `gorm:"foreignKey:UID;constraint:OnUpdate:RESTRICT,OnDelete:CASCADE" json:"-"`
	Group *Group    `gorm:"foreignKey:GID;constraint:OnUpdate:RESTRICT,OnDelete:CASCADE" json:"-"`
}

type Permission struct {
	GeneralFields
	Name string `gorm:"uniqueIndex;type:varchar(128);not null;default:null" json:"name"`
	Note string `gorm:"type:varchar(256)" json:"note"`
}

type PermissionLink struct {
	GeneralFields
	UID        uuid.NullUUID `gorm:"column:uid;type:char(36);uniqueIndex:perm_link" json:"uid"`
	GID        uuid.NullUUID `gorm:"column:gid;type:char(36);uniqueIndex:perm_link" json:"gid"`
	PID        uuid.UUID     `gorm:"column:pid;type:char(36);uniqueIndex:perm_link;not null;default:null" json:"pid"`
	Perm       UserPerm      `gorm:"default:0;not null" json:"perm"`
	User       *User         `gorm:"foreignKey:UID;constraint:OnUpdate:RESTRICT,OnDelete:CASCADE" json:"-"`
	Group      *Group        `gorm:"foreignKey:GID;constraint:OnUpdate:RESTRICT,OnDelete:CASCADE" json:"-"`
	Permission *Permission   `gorm:"foreignKey:PID;constraint:OnUpdate:RESTRICT,OnDelete:CASCADE" json:"-"`
}

func (u *PermissionLink) BeforeCreate(tx *gorm.DB) error {
	if !u.UID.Valid && !u.GID.Valid {
		return tracerr.New("uid and gid can not be both nil")
	}
	if u.UID.Valid && u.GID.Valid {
		return tracerr.New("uid and gid can not be both filled")
	}
	return u.GeneralFields.BeforeCreate(tx)
}

func (u *PermissionLink) BeforeUpdate(tx *gorm.DB) error {
	if !u.UID.Valid && !u.GID.Valid {
		return tracerr.New("uid and gid can not be both nil")
	}
	if u.UID.Valid && u.GID.Valid {
		return tracerr.New("uid and gid can not be both filled")
	}
	return u.GeneralFields.BeforeCreate(tx)
}

type Setting struct {
	GeneralFields
	Name  string `gorm:"uniqueIndex;type:varchar(256);not null;default:null" json:"name"`
	Value string `gorm:"type:string" json:"value"`
}

type Notification struct {
	GeneralFields
	Level   NotifyLevel `gorm:"default:0;not null" json:"level"`
	Name    string      `gorm:"type:varchar(256)" json:"name"`
	Message string      `gorm:"type:varchar(256)" json:"message"`
	Detail  string      `gorm:"type:string" json:"detail"`
}
