# efctl completion zsh

Generate zsh completion script

## Synopsis

Generate the autocompletion script for zsh.

If shell completion is not already enabled in your environment,
you will need to enable it by adding the following to your ~/.zshrc:

  autoload -Uz compinit && compinit

To load completions for every new session, run once:

  $ efctl completion zsh > "${fpath[1]}/_efctl"

You will need to start a new shell for this setup to take effect.

```text
efctl completion zsh
```

## Options

```text
  -h, --help   help for zsh
```

## SEE ALSO

- [efctl completion](efctl_completion.md) - Generate shell completion scripts
