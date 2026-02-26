# efctl env extension init

Initialize the builder-scaffold by copying world artifacts

## Synopsis

Runs Step 6 and 7 of the Builder flow. Copies world artifacts from world-contracts/deployments to builder-scaffold/deployments and configures the builder-scaffold .env file.

```bash
efctl env extension init [flags]
```

## Options

```text
  -h, --help               help for init
  -n, --network string     The network to copy artifacts from (localnet or testnet) (default "localnet")
  -w, --workspace string   Path to the workspace directory (default ".")
```

## SEE ALSO

- [efctl env extension](efctl_env_extension.md) - Manage the builder-scaffold extension flow
