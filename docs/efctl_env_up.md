# efctl env up

Bring up the local environment

## Synopsis

Runs check, setup, start, and deploy sequentially to bring up a fully working EVE Frontier Smart Assembly testing environment.

```bash
efctl env up [flags]
```

## Options

```text
  -h, --help           help for up
      --with-frontend   Enable the builder-scaffold web frontend (Vite dev server on port 5173)
      --with-graphql    Enable the SQL Indexer and GraphQL API
```

## Options inherited from parent commands

```text
      --config-file string   Path to the efctl.yaml configuration file (default "efctl.yaml")
  -w, --workspace string     Path to the workspace directory (default ".")
```

## SEE ALSO

- [efctl env](efctl_env.md) - Manage the local Sui development environment
