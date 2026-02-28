package cmd

import (
	"fmt"
	"os"
	"regexp"

	"efctl/pkg/container"
	"efctl/pkg/ui"

	"github.com/spf13/cobra"
)

// safeScriptNameRe matches only safe script/command names (alphanumeric, hyphens, underscores, dots, slashes).
var safeScriptNameRe = regexp.MustCompile(`^[a-zA-Z0-9_./-]+$`)

var runCmd = &cobra.Command{
	Use:   "run [script-name]",
	Short: "Run a script in the builder-scaffold container",
	Long:  `Runs a predefined script (e.g. from package.json) or a custom arbitrary bash command directly inside the container in the /workspace/builder-scaffold directory.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		scriptName := args[0]
		scriptArgs := args[1:]

		// Validate script name to prevent shell metacharacter injection
		if !safeScriptNameRe.MatchString(scriptName) {
			ui.Error.Println("Invalid script name: only alphanumeric characters, hyphens, underscores, dots, and slashes are allowed")
			os.Exit(1)
		}
		for _, arg := range scriptArgs {
			if !safeScriptNameRe.MatchString(arg) {
				ui.Error.Println(fmt.Sprintf("Invalid argument %q: only alphanumeric characters, hyphens, underscores, dots, and slashes are allowed", arg))
				os.Exit(1)
			}
		}

		ui.Info.Printf("Running script '%s' inside the container...\n", scriptName)

		c, err := container.NewClient()
		if err != nil {
			ui.Error.Println("Failed to initialize container client: " + err.Error())
			os.Exit(1)
		}

		// Build the command using exec "$@" pattern to avoid shell metacharacter interpretation.
		// Arguments are passed as separate exec.Command args, not interpolated into a shell string.
		execArgs := []string{
			"/bin/bash", "-c",
			`cd /workspace/builder-scaffold && exec "$@"`,
			"--", // $0 placeholder for bash -c
		}

		// If no extra args and no spaces, default to pnpm wrapper
		if len(scriptArgs) == 0 {
			execArgs = append(execArgs, "pnpm", scriptName)
		} else {
			execArgs = append(execArgs, scriptName)
			execArgs = append(execArgs, scriptArgs...)
		}

		err = c.Exec(container.ContainerSuiPlayground, execArgs)
		if err != nil {
			ui.Error.Println("Script execution failed: " + err.Error())
			os.Exit(1)
		}

		ui.Success.Println(fmt.Sprintf("Execution of '%s' completed.", scriptName))
	},
}

func init() {
	envCmd.AddCommand(runCmd)
}
