package cmd

import (
	"github.com/spf13/cobra"
)

var workspacePath string

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage the local Sui development environment",
	Long:  `The env command groups operations to bring up and tear down the EVE Frontier local development environment.`,
}

func init() {
	envCmd.PersistentFlags().StringVarP(&workspacePath, "workspace", "w", ".", "Path to the workspace directory")
	rootCmd.AddCommand(envCmd)
}
