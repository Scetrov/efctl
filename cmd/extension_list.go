package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"efctl/pkg/builder"
	"efctl/pkg/ui"

	"github.com/jedib0t/go-pretty/v6/table"
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

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Container Path", "Local Path"})
		t.SetStyle(table.StyleRounded)

		for _, candidate := range candidates {
			// Container path: relative to /workspace
			relContainer := strings.TrimPrefix(candidate.ContainerPath, "/workspace/")

			// Local path: relative to workspacePath
			relLocal, err := filepath.Rel(workspacePath, candidate.HostPath)
			if err != nil {
				relLocal = candidate.HostPath
			}

			t.AppendRow(table.Row{relContainer, relLocal})
		}

		t.Render()
	},
}

func init() {
	extensionCmd.AddCommand(extensionListCmd)
}
