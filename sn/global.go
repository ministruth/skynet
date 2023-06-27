package sn

import (
	"time"

	"github.com/MXWXZ/skynet/utils/tpl"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Version is skynet version.
const Version = "1.0.0"

type DefaultIDIndex = int

const (
	GroupRootID DefaultIDIndex = iota // root user group
	PermAllID                         // full permission
	PermUserID                        // login user permission
	PermGuestID                       // guest permission

	PermManageUserID         // manage user
	PermManageNotificationID // manage notification
	PermManageSystemID       // manage system
	PermManagePluginID       // manage plugin

	DefaultIDMax // max id count
)

type Global struct {
	ID        *tpl.SliceIndex[DefaultIDIndex, uuid.UUID] // skynet default id
	API       API                                        // api
	Menu      Menu                                       // menu
	Engine    *gin.Engine                                // gin engine
	ReCAPTCHA ReCAPTCHA                                  // reCAPTCHA

	// handler
	User         UserHandler         // user
	Group        GroupHandler        // group
	Permission   PermissionHandler   // permission
	Notification NotificationHandler // notification
	Setting      SettingHandler      // setting
	Plugin       Plugin              // plugin

	// db
	DB      *gorm.DB      // db
	Redis   *redis.Client // redis client
	Session Session       // session

	ExitChan  chan bool // send to shutdown
	StartTime time.Time // skynet start time
}

var Skynet Global

func init() {
	Skynet.ExitChan = make(chan bool)
	Skynet.ID = tpl.NewSliceIndex[DefaultIDIndex, uuid.UUID](DefaultIDMax)
}
