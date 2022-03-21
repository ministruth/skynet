package cmd

import (
	"bytes"
	"io/ioutil"
	"skynet/sn/utils"

	_ "skynet/handler"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "skynet",
	Short: "Service Integration and Management Application",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		viper.SetConfigType("yml")
		content, err := ioutil.ReadFile(conf)
		if err != nil {
			utils.WithTrace(err).Fatal(err)
		}
		if err = viper.ReadConfig(bytes.NewBuffer(content)); err != nil {
			utils.WithTrace(err).Fatal(err)
		}

		if verbose {
			log.SetLevel(log.DebugLevel)
		}
	},
}

var conf string
var verbose bool

func init() {
	rootCmd.PersistentFlags().StringVarP(&conf, "conf", "c", "conf.yml", "config file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "show verbose")
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}
