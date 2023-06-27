package cmd

import (
	"io"
	"time"

	"github.com/MXWXZ/skynet/config"
	"github.com/MXWXZ/skynet/db"
	"github.com/MXWXZ/skynet/handler"
	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/utils/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "skynet",
	Short: "Service Integration and Management Application",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// init logger
		if logJSON {
			log.SetJSONFormat()
		} else {
			log.SetTextFormat()
		}
		if logStack {
			log.ShowStack()
		}
		if quiet {
			log.SetOutput(io.Discard)
		} else {
			log.SetOutput()
		}
		if verbose {
			logrus.SetLevel(logrus.DebugLevel)
		}
	},
}

var (
	conf      string
	verbose   bool
	pluginDir string
	logJSON   bool
	logStack  bool
	quiet     bool
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&conf, "conf", "c", "conf.yml", "config file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "show verbose")
	rootCmd.PersistentFlags().StringVarP(&pluginDir, "plugin", "p", "plugin", "plugin folder")
	rootCmd.PersistentFlags().BoolVar(&logJSON, "log-json", false, "print log in JSON format")
	rootCmd.PersistentFlags().BoolVar(&logStack, "log-stack", false, "print log with stack")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "disable log")
}

func load(cmd *cobra.Command, args []string) {
	var err error
	// load config
	if err = config.Load(conf); err != nil {
		log.NewEntry(err).Fatal("Failed to load config")
	}

	// connect db
	sn.Skynet.DB, err = db.NewDB(
		viper.GetString("database.dsn"),
		viper.GetString("database.type"),
		verbose)
	if err != nil {
		log.NewEntry(err).Fatal("Failed to connect database")
	}
	// connect redis
	sn.Skynet.Redis, err = db.NewRedis(
		viper.GetString("redis.dsn"),
		time.Duration(viper.GetInt("redis.timeout"))*time.Second)
	if err != nil {
		log.NewEntry(err).Fatal("Failed to connect redis")
	}
	// init session
	sn.Skynet.Session, err = db.NewSession(
		sn.Skynet.Redis,
		viper.GetString("session.prefix"),
		time.Duration(viper.GetInt("session.timeout"))*time.Second)
	if err != nil {
		log.NewEntry(err).Fatal("Failed to initialize session")
	}

	handler.Init()
	log.New().AddHook(handler.NotificationHook{})

	config.CheckSetting() // delay to log in db
}

// Execute executes the root command.
func Execute(args []string) error {
	if args != nil {
		rootCmd.SetArgs(args)
	}
	return rootCmd.Execute()
}
