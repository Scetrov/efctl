## efctl env extension init

Initialize the builder-scaffold by copying world artifacts

### Synopsis

Runs Step 6 and 7 of the Builder flow. Copies world artifacts from world-contracts/deployments to builder-scaffold/deployments and configures the builder-scaffold .env file.

```
efctl env extension init [flags]
```

### Options

```
  -h, --help               help for init
  -n, --network string     The network to copy artifacts from (localnet or testnet) (default "localnet")
  -w, --workspace string   Path to the workspace directory (default ".")
```

### Options inherited from parent commands

```
      --config-file string   Path to the efctl.yaml or efctl.yml configuration file (default "efctl.yaml")
      --debug                Enable verbose debug logging
      --no-progress          Disable the progress spinner for cleaner CI output
```

### SEE ALSO

* [efctl env extension](efctl_env_extension.md)	 - Manage the builder-scaffold extension flow

