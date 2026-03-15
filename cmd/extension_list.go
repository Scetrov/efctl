package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"efctl/pkg/builder"
	"efctl/pkg/ui"
	"github.com/spf13/cobra"
)

var extensionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available extensions in the workspace",
	Long:  `Scans the current configured workspace for extensions and displays them in a table format showing their container and local paths.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		searchRoots, err := builder.GetPublishSearchRoots(workspacePath)
		if err != nil {
			ui.Error.Println("Failed to get search roots: " + err.Error())
			os.Exit(1)
		}

		candidates, err := builder.DiscoverPublishCandidates(searchRoots)
		if err != nil {
			ui.Error.Println("Failed to discover extensions: " + err.Error())
			os.Exit(1)
		}

		if len(candidates) == 0 {
			ui.Info.Println("No extensions found in the current workspace.")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "CONTAINER PATH\tLOCAL PATH")

		for _, candidate := range candidates {
			fmt.Fprintf(w, "%s\t%s\n", candidate.ContainerPath, candidate.HostPath)
		}
		w.Flush()
	},
}

func init() {
	extensionCmd.AddCommand(extensionListCmd)
}
