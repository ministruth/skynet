package cmd

import (
	"fmt"
	"skynet/sn/utils"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of skynet",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Skynet version", utils.VERSION)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
