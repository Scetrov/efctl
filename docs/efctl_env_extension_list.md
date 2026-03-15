## efctl env extension list

List all available extensions in the workspace

### Synopsis

Scans the current configured workspace for extensions and displays them in a table format showing their container and local paths.

```
efctl env extension list [flags]
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
      --config-file string   Path to the efctl.yaml or efctl.yml configuration file (default "efctl.yaml")
      --debug                Enable verbose debug logging
      --no-progress          Disable the progress spinner for cleaner CI output
  -w, --workspace string     Path to the workspace directory (default ".")
```

### SEE ALSO

* [efctl env extension](efctl_env_extension.md)	 - Manage the builder-scaffold extension flow

