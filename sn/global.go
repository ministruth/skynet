package sn

import (
	"skynet/sn/tpl"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rbcervilla/redisstore/v8"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

var DefaultSetting = []*SNConfig{
	{Name: "debug", Value: false, Checker: func(v any) {
		if !v.(bool) {
			gin.SetMode(gin.ReleaseMode)
		} else {
			log.Warn("Debug mode is on, make it off when put into production")
		}
	}},
	{Name: "redis.address", Value: "127.0.0.1:6379"},
	{Name: "redis.password", Value: ""},
	{Name: "redis.db", Value: 0},
	{Name: "redis.timeout", Value: 30},
	{Name: "database.type", Value: "sqlite"},
	{Name: "database.path", Value: "data.db"},
	{Name: "database.salt_prefix", Value: "en[7", WarnDefault: true, Checker: func(v any) {
		if len(v.(string)) < 4 {
			log.Warn("salt_prefix too short, make it longer")
		}
	}},
	{Name: "database.salt_suffix", Value: "z1&.", WarnDefault: true, Checker: func(v any) {
		if len(v.(string)) < 4 {
			log.Warn("salt_suffix too short, make it longer")
		}
	}},
	{Name: "database.timeout", Value: 30},
	{Name: "listen.address", Value: "0.0.0.0:8080"},
	{Name: "listen.allowhosts", Value: "", WarnDefault: true},
	{Name: "listen.ssl", Value: false, Checker: func(v any) {
		if !v.(bool) {
			log.Warn("ssl is disabled, enable it when put into production")
		} else {
			if viper.GetString("listen.ssl_cert") == "" || viper.GetString("listen.ssl_key") == "" {
				log.Fatal("ssl_cert or ssl_key not provided")
			}
		}
	}},
	{Name: "listen.ssl_cert", Value: ""},
	{Name: "listen.ssl_key", Value: ""},
	{Name: "log_file", Value: ""},
	{Name: "session.cookie", Value: "GOSESSIONID"},
	{Name: "session.expire", Value: 3600},
	{Name: "session.remember", Value: 5184000},
	{Name: "session.prefix", Value: "session_"},
	{Name: "session.timeout", Value: 3},
	{Name: "default_avatar", Value: "default.webp"},
	{Name: "proxy.enable", Value: false, Checker: func(v any) {
		if v.(bool) {
			log.Warn("proxy is enabled, disable it when not behind proxy to prevent IP spoofing")
			if viper.GetString("proxy.header") == "" {
				log.Fatal("proxy header not provided")
			}
		}
	}},
	{Name: "proxy.header", Value: "X-Real-IP"},
	{Name: "recaptcha.enable", Value: false, Public: true, Checker: func(v any) {
		if !v.(bool) {
			log.Warn("reCAPTCHA is disabled, enable it when put into production")
		} else {
			if viper.GetString("recaptcha.sitekey") == "" || viper.GetString("recaptcha.secret") == "" {
				log.Fatal("sitekey or secret not provided")
			}
		}
	}},
	{Name: "recaptcha.cnmirror", Value: false, Public: true},
	{Name: "recaptcha.sitekey", Value: "", Public: true},
	{Name: "recaptcha.secret", Value: ""},
	{Name: "csrf.expire", Value: 10},
	{Name: "csrf.prefix", Value: "csrf_"},
	{Name: "csrf.timeout", Value: 3},
}

func init() {
	Skynet.ID = make([]uuid.UUID, DefaultIDMax)
	Skynet.PermList = new(tpl.SafeMap[uuid.UUID, *SNPermList])
	Skynet.SharedData = new(tpl.SafeMap[string, any])
	for _, v := range DefaultSetting {
		viper.SetDefault(v.Name, v.Value)
	}
}

// SNConfig is config struct for skynet.
type SNConfig struct {
	Name        string    // config name
	Value       any       // config default value
	WarnDefault bool      // show warning if unchanged
	Public      bool      // public to all guest
	Checker     func(any) // config checker
}

// SNGlobal is global variable for skynet.
type SNGlobal struct {
	ID           []uuid.UUID                          // skynet default id
	Translator   *i18n.Bundle                         // skynet i18n
	Engine       *gin.Engine                          // gin engine
	API          SNAPI                                // skynet API
	Plugin       SNPlugin                             // skynet plugin
	Setting      SNSetting                            // skynet setting
	Notification SNNotification                       // skynet notification
	User         SNUser                               // skynet user
	Group        SNGroup                              // skynet group
	Permission   SNPermission                         // skynet permission
	PermList     *tpl.SafeMap[uuid.UUID, *SNPermList] // skynet permission list
	DB           SNDB[*gorm.DB]                       // skynet database
	Redis        SNDB[*redis.Client]                  // skynet redis
	Session      SNDB[*redisstore.RedisStore]         // skynet session
	SharedData   *tpl.SafeMap[string, any]            // skynet plugin shared data/API
	Running      bool                                 // true when skynet running, false when restart scheduled
	StartTime    time.Time                            // skynet start time
}

func (s *SNGlobal) GetID(id DefaultID) uuid.UUID {
	return s.ID[id]
}

func (s *SNGlobal) SetID(id DefaultID, value uuid.UUID) {
	s.ID[id] = value
}

func (s *SNGlobal) GetDB() *gorm.DB {
	return s.DB.Get()
}

func (s *SNGlobal) GetRedis() *redis.Client {
	return s.Redis.Get()
}

func (s *SNGlobal) GetSession() *redisstore.RedisStore {
	return s.Session.Get()
}

// VERSION is skynet version.
const VERSION = "1.0.0"

// Skynet is global variable object for skynet.
var Skynet SNGlobal
