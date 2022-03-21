package sn

import (
	"github.com/google/uuid"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
)

// DBStruct identify database struct.
type DBStruct interface {
	ValidDBStruct()
}

type GeneralFields struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;not null" json:"id"`
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

type UserGroup struct {
	GeneralFields
	Name string `gorm:"type:varchar(32);uniqueIndex;not null" json:"name"`
	Note string `gorm:"type:varchar(256)" json:"note"`
}

type UserGroupLink struct {
	GeneralFields
	UID       uuid.UUID  `gorm:"column:uid;type:uuid;uniqueIndex:ugid_link;not null" json:"uid"`
	GID       uuid.UUID  `gorm:"column:gid;type:uuid;uniqueIndex:ugid_link;not null" json:"gid"`
	User      *User      `gorm:"foreignKey:UID" json:"-"`
	UserGroup *UserGroup `gorm:"foreignKey:GID" json:"-"`
}

type PermissionList struct {
	GeneralFields
	Name string `gorm:"uniqueIndex;type:varchar(128);not null" json:"name"`
	Note string `gorm:"type:varchar(256)" json:"note"`
}

type Permission struct {
	GeneralFields
	UID            uuid.UUID       `gorm:"column:uid;type:uuid;uniqueIndex:perm_link" json:"uid"`
	GID            uuid.UUID       `gorm:"column:gid;type:uuid;uniqueIndex:perm_link" json:"gid"`
	PID            uuid.UUID       `gorm:"column:pid;type:uuid;uniqueIndex:perm_link;not null" json:"pid"`
	Perm           UserPerm        `gorm:"default:0;not null" json:"perm"`
	User           *User           `gorm:"foreignKey:UID" json:"-"`
	UserGroup      *UserGroup      `gorm:"foreignKey:GID" json:"-"`
	PermissionList *PermissionList `gorm:"foreignKey:PID" json:"-"`
}

func (u *Permission) BeforeCreate(tx *gorm.DB) error {
	if u.UID == uuid.Nil && u.GID == uuid.Nil {
		return tracerr.New("uid and gid can not be both nil")
	}
	if u.UID != uuid.Nil && u.GID != uuid.Nil {
		return tracerr.New("uid and gid can not be both filled")
	}
	return u.GeneralFields.BeforeCreate(tx)
}

func (u *Permission) BeforeUpdate(tx *gorm.DB) error {
	if u.UID == uuid.Nil && u.GID == uuid.Nil {
		return tracerr.New("uid and gid can not be both nil")
	}
	if u.UID != uuid.Nil && u.GID != uuid.Nil {
		return tracerr.New("uid and gid can not be both filled")
	}
	return u.GeneralFields.BeforeCreate(tx)
}

type User struct {
	GeneralFields
	Username  string `gorm:"uniqueIndex;type:varchar(32);not null" json:"username"`
	Password  string `gorm:"type:char(32);not null" json:"-"`
	Avatar    []byte `gorm:"type:bytes;not null" json:"avatar"`
	LastLogin int64  `json:"last_login"`
	LastIP    string `gorm:"type:varchar(64)" json:"last_ip"`
}

type Setting struct {
	GeneralFields
	Name  string `gorm:"uniqueIndex;type:varchar(256);not null" json:"name"`
	Value string `gorm:"type:string" json:"value"`
}

type NotifyLevel int32

const (
	NotifyInfo NotifyLevel = iota
	NotifySuccess
	NotifyWarning
	NotifyError
	NotifyFatal
)

type Notification struct {
	GeneralFields
	Level   NotifyLevel `gorm:"default:0;not null" json:"level"`
	Name    string      `gorm:"type:varchar(256)" json:"name"`
	Message string      `gorm:"type:varchar(256)" json:"message"`
	Detail  string      `gorm:"type:string" json:"detail"`
}
