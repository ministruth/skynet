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
	Set(name string, value string) error
	Delete(name string) error
	Get(name string) (string, bool)
	GetCache() map[string]interface{}
	GetAll(cond *SNCondition) ([]*Setting, error)
}

type SNUser interface {
	New(username string, password string, avatar []byte, role UserRole) (string, error)
	Update(id int, username string, password string, role UserRole, avatar []byte, kick bool) error
	Delete(id int) (bool, error)
	Reset(id int) (string, error)
	ResetAll() (map[string]string, error)
	GetAll(cond *SNCondition) ([]*User, error)
	GetByUsername(username string) (*User, error)
	GetByID(id int) (*User, error)
	Count() (int64, error)
}

type SNNotification interface {
	New(level NotifyLevel, name string, message string) error
	MarkRead(id int) error
	MarkAllRead() error
	Delete(id int) error
	DeleteAll() error
	GetAll(cond *SNCondition) ([]*Notification, error)
	GetByID(id int) (*Notification, error)
	Count(read interface{}) (int64, error)
}

// SNPluginCBType is plugin callback type.
type SNPluginCBType int

const (
	BeforeMiddleware SNPluginCBType = iota // (*gin.Context): invoked as middleware before request, return non-nil to abort.
	AfterMiddleware                        // (*gin.Context): invoked as middleware after request.
)

type SNPlugin interface {
	Count() int
	Enable(id uuid.UUID) error
	Disable(id uuid.UUID) error
	GetAll() interface{}
	Get(id uuid.UUID) interface{}
	Fini()
	New(buf []byte) error

	// Call invoke all plugin cb callback with param
	Call(cb SNPluginCBType, param interface{}) []error

	// Update plugin id with same plugin id package buf.
	// Note that you need to trigger restart if true returned!
	Update(id uuid.UUID, buf []byte) error
	Delete(id uuid.UUID) error
}
