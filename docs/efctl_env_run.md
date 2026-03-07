## efctl env run

Run a script in the builder-scaffold container

### Synopsis

Runs a predefined script (e.g. from package.json) or a custom arbitrary bash command directly inside the container in the /workspace/builder-scaffold directory.

```
efctl env run [script-name] [flags]
```

### Options

```
  -h, --help   help for run
```

### Options inherited from parent commands

```
      --config-file string   Path to the efctl.yaml or efctl.yml configuration file (default "efctl.yaml")
      --debug                Enable verbose debug logging
      --no-progress          Disable the progress spinner for cleaner CI output
  -w, --workspace string     Path to the workspace directory (default ".")
```

### SEE ALSO

* [efctl env](efctl_env.md)	 - Manage the local Sui development environment

