package main

import (
	"fmt"
	"skynet/plugin/monitor/shared"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of agent",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Agent version", shared.AgentVersion)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
