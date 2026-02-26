package cmd

import (
	"github.com/spf13/cobra"
)

var extensionCmd = &cobra.Command{
	Use:   "extension",
	Short: "Manage the builder-scaffold extension flow",
	Long:  `The extension command groups operations defined in the EVE Frontier Builder Flow for Docker, such as init and publish.`,
}

func init() {
	// We will attach extension subcommands here
	envCmd.AddCommand(extensionCmd)
}
