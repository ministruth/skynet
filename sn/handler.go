package sn

import "github.com/google/uuid"

type SNCondition struct {
	Order    []interface{}
	Distinct []interface{}
	Limit    interface{}
	Offset   interface{}
	Where    interface{}
	Args     []interface{}
}

type SNSetting interface {
	New(name string, value string) error
	Update(name string, value string) error
	Delete(name string) error
	Get(name string) (string, bool)
	GetCache() map[string]string
	GetAll(cond *SNCondition) ([]*Settings, error)
}

type SNUser interface {
	New(username string, password string, avatar []byte, role UserRole) (string, error)
	Update(id int, username string, password string, role UserRole, avatar []byte, kick bool) error
	Delete(id int) (bool, error)
	Reset(id int) (string, error)
	ResetAll() (map[string]string, error)
	GetAll(cond *SNCondition) ([]*Users, error)
	GetByUsername(username string) (*Users, error)
	GetByID(id int) (*Users, error)
}

type SNNotification interface {
	New(level NotifyLevel, name string, message string) error
	MarkRead(id int) error
	MarkAllRead() error
	Delete(id int) error
	DeleteAll() error
	GetAll(cond *SNCondition) ([]*Notifications, error)
	GetByID(id int) (*Notifications, error)
	Count(read interface{}) (int64, error)
}

type SNPlugin interface {
	Enable(id uuid.UUID) error
	Disable(id uuid.UUID) error
	GetAll() interface{}
	Get(id uuid.UUID) interface{}
	Fini()
}
