package sn

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserHandler interface {
	WithTx(tx *gorm.DB) UserHandler

	// New create new user, by default no user group will be attached.
	//
	// When avatar is nil, use default avatar.
	New(username string, password string, avatar string) (*User, error)

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

	// GetAll get all user by condition.
	GetAll(cond *Condition) ([]*User, error)

	// Get get user by id.
	Get(id uuid.UUID) (*User, error)

	// GetByName get user by name.
	//
	// Return nil,nil when user not found.
	GetByName(name string) (*User, error)

	// Count count user by condition.
	Count(cond *Condition) (int64, error)

	// Kick kick user id login.
	Kick(id uuid.UUID) error

	// Reset reset user password by id, return new password.
	Reset(id uuid.UUID) (string, error)

	// Update update user infos by user.ID.
	// Password will be hashed after the function call.
	// Users will be kicked if password is updated.
	Update(column []string, user *User) error

	// Delete delete the entire user, including link and permission.
	Delete(id uuid.UUID) error
}

type GroupHandler interface {
	WithTx(tx *gorm.DB) GroupHandler

	// New create new usergroup.
	New(name string, note string) (*Group, error)

	// Link link all uid user to all gid group.
	Link(uid []uuid.UUID, gid []uuid.UUID) ([]*UserGroupLink, error)

	// Unlink unlink user and group.
	//
	// If uid!=nil and gid!=nil, remove each uid with each gid.
	// If uid==nil, remove all users in each gid.
	// If gid==nil, remove all groups in each uid.
	Unlink(uid []uuid.UUID, gid []uuid.UUID) error

	// GetGroupAllUser get group id all users.
	GetGroupAllUser(id uuid.UUID, cond *Condition) ([]*UserGroupLink, error)

	// CountGroupAllUser count group user by condition.
	CountGroupAllUser(id uuid.UUID, cond *Condition) (int64, error)

	// GetAll get all group by condition.
	GetAll(cond *Condition) ([]*Group, error)

	// Get get group by id.
	Get(id uuid.UUID) (*Group, error)

	// GetByName get group by name.
	GetByName(name string) (*Group, error)

	// GetUserAllGroup get user id all groups.
	GetUserAllGroup(id uuid.UUID) ([]*UserGroupLink, error)

	// Count count group by condition.
	Count(cond *Condition) (int64, error)

	// Update update group infos by group.ID.
	Update(column []string, group *Group) error

	// Delete delete the entire group, including link and permission.
	Delete(id uuid.UUID) error
}

type PermissionHandler interface {
	WithTx(tx *gorm.DB) PermissionHandler

	// AddToUser add permission to user uid.
	AddToUser(uid uuid.UUID, perm []*PermEntry) (ret []*PermissionLink, err error)

	// AddToGroup add permission to group gid.
	AddToGroup(gid uuid.UUID, perm []*PermEntry) (ret []*PermissionLink, err error)

	// GetEntry return permission entries.
	GetEntry() ([]*Permission, error)

	// Grant grant uid or gid with pid and perm.
	//
	// uid and gid should not be both uuid.Nil, otherwise, nil will be returned.
	// If uid and gid both have value, pid will be granted to all of them.
	//
	// If perm==0, permission will be explicitly forbidden.
	// If perm==-1, permission will be revoked.
	Grant(uid uuid.UUID, gid uuid.UUID, pid uuid.UUID, perm UserPerm) error

	// GetAll find all permission by condition, if join is true,
	// return records will join PermissionList.
	//
	// uid and gid should not be both uuid.Nil or both have value, otherwise,
	// nil,nil will be returned
	GetAll(uid uuid.UUID, gid uuid.UUID, joinUser bool, joinGroup bool, joinPerm bool) ([]*PermissionLink, error)

	// GetUser get user perm list.
	GetUser(uid uuid.UUID) (map[uuid.UUID]*PermEntry, error)

	// GetGroup get group perm list.
	GetGroup(gid uuid.UUID) (map[uuid.UUID]*PermEntry, error)

	// GetUserMerged returns merged user permission list.
	GetUserMerged(uid uuid.UUID) (map[uuid.UUID]*PermEntry, error)

	// Delete delete by permission link id.
	Delete(id uuid.UUID) error

	// DeleteUser delete user uid permission pid.
	//
	// If pid == uuid.Nil, all user permission will be deleted.
	DeleteUser(uid uuid.UUID, pid uuid.UUID) (int64, error)

	// DeleteGroup delete group gid permission pid.
	//
	// If pid == uuid.Nil, all group permission will be deleted.
	DeleteGroup(gid uuid.UUID, pid uuid.UUID) (int64, error)
}

type NotificationHandler interface {
	WithTx(tx *gorm.DB) NotificationHandler
	SetUnread(num int64)
	GetUnread() int64
	New(level NotifyLevel, name string, message string, detail string) error
	GetAll(cond *Condition) ([]*Notification, error)
	Get(id uuid.UUID) (*Notification, error)
	Count(cond *Condition) (int64, error)
	Delete(id uuid.UUID) (bool, error)
	DeleteAll() (int64, error)
}

type SettingHandler interface {
	WithTx(tx *gorm.DB) SettingHandler
	BuildCache() error

	// GetAll return all settings.
	//
	// Copies are returned, modification will not be saved.
	GetAll() map[string]string

	// Get get name setting.
	Get(name string) (string, bool)

	// Set set setting name with value.
	Set(name string, value string) error

	// Delete delete name setting.
	Delete(name string) (bool, error)

	// DeleteAll delete all settings.
	DeleteAll() (int64, error)
}
