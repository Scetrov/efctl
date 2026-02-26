package cmd

import (
	"github.com/spf13/cobra"
)

var suiCmd = &cobra.Command{
	Use:   "sui",
	Short: "Manage the local Sui client and environment",
	Long:  `The sui command group provides utilities for installing and configuring the Sui client for use with EVE Frontier.`,
}

func init() {
	rootCmd.AddCommand(suiCmd)
}
