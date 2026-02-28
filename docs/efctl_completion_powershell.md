# efctl completion powershell

Generate powershell completion script

## Synopsis

Generate the autocompletion script for PowerShell.

To load completions in your current shell session:

  PS> efctl completion powershell | Out-String | Invoke-Expression

To install permanently, add the output to your PowerShell profile:

  PS> efctl completion powershell >> $PROFILE

```text
efctl completion powershell
```

## Options

```text
  -h, --help   help for powershell
```

## SEE ALSO

- [efctl completion](efctl_completion.md) - Generate shell completion scripts
