## efctl env faucet

Request gas from the local faucet

### Synopsis

Request gas coins from the local Sui faucet for a specific address.

```
efctl env faucet [flags]
```

### Options

```
  -a, --address string      The address to receive gas (required)
      --faucet-url string   The URL of the faucet (default "http://localhost:9123")
  -h, --help                help for faucet
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

