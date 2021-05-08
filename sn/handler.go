package sn

import "github.com/google/uuid"

type SNSetting interface {
	AddSetting(name string, value string) error
	EditSetting(name string, value string) error
	DelSetting(name string) error
	GetSetting(name string) (string, bool)
	Get() map[string]string
}

type SNUser interface {
	AddUser(username string, password string, avatar []byte, role UserRole) (string, error)
	EditUser(id int, username string, password string, role UserRole, avatar []byte, kick bool) error
	DelUser(id int) (bool, error)
	ResetUser(username string) (string, error)
	ResetAllUser() (map[string]string, error)
	GetUser() ([]Users, error)
}

type SNPlugin interface {
	EnablePlugin(id uuid.UUID) error
	DisablePlugin(id uuid.UUID) error
	GetAllPlugin() interface{}
	GetPlugin(id uuid.UUID) interface{}
}
