package db

import (
	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/utils/log"
	"github.com/google/uuid"

	"github.com/glebarez/sqlite"
	"github.com/ztrue/tracerr"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type defaultPerm struct {
	ID   sn.DefaultIDIndex
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

// NewDB connect database with config.
func NewDB(dsn string, dbtype string, verbose bool) (*gorm.DB, error) {
	log.New().WithFields(log.F{
		"type": dbtype,
	}).Debug("Connecting to database")

	var l logger.Interface
	if verbose {
		l = logger.Default.LogMode(logger.Info)
	} else {
		l = logger.Default.LogMode(logger.Silent) // disable log
	}

	var dial gorm.Dialector
	switch dbtype {
	case "sqlite":
		dial = sqlite.Open(dsn)
	case "mysql":
		dial = mysql.Open(dsn)
	default:
		log.New().Fatalf("Database type `%v` is not supported", dbtype)
	}

	db, err := gorm.Open(dial, &gorm.Config{
		FullSaveAssociations: false,
		Logger:               l,
	})
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	err = tracerr.Wrap(db.AutoMigrate(new(sn.User), new(sn.Setting), new(sn.Notification), new(sn.Group),
		new(sn.UserGroupLink), new(sn.Permission), new(sn.PermissionLink)))
	if err != nil {
		return nil, err
	}
	if err = dbInit(db); err != nil {
		return nil, err
	}
	log.New().Debug("Database connected")
	return db, nil
}

func dbInit(db *gorm.DB) error {
	var root sn.Group
	var rootPerm sn.PermissionLink

	if err := tracerr.Wrap(db.Where(&sn.Group{Name: "root"}).
		Attrs(&sn.Group{Note: "root"}).
		FirstOrCreate(&root).Error); err != nil {
		return err
	}
	sn.Skynet.ID.Set(sn.GroupRootID, root.ID)
	for _, v := range perm {
		var list sn.Permission
		if err := tracerr.Wrap(db.Where(&sn.Permission{Name: v.Name}).
			Attrs(&sn.Permission{Note: v.Note}).
			FirstOrCreate(&list).Error); err != nil {
			return err
		}
		sn.Skynet.ID.Set(v.ID, list.ID)
	}
	if err := tracerr.Wrap(db.Where(&sn.PermissionLink{GID: uuid.NullUUID{UUID: root.ID, Valid: true},
		PID: sn.Skynet.ID.Get(sn.PermAllID)}).
		Attrs(&sn.PermissionLink{Perm: sn.PermAll}).
		FirstOrCreate(&rootPerm).Error); err != nil {
		return err
	}

	return nil
}
