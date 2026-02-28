package cmd

import (
	"os"

	"efctl/pkg/config"
	"efctl/pkg/ui"
	"efctl/pkg/validate"
	"github.com/spf13/cobra"
)

var configFile string

var rootCmd = &cobra.Command{
	Use:   "efctl",
	Short: "efctl manages the local EVE Frontier Sui development environment",
	Long:  `A fast and flexible CLI to automate the setup, deployment, and teardown of the EVE Frontier local world contracts and smart gates.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load(configFile)
		if err != nil {
			ui.Error.Println("Failed to load config: " + err.Error())
			os.Exit(1)
		}
		config.Loaded = cfg

		// Validate workspace path if a workspace-aware command is being used
		if workspacePath != "" && workspacePath != "." {
			if err := validate.WorkspacePath(workspacePath); err != nil {
				ui.Error.Println("Invalid workspace path: " + err.Error())
				os.Exit(1)
			}
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config-file", config.DefaultConfigFile, "Path to the efctl.yaml configuration file")
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	ui.PrintBanner()
	if err := rootCmd.Execute(); err != nil {
		ui.Error.Println(err.Error())
		os.Exit(1)
	}
}

// GetRootCmd returns the root cobra command
func GetRootCmd() *cobra.Command {
	return rootCmd
}
