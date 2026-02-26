package cmd

import (
	"fmt"
	"os"
	"strings"

	"efctl/pkg/container"
	"efctl/pkg/ui"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [script-name]",
	Short: "Run a script in the builder-scaffold container",
	Long:  `Runs a predefined script (e.g. from package.json) or a custom arbitrary bash command directly inside the container in the /workspace/builder-scaffold directory.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		scriptName := args[0]
		scriptArgs := args[1:]

		ui.Info.Printf("Running script '%s' inside the container...\n", scriptName)

		c, err := container.NewClient()
		if err != nil {
			ui.Error.Println("Failed to initialize container client: " + err.Error())
			os.Exit(1)
		}

		// Prepare the command
		var bashCmd string
		if len(scriptArgs) > 0 || strings.Contains(scriptName, " ") {
			// If it contains spaces or has extra args, treat as a raw command
			bashCmd = fmt.Sprintf("cd /workspace/builder-scaffold && %s %s", scriptName, strings.Join(scriptArgs, " "))
			bashCmd = strings.TrimSpace(bashCmd)
		} else {
			// Otherwise default to pnpm for convenience
			bashCmd = fmt.Sprintf("cd /workspace/builder-scaffold && pnpm %s", scriptName)
		}

		err = c.Exec(container.ContainerSuiPlayground, []string{"/bin/bash", "-c", bashCmd})
		if err != nil {
			ui.Error.Println("Script execution failed: " + err.Error())
			os.Exit(1)
		}

		ui.Success.Println(fmt.Sprintf("Execution of '%s' completed.", scriptName))
	},
}

func init() {
	// Not adding persistent flags specific to this because `workspacePath` is global, but
	// maybe we need to configure which directory or container to execute against if needed.
	envCmd.AddCommand(runCmd)
}
