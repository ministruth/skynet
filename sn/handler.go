package sn

import (
	"fmt"
	"plugin"
	"skynet/sn/utils"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SNCondition limit condition search, however, sqli is not protected.
//
// Unprotected fields: Order, Distinct, Where(when not use ? as argument form)
//
// Warning: Caller should check user input on their OWN!
type SNCondition struct {
	Order    []any
	Distinct []any
	Limit    any
	Offset   any
	Query    string
	Args     []any
}

func (c *SNCondition) newCondition(or bool, query string, args ...any) *SNCondition {
	if c.Query == "" {
		c.Query = fmt.Sprintf("(%v)", query)
		c.Args = args
	} else {
		if or {
			c.Query = fmt.Sprintf("%v OR (%v)", c.Query, query)
		} else {
			c.Query = fmt.Sprintf("%v AND (%v)", c.Query, query)
		}
		c.Args = append(c.Args, args)
	}
	return c
}

func (c *SNCondition) And(query string, args ...any) *SNCondition {
	return c.newCondition(false, query, args...)
}

func (c *SNCondition) Or(query string, args ...any) *SNCondition {
	return c.newCondition(true, query, args...)
}

type SNHandler[T any] interface {
	WithTx(tx *gorm.DB) T
}

type SNHandlerDelete interface {
	Delete(id uuid.UUID) (bool, error)
}

type SNHandlerDeleteAll interface {
	DeleteAll() (int64, error)
}

type SNHandlerGetAll[T DBStruct] interface {
	GetAll(cond *SNCondition) ([]*T, error)
}

type SNHandlerGet[T DBStruct] interface {
	Get(id uuid.UUID) (*T, error)
}

type SNHandlerCount interface {
	Count(cond *SNCondition) (int64, error)
}

type SNSetting interface {
	SNHandler[SNSetting]

	// Set set setting name with value.
	//
	// When error happens, current setting will not change.
	Set(name string, value string) error

	// Delete delete name setting.
	//
	// When error happens, current setting will not change.
	Delete(name string) (bool, error)

	// Get get name setting.
	Get(name string) (string, bool)

	// GetAll return all settings.
	//
	// Copies are returned, modification will not be saved.
	GetAll() map[string]string
}

type SNGroup interface {
	SNHandler[SNGroup]
	SNHandlerGet[UserGroup]
	SNHandlerGetAll[UserGroup]
	SNHandlerCount

	// DeleteGroup delete all user group data and associated permission.
	//
	// Warning: this function will not delete permission.
	SNHandlerDelete

	// New create new usergroup.
	New(name string, note string) (*UserGroup, error)

	// Link link all uid user to all gid group.
	//
	// Note: For performance reasons, this function will not check whether uid or gid is valid.
	Link(uid []uuid.UUID, gid []uuid.UUID) ([]*UserGroupLink, error)

	// Unlink delete user data in user group.
	// When gid is uuid.Nil, delete user in all group.
	// When uid is uuid.Nil, delete group all user.
	//
	// Warning: this function will not delete permission.
	Unlink(uid uuid.UUID, gid uuid.UUID) (int64, error)

	// Update update user group infos, properties remain no change if left empty.
	Update(id uuid.UUID, name string, note *string) error

	// GetGroupAllUser get group id all users.
	GetGroupAllUser(id uuid.UUID) ([]*User, error)

	// GetUserAllGroup get user id all groups.
	GetUserAllGroup(id uuid.UUID) ([]*UserGroup, error)

	// GetByName get group by name.
	//
	// Return nil,nil when group not found.
	GetByName(name string) (*UserGroup, error)
}

type SNUser interface {
	SNHandler[SNUser]
	SNHandlerGet[User]
	SNHandlerGetAll[User]
	SNHandlerCount

	// Delete delete all user data.
	//
	// Warning: this function will not delete permission or unlink group.
	SNHandlerDelete

	// New create new user and return created user and created pass, when password is empty, generate random pass,
	// by default no user group will be attached.
	New(username string, password string, avatar *utils.WebpImage) (*User, string, error)

	// Kick kick user id login.
	Kick(id uuid.UUID) error

	// Update update user infos, properties remain no change if left empty.
	Update(id uuid.UUID, username string, password string,
		avatar *utils.WebpImage, lastTime *time.Time, lastIP string) error

	// Reset reset user password by id, return new password.
	//
	// Return "",nil when user not found.
	Reset(id uuid.UUID) (string, error)

	// GetByName get user by name.
	//
	// Return nil,nil when user not found.
	GetByName(name string) (*User, error)

	// CheckPass check whether user and pass match.
	//
	// If error, return nil,-1,err.
	//
	// If user not found, return nil,1,nil.
	//
	// If pass not match, return nil,2,nil.
	//
	// Return user,0,nil if all match.
	CheckPass(user string, pass string) (*User, int, error)
}

type SNPermission interface {
	SNHandler[SNPermission]

	// AddToGroup add permission to group gid.
	AddToGroup(gid uuid.UUID, perm []*SNPerm) ([]*Permission, error)

	// GetAll find all permission by condition, if join is true,
	// return records will join PermissionList.
	//
	// uid and gid should not be both uuid.Nil or both have value, otherwise,
	// nil,nil will be returned
	GetAll(uid uuid.UUID, gid uuid.UUID, join bool) ([]*Permission, error)

	// DeleteAll delete all uid or gid permission.
	//
	// Note: uid or gid is uuid.Nil means not base on this condition.
	// If both uuid.Nil, do nothing. If both given, delete using OR condition.
	DeleteAll(uid uuid.UUID, gid uuid.UUID) (int64, error)
}

type SNNotification interface {
	SNHandler[SNNotification]
	SNHandlerDelete
	SNHandlerDeleteAll
	SNHandlerGetAll[Notification]
	SNHandlerGet[Notification]
	SNHandlerCount

	New(level NotifyLevel, name string, message string, detail string) error
}

// SNPluginCBType is plugin callback type.
type SNPluginCBType int32

const (
	BeforeMiddleware SNPluginCBType = iota // (*gin.Context): invoked as middleware before request, return non-nil to abort.
	AfterMiddleware                        // (*gin.Context): invoked as middleware after request.

	CallBackMax
)

type SNPluginCallback func(interface{}) error

type SNPluginInfo struct {
	ID            uuid.UUID                           `json:"id"`             // plugin unique ID
	Name          string                              `json:"name"`           // plugin name, unique suggested
	Version       string                              `json:"version"`        // plugin version
	SkynetVersion string                              `json:"skynet_version"` // compatible skynet version
	Callback      map[SNPluginCBType]SNPluginCallback `json:"-"`              // plugin callback
}

type SNPluginEntry struct {
	*SNPluginInfo
	Path      string         `json:"path"`    // runtime absolute path, no ending / unless root
	Enable    bool           `json:"enable"`  // is plugin enabled
	Message   string         `json:"message"` // plugin message
	Interface interface{}    `json:"-"`       // plugin interface
	Loader    *plugin.Plugin `json:"-"`       // golang plugin loader
}

func (p *SNPluginEntry) Disable(msg string) {
	p.Enable = false
	p.Message = msg
}

type SNPlugin interface {
	Count() int
	Enable(id uuid.UUID) error
	Disable(id uuid.UUID) error
	GetAll() []*SNPluginEntry
	Get(id uuid.UUID) *SNPluginEntry
	Fini()
	New(buf []byte) error

	// Call invoke all plugin cb callback with param
	Call(cb SNPluginCBType, param interface{}) []error

	// Update plugin id with same plugin id package buf.
	//
	// Note: need to reload to make effect.
	Update(id uuid.UUID, buf []byte) error

	// Delete plugin id.
	//
	// Note: for enabled plugin, need to reload to make effect.
	Delete(id uuid.UUID) error
}
