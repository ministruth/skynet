package cmd

import (
	"bytes"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "skynet",
	Short: "Service Integration and Management Application",
}

var conf string
var verbose bool

var defaultSettings = map[string]interface{}{
	"redis.address":         "127.0.0.1:6379",
	"redis.password":        "",
	"redis.db":              0,
	"database.type":         "sqlite",
	"database.path":         "data.db",
	"*database.salt_prefix": "en[7",
	"*database.salt_suffix": "z1&.",
	"*csrf_secret":          "01234567890123456789012345678912",
	"listen_addr":           "0.0.0.0:8080",
	"log_file":              "log.txt",
	"session.cookie":        "GOSESSIONID",
	"session.expire":        3600,
	"session.remember":      5184000,
	"session.prefix":        "session_",
	"default_avatar":        "default.webp",
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&conf, "conf", "c", "conf.yml", "config file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "show verbose")

	// init config param
	for k, v := range defaultSettings {
		if k[0] == '*' {
			viper.SetDefault(k[1:], v)
		} else {
			viper.SetDefault(k, v)
		}
	}

	viper.SetConfigType("yml")
	content, err := ioutil.ReadFile(conf)
	if err != nil {
		log.Fatal("Can not read config file: ", err)
	}
	err = viper.ReadConfig(bytes.NewBuffer(content))
	if err != nil {
		log.Fatal("Config file invalid: ", err)
	}

	if verbose {
		log.SetLevel(log.DebugLevel)
	}
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}
