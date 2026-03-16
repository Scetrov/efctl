## efctl env extension test

Run sui move test for a Move contract

### Synopsis

Runs 'sui move test' for the specified extension contract (path relative to /workspace) inside the container.

```
efctl env extension test [extension-path] [flags]
```

### Options

```
  -h, --help             help for test
  -n, --network string   The network to test for (localnet or testnet) (default "localnet")
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

