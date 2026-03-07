## efctl env up

Bring up the local environment

### Synopsis

Runs check, setup, start, and deploy sequentially to bring up a fully working EVE Frontier Smart Assembly testing environment.

```
efctl env up [flags]
```

### Options

```
  -h, --help            help for up
      --with-frontend   Enable the builder-scaffold web frontend (Vite dev server on port 5173) (default true)
      --with-graphql    Enable the SQL Indexer and GraphQL API (default true)
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

