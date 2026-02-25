# Usage Guide

`efctl` provides several commands to interact with and manage the EVE Frontier local world contracts and smart gates.

## Global Commands

### `efctl [command] --help`

Use the `--help` flag with any command to see the available options and subcommands.

---

## Environment Management

The `env` command groups operations to bring up and tear down the EVE Frontier local development environment.

### `efctl env up`

Brings up the local environment. It sequentially runs checks, setup, start, and deployment instructions to spin up a fully working EVE Frontier Smart Assembly testing environment.

**Options:**

- `--with-graphql`: Enable the SQL Indexer and GraphQL API.
- `-w, --workspace string`: Path to the workspace directory. (default: `.`)

### `efctl env down`

Tears down the local environment, stopping containers and cleaning up images/volumes.

**Options:**

- `-w, --workspace string`: Path to the workspace directory. (default: `.`)

---

## GraphQL Interaction

The `graphql` command allows you to interact with the Sui GraphQL RPC. By default, it runs against `http://localhost:9125/graphql`.

### `efctl graphql object [address]`

Query a specific object by its ID/address.

**Options:**

- `-e, --endpoint string`: Sui GraphQL RPC endpoint.

### `efctl graphql package [address]`

Query a package and list its associated modules by package ID/address.

**Options:**

- `-e, --endpoint string`: Sui GraphQL RPC endpoint.
