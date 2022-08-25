package db

import (
	"github.com/MXWXZ/skynet/utils/log"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DefaultID []uuid.UUID // skynet default id

func init() {
	DefaultID = make([]uuid.UUID, DefaultIDMax)
}

func SetDefaultID(index DefaultIDIndex, value uuid.UUID) {
	DefaultID[index] = value
}

func GetDefaultID(index DefaultIDIndex) uuid.UUID {
	return DefaultID[index]
}

type DefaultIDIndex int32

const (
	GroupRootID DefaultIDIndex = iota // root user group
	PermAllID                         // full permission
	PermUserID                        // login user permission
	PermGuestID                       // guest permission

	PermManageUserID         // manage user
	PermManageUserPermID     // manage user permission
	PermManageGroupID        // manage user group
	PermManageGroupPermID    // manage user group
	PermManageNotificationID // manage notification
	PermManageSystemID       // manage system
	PermManagePluginID       // manage plugin

	DefaultIDMax // max id count
)

type defaultPerm struct {
	ID   DefaultIDIndex
	Name string
	Note string
}

var perm = []*defaultPerm{
	{
		ID:   PermAllID,
		Name: "all",
		Note: "all",
	},
	{
		ID:   PermUserID,
		Name: "user",
		Note: "all login user",
	},
	{
		ID:   PermGuestID,
		Name: "guest",
		Note: "all guest/login user",
	},
	{
		ID:   PermManageUserID,
		Name: "manage.user",
		Note: "user management",
	},
	{
		ID:   PermManageUserPermID,
		Name: "manage.user.perm",
		Note: "user permission management",
	},
	{
		ID:   PermManageGroupID,
		Name: "manage.group",
		Note: "group management",
	},
	{
		ID:   PermManageGroupPermID,
		Name: "manage.group.perm",
		Note: "group permission management",
	},
	{
		ID:   PermManageNotificationID,
		Name: "manage.notification",
		Note: "notification management",
	},
	{
		ID:   PermManageSystemID,
		Name: "manage.system",
		Note: "system management",
	},
	{
		ID:   PermManagePluginID,
		Name: "manage.plugin",
		Note: "plugin management",
	},
}

var DB *gorm.DB

// NewDB connect database with config.
func NewDB() {
	path := viper.GetString("database.path")
	dbtype := viper.GetString("database.type")
	log.New().WithFields(log.F{
		"path": path,
		"type": dbtype,
	}).Debug("Connecting to database")

	var err error
	switch dbtype {
	case "sqlite":
		var l logger.Interface
		if viper.GetBool("debug") {
			l = logger.Default.LogMode(logger.Info)
		} else {
			l = logger.Default.LogMode(logger.Silent) // disable log
		}
		DB, err = gorm.Open(sqlite.Open(path), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
			FullSaveAssociations:                     false,
			Logger:                                   l,
		})
		if err != nil {
			log.NewEntry(tracerr.Wrap(err)).Fatal("Failed to connect sqlite database")
		}
		err = DB.AutoMigrate(new(User), new(Setting), new(Notification), new(UserGroup),
			new(UserGroupLink), new(Permission), new(PermissionList))
		if err != nil {
			log.NewEntry(tracerr.Wrap(err)).Fatal("Failed to migrate sqlite database")
		}
	default:
		log.New().Fatalf("Database type %s not supported", dbtype)
	}

	if err = dbInit(); err != nil {
		log.NewEntry(err).Fatal("Failed to init sqlite database")
	}
	log.New().Debug("Database connected")
}

func dbInit() error {
	var root UserGroup
	var rootPerm Permission
	if err := tracerr.Wrap(DB.Where(&UserGroup{Name: "root"}).
		Attrs(&UserGroup{Note: "root"}).
		FirstOrCreate(&root).Error); err != nil {
		return err
	}
	SetDefaultID(GroupRootID, root.ID)
	for _, v := range perm {
		var list PermissionList
		if err := tracerr.Wrap(DB.Where(&PermissionList{Name: v.Name}).
			Attrs(&PermissionList{Note: v.Note}).
			FirstOrCreate(&list).Error); err != nil {
			return err
		}
		SetDefaultID(v.ID, list.ID)
	}
	if err := tracerr.Wrap(DB.Where(&Permission{GID: root.ID, PID: GetDefaultID(PermAllID)}).
		Attrs(&Permission{Perm: PermAll}).
		FirstOrCreate(&rootPerm).Error); err != nil {
		return err
	}
	return nil
}
