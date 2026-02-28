# efctl completion bash

Generate bash completion script

## Synopsis

Generate the autocompletion script for bash.

To load completions in your current shell session:

  $ source <(efctl completion bash)

To install permanently (Linux):

  $ efctl completion bash > /etc/bash_completion.d/efctl

To install permanently (macOS with Homebrew):

  $ efctl completion bash > $(brew --prefix)/etc/bash_completion.d/efctl

```text
efctl completion bash
```

## Options

```text
  -h, --help   help for bash
```

## SEE ALSO

- [efctl completion](efctl_completion.md) - Generate shell completion scripts
