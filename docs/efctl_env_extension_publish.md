## efctl env extension publish

Publish a custom extension contract

### Synopsis

Runs Step 8 of the Builder flow. Publishes the single auto-discovered extension contract locally via the container and updates BUILDER_PACKAGE_ID and EXTENSION_CONFIG_ID in .env

```
efctl env extension publish [flags]
```

### Options

```
  -h, --help             help for publish
  -n, --network string   The network to publish to (localnet or testnet) (default "localnet")
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

