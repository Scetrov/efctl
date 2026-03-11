# Usage Guide

`efctl` provides several commands to interact with and manage the EVE Frontier local world contracts and smart gates.

## Global Options

- `--config-file string`: Path to the `efctl.yaml` or `efctl.yml` configuration file. (default: `efctl.yaml`)
- `--debug`: Enable verbose debug logging.
- `--no-progress`: Disable the progress spinner for cleaner CI output.
- `--help`: Use the `--help` flag with any command to see the available options and subcommands.

---

## Configuration File

`efctl` supports optional `efctl.yaml` and `efctl.yml` configuration files. By default it starts in the **current working directory** (the directory from which you run the `efctl` command) and recursively searches parent directories until it finds one.

If both `efctl.yaml` and `efctl.yml` exist in the same directory, `efctl.yaml` is preferred.

**Important:** Config discovery is independent of the `--workspace` flag. Use `--config-file` to bypass discovery and load an explicit config path.

To see which config file is loaded (or if none was found), use the `--debug` flag:
```bash
efctl --debug env up
```

All properties are optional. CLI flags override values from the config file.

Run `efctl init` to scaffold a config file with the current recommended defaults.

```yaml
# Enable the builder-scaffold web frontend (Vite dev server on port 5173)
with-frontend: true

# Enable the SQL Indexer and GraphQL API
with-graphql: true

# Git clone URL for the world-contracts repository
world-contracts-url: "https://github.com/evefrontier/world-contracts.git"

# Ref (branch, tag, or commit) to checkout for world-contracts (default: main)
world-contracts-ref: "v0.0.18"

# Git clone URL for the builder-scaffold repository
builder-scaffold-url: "https://github.com/evefrontier/builder-scaffold.git"

# Ref (branch, tag, or commit) to checkout for builder-scaffold (default: main)
builder-scaffold-ref: "v0.0.2"
```

### `efctl init`

Creates an `efctl.yaml` file in the current directory. Use `--config-file` to target a different path and `--force` to overwrite an existing file.

---

## Environment Management

The `env` command groups operations to bring up, manage, and tear down the EVE Frontier local development environment.

### `efctl env up`

Brings up the local environment. It sequentially runs checks, setup, start, and deployment instructions to spin up a fully working EVE Frontier Smart Assembly testing environment.

**Options:**

- `--with-frontend`: Enable the builder-scaffold web frontend (Vite dev server on port 5173).
- `--with-graphql`: Enable the SQL Indexer and GraphQL API.
- `-w, --workspace string`: Path to the workspace directory. (default: `.`)

### `efctl env down`

Tears down the local environment, stopping containers and cleaning up images/volumes.

**Options:**

- `-w, --workspace string`: Path to the workspace directory. (default: `.`)

### `efctl env status`

Displays the current status of the local environment containers.

**Options:**

- `-w, --workspace string`: Path to the workspace directory. (default: `.`)

### `efctl env shell`

Drops you into an interactive `/bin/bash` shell inside the `builder-scaffold` container.

**Options:**

- `-w, --workspace string`: Path to the workspace directory. (default: `.`)

### `efctl env dash`

Opens an interactive terminal dashboard for inspecting and managing the local environment.

**Options:**

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

Publishes the custom extension to the smart assembly testnet and updates the builder.env. `[contract-path]` must be relative to `builder-scaffold/move-contracts` or `world-contracts/contracts`.

**Options:**

- `-n, --network string`: The network to publish to (default: `localnet`)
- `-w, --workspace string`: Path to the workspace directory. (default: `.`)

---

## Script Execution

### `efctl env run [script-name] [args...]`

Runs a predefined script (e.g. from `package.json`) or a custom command directly inside the container in the `/workspace/builder-scaffold` directory.

---

## GraphQL Interaction

The `graphql` command allows you to interact with the Sui GraphQL RPC. By default, it runs against `http://localhost:9125/graphql`.

### `efctl graphql object [address]`

Query a specific object by its ID/address.

**Options:**

- `-e, --endpoint string`: Sui GraphQL RPC endpoint. (default: `http://localhost:9125/graphql`)

### `efctl graphql package [address]`

Query a package and list its associated modules by package ID/address.

**Options:**

- `-e, --endpoint string`: Sui GraphQL RPC endpoint. (default: `http://localhost:9125/graphql`)

---

## World Interaction

The `world` command groups operations used to query and interact with the deployed EVE Frontier local world contracts.

### `efctl world query [object_id]`

A utility that queries your smart assemblies (gates, turrets) or tokens on-chain to provide a human-readable list of information using GraphQL.

**Options:**

- `-e, --endpoint string`: Sui GraphQL endpoint. (default: `http://localhost:9125/graphql`)

---

## Sui Dependency Management

The `sui` command helps you manage the host-level Sui dependencies, primarily setting up the local CLI client that efctl needs.

### `efctl sui install`

Installs the Sui CLI binary directly depending on your OS and architecture. Uses the pre-built binaries from GitHub Releases to save compilation time.

**Options:**

- `-v, --version string`: Set the required sui-cli version. (default: `mainnet-v1.41.0`)

---

## Other Commands

### `efctl update`

Downloads and installs the latest version of `efctl` directly from GitHub releases. Updates the binary in-place or helps you download it if you don't have write access to its current location.

### `efctl completion`

Generate shell autocomplete features for `bash`, `fish`, `powershell`, or `zsh`.

### `efctl version`

Prints the current version of the `efctl` binary.
