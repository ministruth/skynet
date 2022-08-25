package config

import (
	"bytes"
	"io/ioutil"

	"github.com/MXWXZ/skynet/utils/log"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
)

func init() {
	for _, v := range DefaultSetting {
		viper.SetDefault(v.Name, v.Value)
	}
}

// Config is config struct for skynet.
type Config struct {
	Name        string    // config name
	Value       any       // config default value
	WarnDefault bool      // show warning if unchanged
	Public      bool      // public to all frontend guest
	Checker     func(any) // config checker
}

var DefaultSetting = []*Config{
	{Name: "debug", Value: false, Checker: func(v any) {
		if !v.(bool) {
			gin.SetMode(gin.ReleaseMode)
		} else {
			log.New().Warn("Debug mode is on, make it off when put into production")
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
			log.New().Warn("salt_prefix too short, make it longer")
		}
	}},
	{Name: "database.salt_suffix", Value: "z1&.", WarnDefault: true, Checker: func(v any) {
		if len(v.(string)) < 4 {
			log.New().Warn("salt_suffix too short, make it longer")
		}
	}},
	{Name: "listen.address", Value: "0.0.0.0:8080"},
	{Name: "listen.allowhosts", Value: "", WarnDefault: true},
	{Name: "listen.ssl", Value: false, Checker: func(v any) {
		if !v.(bool) {
			log.New().Warn("ssl is disabled, enable it when put into production")
		} else {
			if viper.GetString("listen.ssl_cert") == "" || viper.GetString("listen.ssl_key") == "" {
				log.New().Fatal("ssl_cert or ssl_key not provided")
			}
		}
	}},
	{Name: "listen.ssl_cert", Value: ""},
	{Name: "listen.ssl_key", Value: ""},
	{Name: "log.console", Value: true},
	{Name: "log.file", Value: ""},
	{Name: "log.json", Value: false},
	{Name: "log.stack", Value: false},
	{Name: "session.cookie", Value: "GOSESSIONID"},
	{Name: "session.expire", Value: 3600},
	{Name: "session.remember", Value: 5184000},
	{Name: "session.prefix", Value: "session_"},
	{Name: "session.timeout", Value: 3},
	{Name: "default_avatar", Value: "default.webp"},
	{Name: "proxy.enable", Value: false, Checker: func(v any) {
		if v.(bool) {
			log.New().Warn("proxy is enabled, disable it when not behind proxy to prevent IP spoofing")
			if viper.GetString("proxy.header") == "" {
				log.New().Fatal("proxy header not provided")
			}
		}
	}},
	{Name: "proxy.header", Value: "X-Real-IP"},
	{Name: "proxy.trusted", Value: ""},
	{Name: "recaptcha.enable", Value: false, Public: true, Checker: func(v any) {
		if !v.(bool) {
			log.New().Warn("reCAPTCHA is disabled, enable it when put into production")
		} else {
			if viper.GetString("recaptcha.sitekey") == "" || viper.GetString("recaptcha.secret") == "" {
				log.New().Fatal("sitekey or secret not provided")
			}
		}
	}},
	{Name: "recaptcha.cnmirror", Value: false, Public: true},
	{Name: "recaptcha.sitekey", Value: "", Public: true},
	{Name: "recaptcha.secret", Value: ""},
	{Name: "recaptcha.timeout", Value: 10},
	{Name: "csrf.expire", Value: 10},
	{Name: "csrf.prefix", Value: "csrf_"},
	{Name: "csrf.timeout", Value: 3},
	{Name: "plugin.timeout", Value: 60},
}

func Load(path string, debug bool) error {
	viper.SetConfigType("yml")
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return tracerr.Wrap(err)
	}
	if err = viper.ReadConfig(bytes.NewBuffer(content)); err != nil {
		return tracerr.Wrap(err)
	}

	if debug || viper.GetBool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
	}
	return nil
}

func CheckSetting() {
	for _, v := range DefaultSetting {
		if v.WarnDefault && viper.Get(v.Name) == v.Value {
			log.New().Warnf("Setting %v has default value, please modify your config file for safety", v.Name)
		}
		if v.Checker != nil {
			v.Checker(viper.Get(v.Name))
		}
	}
}
