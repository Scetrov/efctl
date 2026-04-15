# Usage Guide

`efctl` provides several commands to interact with and manage the EVE Frontier local world contracts and smart gates.

## Global Options

- `--config-file string`: Path to the `efctl.yaml` or `efctl.yml` configuration file. (default: `efctl.yaml`)
- `--debug`: Enable verbose debug logging.
- `--no-progress`: Disable the progress spinner for cleaner CI output.
- `--help`: Use the `--help` flag with any command to see the available options and subcommands.

---

## ⚙️ Configuration File

`efctl` looks for `efctl.yaml` or `efctl.yml` in the current directory and its parents.

> [!TIP]
> Run `efctl init` to scaffold a config file with the current recommended defaults.

### Configuration Fields

| Field | Description | Default |
| :--- | :--- | :--- |
| `with-frontend` | Enable builder-scaffold web frontend (Vite) | `false` |
| `with-graphql` | Enable SQL Indexer and GraphQL API | `false` |
| `world-contracts-url` | Git URL for world contracts | `https://github.com/evefrontier/world-contracts.git` |
| `world-contracts-ref` | Branch, tag, or commit for world contracts | `v0.0.23` |
| `builder-scaffold-url` | Git URL for builder-scaffold | `https://github.com/evefrontier/builder-scaffold.git` |
| `builder-scaffold-ref` | Branch, tag, or commit for builder-scaffold | `v0.0.2` |
| `git-autocrlf` | Enable Git `core.autocrlf` for clones | `false` |
| `container-engine` | Container engine to use (`docker`, `podman`) | `auto-detect` |
| `additional-bind-mounts` | List of custom host paths to mount | `[]` |

#### Example `efctl.yaml`

```yaml
# Enable components
with-frontend: true
with-graphql: true

# Control versions
world-contracts-ref: "v0.0.23"
builder-scaffold-ref: "v0.0.2"

# Custom mounts
additional-bind-mounts:
  - hostPath: ./my-assets
    identifier: assets
```

> [!IMPORTANT]
> Config discovery is independent of the `--workspace` flag. CLI flags always override values from the config file.

---

## 🏗️ Environment Management

The `env` command groups operations to manage the EVE Frontier local development environment.

### `efctl env up`

Brings up the local environment. It sequentially runs checks, setup, start, and deployment instructions.

**Common Options:**

- `--with-frontend`: Enable the web frontend.
- `--with-graphql`: Enable the GraphQL API.
- `-w, --workspace PATH`: Path to the workspace directory (default: `.`).

### `efctl env down`

Tears down the local environment, stopping containers and cleaning up images/volumes.

### `efctl env status`

Displays the current status of the local environment containers. Perfect for verifying if services are running.

### `efctl env dash`

Opens a high-performance interactive terminal dashboard.

---

## 🚀 Extension Flow

Automate the setup and publishing of custom extensions within the EVE Frontier ecosystem.

### `efctl env extension init`

Initializes the builder-scaffold by synchronizing world artifacts and configuring the environment.

### `efctl env extension publish`

Publishes your extension to the localnet/testnet.

> [!NOTE]
> `efctl` auto-discovers extensions by looking for `Move.toml` files that declare a `world` dependency.

**Publish Workflow:**
1. Scan `builder-scaffold/move-contracts` and `world-contracts/contracts`.
2. Verify exactly one publish candidate is found.
3. Build and publish the package to the target network.

---

## 📜 Script & Tool Interactions

### `efctl env run [command]`

Execute commands directly inside the `builder-scaffold` container.

**Examples:**

```bash
# Run a package.json script
efctl env run npm run dev

# Run a custom move command
efctl env run sui client objects
```

### `efctl env faucet`

Request gas coins from the local Sui faucet.

```bash
# Fund a specific address
efctl env faucet --address 0x...
```

---

## Extension Flow

The `extension` command groups operations defined in the EVE Frontier Builder Flow for Docker, automating the setup and publishing of custom extensions.

### `efctl env extension init`

Initializes the builder-scaffold by copying world artifacts `deployments/localnet` into the `builder-scaffold` directory and configuring its `.env` file inline.

**Options:**

- `-n, --network string`: The network to copy artifacts from (default: `localnet`)
- `-w, --workspace string`: Path to the workspace directory. (default: `.`)

### `efctl env extension publish`

Publishes the single auto-discovered extension to the smart assembly testnet and updates `builder-scaffold/.env`.

The command scans immediate child directories under `builder-scaffold/move-contracts`, `world-contracts/contracts`, and any configured additional bind mounts. A directory is only considered a publish candidate if it contains a `Move.toml` file and declares a `world` dependency, which filters out shared dependency packages such as `world` itself.

If zero candidates are found, or if more than one candidate is found, the command aborts with an error.

Optional custom bind mounts can be configured in `efctl.yaml`:

```yaml
additional-bind-mounts:
	- hostPath: ./some/path
		identifier: some_path
```

Each configured mount is exposed to the Sui container at `/workspace/{identifier}`.

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
