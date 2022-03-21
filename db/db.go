package db

import (
	"context"
	"skynet/sn"
	"skynet/sn/utils"

	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type defaultPerm struct {
	ID   sn.DefaultID
	Name string
	Note string
}

var perm = []*defaultPerm{
	{
		ID:   sn.PermAllID,
		Name: "all",
		Note: "all",
	},
	{
		ID:   sn.PermUserID,
		Name: "user",
		Note: "all login user",
	},
	{
		ID:   sn.PermGuestID,
		Name: "guest",
		Note: "all guest user",
	},
	{
		ID:   sn.PermManageUserID,
		Name: "manage.user",
		Note: "user management",
	},
	{
		ID:   sn.PermManageUserPermID,
		Name: "manage.user.perm",
		Note: "user permission management",
	},
	{
		ID:   sn.PermManageGroupID,
		Name: "manage.group",
		Note: "group management",
	},
	{
		ID:   sn.PermManageGroupPermID,
		Name: "manage.group.perm",
		Note: "group permission management",
	},
	{
		ID:   sn.PermManageNotificationID,
		Name: "manage.notification",
		Note: "notification management",
	},
	{
		ID:   sn.PermManageSystemID,
		Name: "manage.system",
		Note: "system management",
	},
	{
		ID:   sn.PermManagePluginID,
		Name: "manage.plugin",
		Note: "plugin management",
	},
}

// DBType presents backend database type.
type DBType int32

const (
	DBType_Sqlite DBType = iota // sqlite backend
)

// DBConfig is connection config for database.
type DBConfig struct {
	Type DBType // backend database type
	Path string // connect url/file path
}

type dbClient struct {
	dbClient *gorm.DB
}

// NewDB create new database object, exit when facing any error.
func NewDB(ctx context.Context, conf *DBConfig) sn.SNDB[*gorm.DB] {
	var ret dbClient
	var err error

	switch conf.Type {
	case DBType_Sqlite:
		var l logger.Interface
		if viper.GetBool("debug") {
			l = logger.Default.LogMode(logger.Info)
		} else {
			l = logger.Default.LogMode(logger.Silent) // disable log
		}
		ret.dbClient, err = gorm.Open(sqlite.Open(conf.Path), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
			FullSaveAssociations:                     false,
			Logger:                                   l,
		})
		if err != nil {
			utils.WithTrace(err).Fatal("Failed to connect sqlite database: ", err)
		}
		err = ret.dbClient.AutoMigrate(new(sn.User), new(sn.Setting), new(sn.Notification), new(sn.UserGroup),
			new(sn.UserGroupLink), new(sn.Permission), new(sn.PermissionList))
		if err != nil {
			utils.WithTrace(err).Fatal("Failed to connect sqlite database: ", err)
		}
	}
	if err = ret.Init(); err != nil {
		utils.WithTrace(err).Fatal("Failed to connect sqlite database: ", err)
	}
	return &ret
}

func (c *dbClient) Get() *gorm.DB {
	if c.dbClient == nil {
		panic("DB not init")
	}
	return c.dbClient
}

func (c *dbClient) Init() error {
	var root sn.UserGroup
	var rootPerm sn.Permission
	if err := tracerr.Wrap(c.dbClient.Where(&sn.UserGroup{Name: "root"}).
		Attrs(&sn.UserGroup{Note: "root"}).
		FirstOrCreate(&root).Error); err != nil {
		return err
	}
	sn.Skynet.SetID(sn.GroupRootID, root.ID)
	for _, v := range perm {
		var list sn.PermissionList
		if err := tracerr.Wrap(c.dbClient.Where(&sn.PermissionList{Name: v.Name}).
			Attrs(&sn.PermissionList{Note: v.Note}).
			FirstOrCreate(&list).Error); err != nil {
			return err
		}
		sn.Skynet.SetID(v.ID, list.ID)
	}
	if err := tracerr.Wrap(c.dbClient.Where(&sn.Permission{GID: root.ID, PID: sn.Skynet.GetID(sn.PermAllID)}).
		Attrs(&sn.Permission{Perm: sn.PermAll}).
		FirstOrCreate(&rootPerm).Error); err != nil {
		return err
	}
	return nil
}
