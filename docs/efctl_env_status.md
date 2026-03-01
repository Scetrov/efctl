# efctl env status

Show environment status without launching the dashboard

## Synopsis

Shows container status, port usage, chain health, and deployed world metadata in a lightweight non-interactive output.

```text
efctl env status [flags]
```

## Options

```text
  -h, --help             help for status
      --rpc-url string   Sui JSON-RPC endpoint URL (default "http://localhost:9000")
```

## Options inherited from parent commands

```text
      --config-file string   Path to the efctl.yaml configuration file (default "efctl.yaml")
  -w, --workspace string     Path to the workspace directory (default ".")
```

## SEE ALSO

- [efctl env](efctl_env.md) - Manage the local Sui development environment
