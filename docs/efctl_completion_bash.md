## efctl completion bash

Generate bash completion script

### Synopsis

Generate the autocompletion script for bash.

To load completions in your current shell session:

  $ source <(efctl completion bash)

To install permanently (Linux):

  $ efctl completion bash > /etc/bash_completion.d/efctl

To install permanently (macOS with Homebrew):

  $ efctl completion bash > $(brew --prefix)/etc/bash_completion.d/efctl


```
efctl completion bash
```

### Options

```
  -h, --help   help for bash
```

### Options inherited from parent commands

```
      --config-file string   Path to the efctl.yaml or efctl.yml configuration file (default "efctl.yaml")
      --debug                Enable verbose debug logging
      --no-progress          Disable the progress spinner for cleaner CI output
```

### SEE ALSO

* [efctl completion](efctl_completion.md)	 - Generate shell completion scripts

