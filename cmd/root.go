package cmd

import (
	"skynet/config"
	"skynet/utils/log"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "skynet",
	Short: "Service Integration and Management Application",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := config.Load(conf, verbose)
		if err != nil {
			log.NewEntry(err).Fatal("Failed to load config")
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
func Execute(args []string) error {
	if args != nil {
		rootCmd.SetArgs(args)
	}
	return rootCmd.Execute()
}
