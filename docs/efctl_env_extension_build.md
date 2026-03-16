## efctl env extension build

Compile a Move contract without publishing

### Synopsis

Compiles the specified extension contract (path relative to /workspace) inside the container, catching errors earlier.

```
efctl env extension build [extension-path] [flags]
```

### Options

```
  -h, --help             help for build
  -n, --network string   The network to build for (localnet or testnet) (default "localnet")
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

