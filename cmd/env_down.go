package cmd

import (
	"fmt"
	"os"

	"efctl/pkg/setup"
	"efctl/pkg/sui"
	"efctl/pkg/ui"
	"github.com/spf13/cobra"
)

var envDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Tear down the local environment",
	Long:  `Cleans up the local Sui development environment by stopping and removing all related containers.`,
	Run: func(cmd *cobra.Command, args []string) {
		ui.Info.Println("Starting cleanup...")
		// Assuming setup.CleanEnvironment doesn't need workspacePath currently,
		// but if it ever does, workspacePath is accessible from env.go
		err := setup.CleanEnvironment()
		if err != nil {
			ui.Error.Println("Cleanup failed: " + err.Error())
			os.Exit(1)
		}

		// Also teardown Sui client configuration
		if err := sui.TeardownSui(); err != nil {
			ui.Warn.Println("Sui client teardown failed: " + err.Error())
		}

		ui.Success.Println(fmt.Sprintf("%s Environment cleaned up successfully.", ui.CleanEmoji))
	},
}

func init() {
	envCmd.AddCommand(envDownCmd)
}
