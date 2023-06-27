package cmd

import (
	"fmt"

	"github.com/MXWXZ/skynet/sn"
	"github.com/spf13/cobra"
)

var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Show skynet version",
		Run:   version,
	}
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

func version(cmd *cobra.Command, args []string) {
	fmt.Printf("skynet version %v\n", sn.Version)
}
