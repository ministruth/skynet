package config

import (
	"bytes"
	"fmt"
	"os"

	"github.com/MXWXZ/skynet/utils/log"

	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
)

func init() {
	for _, v := range DefaultSetting {
		if v.Value == nil && !v.Required {
			panic(fmt.Sprintf("Config `%v` not valid", v.Name))
		}
		if !v.Required {
			viper.SetDefault(v.Name, v.Value)
		}
	}
}

// Config is config struct for skynet.
// Note that this only defines the default config, please use viper API such as viper.GetString to get runtime config.
type Config struct {
	Name        string    // config name
	Value       any       // config default value
	Required    bool      // force user input
	WarnDefault bool      // show warning if unchanged
	Public      bool      // public to all frontend guest
	Checker     func(any) // config checker
}

var DefaultSetting = []*Config{
	// root
	{Name: "avatar", Value: "default.webp"},
	// csrf
	{Name: "csrf.prefix", Value: "csrf_"},
	{Name: "csrf.expire", Value: 10},
	// redis
	{Name: "redis.dsn", Required: true},
	{Name: "redis.timeout", Value: 10},
	// database
	{Name: "database.type", Required: true},
	{Name: "database.dsn", Required: true},
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
	// session
	{Name: "session.cookie", Value: "GOSESSIONID"},
	{Name: "session.expire", Value: 3600},
	{Name: "session.remember", Value: 5184000},
	{Name: "session.prefix", Value: "session_"},
	{Name: "session.timeout", Value: 10},
	// header
	{Name: "header.csp", Value: "default-src 'none'"},
	// listen
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
	// proxy
	{Name: "proxy.enable", Value: false},
	{Name: "proxy.header", Value: "X-Forwarded-For"},
	{Name: "proxy.trusted", Value: ""},
	// recaptcha
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
}

func Load(path string) error {
	viper.SetConfigType("yml")
	content, err := os.ReadFile(path)
	if err != nil {
		return tracerr.Wrap(err)
	}
	if err = tracerr.Wrap(viper.ReadConfig(bytes.NewBuffer(content))); err != nil {
		return err
	}
	return nil
}

func CheckSetting() {
	for _, v := range DefaultSetting {
		if v.Required && !viper.IsSet(v.Name) {
			log.New().Fatalf("Setting `%v` is not presented", v.Name)
		}
		if v.WarnDefault && viper.Get(v.Name) == v.Value {
			log.New().Warnf("Setting `%v` has default value, please modify your config file for safety", v.Name)
		}
		if v.Checker != nil {
			v.Checker(viper.Get(v.Name))
		}
	}
}
