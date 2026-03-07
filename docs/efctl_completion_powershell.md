## efctl completion powershell

Generate powershell completion script

### Synopsis

Generate the autocompletion script for PowerShell.

To load completions in your current shell session:

  PS> efctl completion powershell | Out-String | Invoke-Expression

To install permanently, add the output to your PowerShell profile:

  PS> efctl completion powershell >> $PROFILE


```
efctl completion powershell
```

### Options

```
  -h, --help   help for powershell
```

### Options inherited from parent commands

```
      --config-file string   Path to the efctl.yaml or efctl.yml configuration file (default "efctl.yaml")
      --debug                Enable verbose debug logging
      --no-progress          Disable the progress spinner for cleaner CI output
```

### SEE ALSO

* [efctl completion](efctl_completion.md)	 - Generate shell completion scripts

