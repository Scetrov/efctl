package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [shell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for efctl.

Supported shells: bash, zsh, fish, powershell.

To load completions:

Bash:
  $ source <(efctl completion bash)

  # To install permanently (Linux):
  $ efctl completion bash > /etc/bash_completion.d/efctl

  # To install permanently (macOS with Homebrew):
  $ efctl completion bash > $(brew --prefix)/etc/bash_completion.d/efctl

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. Add the following to your ~/.zshrc:
  $ autoload -Uz compinit && compinit

  $ efctl completion zsh > "${fpath[1]}/_efctl"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ efctl completion fish | source

  # To install permanently:
  $ efctl completion fish > ~/.config/fish/completions/efctl.fish

PowerShell:
  PS> efctl completion powershell | Out-String | Invoke-Expression

  # To install permanently, add the output to your PowerShell profile:
  PS> efctl completion powershell >> $PROFILE
`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// No-op: completion generation does not require config loading
	},
}

var completionBashCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generate bash completion script",
	Long: `Generate the autocompletion script for bash.

To load completions in your current shell session:

  $ source <(efctl completion bash)

To install permanently (Linux):

  $ efctl completion bash > /etc/bash_completion.d/efctl

To install permanently (macOS with Homebrew):

  $ efctl completion bash > $(brew --prefix)/etc/bash_completion.d/efctl
`,
	Args:                  cobra.NoArgs,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenBashCompletionV2(os.Stdout, true)
	},
}

var completionZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generate zsh completion script",
	Long: `Generate the autocompletion script for zsh.

If shell completion is not already enabled in your environment,
you will need to enable it by adding the following to your ~/.zshrc:

  autoload -Uz compinit && compinit

To load completions for every new session, run once:

  $ efctl completion zsh > "${fpath[1]}/_efctl"

You will need to start a new shell for this setup to take effect.
`,
	Args:                  cobra.NoArgs,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenZshCompletion(os.Stdout)
	},
}

var completionFishCmd = &cobra.Command{
	Use:   "fish",
	Short: "Generate fish completion script",
	Long: `Generate the autocompletion script for fish.

To load completions in your current shell session:

  $ efctl completion fish | source

To install permanently:

  $ efctl completion fish > ~/.config/fish/completions/efctl.fish
`,
	Args:                  cobra.NoArgs,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenFishCompletion(os.Stdout, true)
	},
}

var completionPowershellCmd = &cobra.Command{
	Use:   "powershell",
	Short: "Generate powershell completion script",
	Long: `Generate the autocompletion script for PowerShell.

To load completions in your current shell session:

  PS> efctl completion powershell | Out-String | Invoke-Expression

To install permanently, add the output to your PowerShell profile:

  PS> efctl completion powershell >> $PROFILE
`,
	Args:                  cobra.NoArgs,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenPowerShellCompletion(os.Stdout)
	},
}

func init() {
	completionCmd.AddCommand(completionBashCmd)
	completionCmd.AddCommand(completionZshCmd)
	completionCmd.AddCommand(completionFishCmd)
	completionCmd.AddCommand(completionPowershellCmd)
	rootCmd.AddCommand(completionCmd)
}
