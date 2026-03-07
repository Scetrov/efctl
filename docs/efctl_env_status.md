## efctl env status

Show environment status without launching the dashboard

### Synopsis

Shows container status, port usage, chain health, and deployed world metadata in a lightweight non-interactive output.

```
efctl env status [flags]
```

### Options

```
  -h, --help             help for status
      --rpc-url string   Sui JSON-RPC endpoint URL (default "http://localhost:9000")
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

