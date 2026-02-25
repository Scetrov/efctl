package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Set via -ldflags at build time
var (
	Version   = "dev"
	CommitSHA = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of efctl",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("efctl %s (%s) built %s %s/%s\n", Version, CommitSHA, BuildDate, runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
