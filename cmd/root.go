package cmd

import (
	"os"

	"efctl/pkg/ui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "efctl",
	Short: "efctl manages the local EVE Frontier Sui development environment",
	Long:  `A fast and flexible CLI to automate the setup, deployment, and teardown of the EVE Frontier local world contracts and smart gates.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	ui.PrintBanner()
	if err := rootCmd.Execute(); err != nil {
		ui.Error.Println(err.Error())
		os.Exit(1)
	}
}
