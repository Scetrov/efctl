package cmd

import (
	"efctl/pkg/builder"
	"efctl/pkg/ui"
	"github.com/spf13/cobra"
	"os"
)

var extensionPublishCmd = &cobra.Command{
	Use:   "publish [contract-path]",
	Short: "Publish a custom extension contract",
	Long:  `Runs Step 8 of the Builder flow. Publishes the custom contract locally via the container and updates BUILDER_PACKAGE_ID and EXTENSION_CONFIG_ID in .env`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contractPath := args[0]
		ui.Info.Printf("Publishing extension contract from %s...\n", contractPath)

		if err := builder.PublishExtension(workspacePath, envNetwork, contractPath); err != nil {
			ui.Error.Println("Publish failed: " + err.Error())
			os.Exit(1)
		}

		ui.Success.Println("Extension contract published successfully.")
	},
}

func init() {
	extensionPublishCmd.Flags().StringVarP(&envNetwork, "network", "n", "localnet", "The network to publish to (localnet or testnet)")
	extensionCmd.AddCommand(extensionPublishCmd)
}
