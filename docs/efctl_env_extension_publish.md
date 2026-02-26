# efctl env extension publish

Publish a custom extension contract

## Synopsis

Runs Step 8 of the Builder flow. Publishes the custom contract locally via the container and outputs BUILDER_PACKAGE_ID and EXTENSION_CONFIG_ID.

```bash
efctl env extension publish [contract-path] [flags]
```

## Options

```text
  -h, --help               help for publish
  -n, --network string     The network to publish to (localnet or testnet) (default "localnet")
```

## SEE ALSO

- [efctl env extension](efctl_env_extension.md) - Manage the builder-scaffold extension flow
