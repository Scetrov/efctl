# Usage Guide

`efctl` provides several commands to interact with and manage the EVE Frontier local world contracts and smart gates.

## Global Options

- `--config-file string`: Path to the `efctl.yaml` configuration file. (default: `efctl.yaml`)
- `--help`: Use the `--help` flag with any command to see the available options and subcommands.

---

## Configuration File

`efctl` supports an optional `efctl.yaml` configuration file. By default it looks for `efctl.yaml` in the current directory. Use `--config-file` to specify a different path.

All properties are optional. CLI flags override values from the config file.

```yaml
# Enable the builder-scaffold web frontend (Vite dev server on port 5173)
with-frontend: false

# Enable the SQL Indexer and GraphQL API
with-graphql: false

# Git clone URL for the world-contracts repository
world-contracts-url: "https://github.com/evefrontier/world-contracts.git"

# Branch to checkout for world-contracts (default: main)
world-contracts-branch: "main"

# Git clone URL for the builder-scaffold repository
builder-scaffold-url: "https://github.com/evefrontier/builder-scaffold.git"

# Branch to checkout for builder-scaffold (default: main)
builder-scaffold-branch: "main"
```

---

## Environment Management

The `env` command groups operations to bring up and tear down the EVE Frontier local development environment.

### `efctl env up`

Brings up the local environment. It sequentially runs checks, setup, start, and deployment instructions to spin up a fully working EVE Frontier Smart Assembly testing environment.

**Options:**

- `--with-frontend`: Enable the builder-scaffold web frontend (Vite dev server on port 5173).
- `--with-graphql`: Enable the SQL Indexer and GraphQL API.
- `-w, --workspace string`: Path to the workspace directory. (default: `.`)

These flags can also be set via `efctl.yaml` (`with-frontend`, `with-graphql`). CLI flags take precedence over config file values.

### `efctl env down`

Tears down the local environment, stopping containers and cleaning up images/volumes.

**Options:**

- `-w, --workspace string`: Path to the workspace directory. (default: `.`)

- `-w, --workspace string`: Path to the workspace directory. (default: `.`)

---

## Extension Flow

The `extension` command groups operations defined in the EVE Frontier Builder Flow for Docker, automating the setup and publishing of custom extensions.

### `efctl env extension init`

Initializes the builder-scaffold by copying world artifacts `deployments/localnet` into the `builder-scaffold` directory and configuring its `.env` file inline.

**Options:**

- `-n, --network string`: The network to copy artifacts from (default: `localnet`)
- `-w, --workspace string`: Path to the workspace directory. (default: `.`)

### `efctl env extension publish [contract-path]`

Publishes the custom extension to the smart assembly testnet and updates the builder.env. `[contract-path]` must be relative to `builder-scaffold/move-contracts`.

**Options:**

- `-n, --network string`: The network to publish to (default: `localnet`)

---

## Script Execution

### `efctl env run [script-name] [args...]`

Runs a predefined script (e.g. from `package.json`) or a custom command directly inside the container in the `/workspace/builder-scaffold` directory.

Example: `efctl env run authorise-gate`

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
