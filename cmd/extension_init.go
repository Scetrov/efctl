package cmd

import (
	"efctl/pkg/builder"
	"efctl/pkg/ui"
	"github.com/spf13/cobra"
	"os"
)

var extensionInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the builder-scaffold by copying world artifacts",
	Long:  `Runs Step 6 and 7 of the Builder flow. Copies world artifacts from world-contracts/deployments to builder-scaffold/deployments and configures the builder-scaffold .env file.`,
	Run: func(cmd *cobra.Command, args []string) {
		ui.Info.Println("Initializing builder-scaffold extensions environment...")

		if err := builder.InitExtensionEnv(workspacePath, envNetwork); err != nil {
			ui.Error.Println("Initialization failed: " + err.Error())
			os.Exit(1)
		}

		ui.Success.Println("builder-scaffold successfully initialized.")
	},
}

var envNetwork string

func init() {
	extensionInitCmd.Flags().StringVarP(&envNetwork, "network", "n", "localnet", "The network to copy artifacts from (localnet or testnet)")
	// Inherit workspacePath which is set in root.go or typically handled by persistent flags (Wait, is workspacePath global in cmd?)
	extensionInitCmd.Flags().StringVarP(&workspacePath, "workspace", "w", ".", "Path to the workspace directory")
	extensionCmd.AddCommand(extensionInitCmd)
}
