package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const VERSION = "v1.0.0-dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of skynet",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Skynet version", VERSION)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
