## efctl completion fish

Generate fish completion script

### Synopsis

Generate the autocompletion script for fish.

To load completions in your current shell session:

  $ efctl completion fish | source

To install permanently:

  $ efctl completion fish > ~/.config/fish/completions/efctl.fish


```
efctl completion fish
```

### Options

```
  -h, --help   help for fish
```

### Options inherited from parent commands

```
      --config-file string   Path to the efctl.yaml or efctl.yml configuration file (default "efctl.yaml")
      --debug                Enable verbose debug logging
      --no-progress          Disable the progress spinner for cleaner CI output
```

### SEE ALSO

* [efctl completion](efctl_completion.md)	 - Generate shell completion scripts

