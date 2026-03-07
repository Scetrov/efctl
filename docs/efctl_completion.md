## efctl completion

Generate shell completion scripts

### Synopsis

Generate shell completion scripts for efctl.

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


### Options

```
  -h, --help   help for completion
```

### Options inherited from parent commands

```
      --config-file string   Path to the efctl.yaml or efctl.yml configuration file (default "efctl.yaml")
      --debug                Enable verbose debug logging
      --no-progress          Disable the progress spinner for cleaner CI output
```

### SEE ALSO

* [efctl](efctl.md)	 - efctl manages the local EVE Frontier Sui development environment
* [efctl completion bash](efctl_completion_bash.md)	 - Generate bash completion script
* [efctl completion fish](efctl_completion_fish.md)	 - Generate fish completion script
* [efctl completion powershell](efctl_completion_powershell.md)	 - Generate powershell completion script
* [efctl completion zsh](efctl_completion_zsh.md)	 - Generate zsh completion script

