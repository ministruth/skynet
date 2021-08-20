package sn

import "github.com/gin-gonic/gin"

func init() {
	Skynet.SharedData = make(map[string]interface{})
}

// SNGlobal is global variable for skynet.
type SNGlobal struct {
	Engine       *gin.Engine            // gin engine
	StaticFile   *gin.RouterGroup       // static file router
	API          SNAPI                  // skynet API
	Page         SNPage                 // skynet page
	Plugin       SNPlugin               // skynet plugin
	Setting      SNSetting              // skynet setting
	Notification SNNotification         // skynet notification
	User         SNUser                 // skynet user
	DB           SNDB                   // skynet database
	Redis        SNDB                   // skynet redis
	Session      SNDB                   // skynet session
	SharedData   map[string]interface{} // skynet plugin shared data/API
}

// VERSION is skynet version.
const VERSION = "1.0.0"

// Skynet is global variable object for skynet.
var Skynet SNGlobal
