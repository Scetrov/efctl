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
		// We use `pnpm run <script>` by default for convenience if passing a script name
		// like "authorise-gate". If the user wants to pass arguments, we append them.

		var bashCmd string
		if len(scriptArgs) > 0 {
			bashCmd = fmt.Sprintf("cd /workspace/builder-scaffold && pnpm %s %s", scriptName, strings.Join(scriptArgs, " "))
		} else {
			bashCmd = fmt.Sprintf("cd /workspace/builder-scaffold && pnpm %s", scriptName)
		}

		err = c.Exec(container.ContainerSuiPlayground, []string{"/bin/bash", "-c", bashCmd})
		if err != nil {
			ui.Error.Println("Script execution failed: " + err.Error())
			os.Exit(1)
		}

		ui.Success.Println(fmt.Sprintf("Script '%s' completed successfully.", scriptName))
	},
}

func init() {
	// Not adding persistent flags specific to this because `workspacePath` is global, but
	// maybe we need to configure which directory or container to execute against if needed.
	envCmd.AddCommand(runCmd)
}
