package cmd

import (
	"os"

	"efctl/pkg/builder"
	"efctl/pkg/container"
	"efctl/pkg/ui"
	"efctl/pkg/validate"
	"github.com/spf13/cobra"
)

var extensionPublishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish a custom extension contract",
	Long:  `Runs Step 8 of the Builder flow. Publishes the single auto-discovered extension contract locally via the container and updates BUILDER_PACKAGE_ID and EXTENSION_CONFIG_ID in .env`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := validate.Network(envNetwork); err != nil {
			ui.Error.Println(err.Error())
			os.Exit(1)
		}

		c, err := container.NewClient()
		if err != nil {
			ui.Error.Println("Failed to create container client: " + err.Error())
			os.Exit(1)
		}

		if err := builder.PublishExtension(c, workspacePath, envNetwork); err != nil {
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
