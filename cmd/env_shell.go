package cmd

import (
	"fmt"
	"os"

	"efctl/pkg/container"
	"efctl/pkg/ui"
	"github.com/spf13/cobra"
)

var envShellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Open a shell inside the running container",
	Long:  `Executes into the running sui-playground container with an interactive bash shell.`,
	Run: func(cmd *cobra.Command, args []string) {
		ui.Info.Println("Opening shell in container...")

		c, err := container.NewClient()
		if err != nil {
			ui.Error.Println("Failed to initialize container client: " + err.Error())
			os.Exit(1)
		}

		if err := c.InteractiveShell(container.ContainerSuiPlayground); err != nil {
			ui.Error.Println(fmt.Sprintf("Failed to open shell: %v", err))
			os.Exit(1)
		}
	},
}

func init() {
	envCmd.AddCommand(envShellCmd)
}
