package cmd

import (
	"os"

	"efctl/pkg/builder"
	"efctl/pkg/container"
	"efctl/pkg/ui"
	"efctl/pkg/validate"
	"github.com/spf13/cobra"
)

var extensionTestCmd = &cobra.Command{
	Use:   "test [extension-path]",
	Short: "Run sui move test for a Move contract",
	Long:  `Runs 'sui move test' for the specified extension contract (path relative to /workspace) inside the container.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		extensionPath := args[0]
		if err := validate.Network(envNetwork); err != nil {
			ui.Error.Println(err.Error())
			os.Exit(1)
		}

		c, err := container.NewClient()
		if err != nil {
			ui.Error.Println("Failed to create container client: " + err.Error())
			os.Exit(1)
		}

		candidate, err := builder.GetCandidate(workspacePath, extensionPath)
		if err != nil {
			ui.Error.Printf("Error: extension %q not found.\n\n", extensionPath)
			closest := builder.FindClosestMatch(workspacePath, extensionPath)
			if len(closest) > 0 {
				ui.Info.Println("Did you mean:")
				for _, match := range closest {
					ui.Info.Printf("  - %s\n", match)
				}
			}
			os.Exit(1)
		}

		if err := builder.TestExtension(c, workspacePath, envNetwork, candidate); err != nil {
			ui.Error.Println("Tests failed: " + err.Error())
			os.Exit(1)
		}
	},
}

func init() {
	extensionTestCmd.Flags().StringVarP(&envNetwork, "network", "n", "localnet", "The network to test for (localnet or testnet)")
	extensionCmd.AddCommand(extensionTestCmd)
}
