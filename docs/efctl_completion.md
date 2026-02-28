# efctl completion

Generate shell completion scripts

## Synopsis

Generate shell completion scripts for efctl.

Supported shells: bash, zsh, fish, powershell.

To load completions:

Bash:
  $ source <(efctl completion bash)

Zsh:
  $ efctl completion zsh > "${fpath[1]}/_efctl"

Fish:
  $ efctl completion fish | source

PowerShell:
  PS> efctl completion powershell | Out-String | Invoke-Expression

## Options

```text
  -h, --help   help for completion
```

## SEE ALSO

- [efctl](efctl.md) - efctl manages the local EVE Frontier Sui development environment
- [efctl completion bash](efctl_completion_bash.md) - Generate bash completion script
- [efctl completion zsh](efctl_completion_zsh.md) - Generate zsh completion script
- [efctl completion fish](efctl_completion_fish.md) - Generate fish completion script
- [efctl completion powershell](efctl_completion_powershell.md) - Generate powershell completion script
