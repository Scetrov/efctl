# TODO

Feature ideas and improvements for `efctl`.

---

## Environment Lifecycle

- [x] **`efctl env status`** — Lightweight check showing which containers are running, port availability, and chain health without the full TUI dashboard overhead.
- [ ] **`efctl env restart`** — Currently only available via the dashboard's `r` hotkey. A standalone command would be useful for scripting and CI.
- [ ] **`efctl env logs [container]`** — Stream or tail logs from a specific container (sui-playground, postgres, frontend) without entering the TUI.

## Builder Flow

- [x] **`efctl env extension test [contract-path]`** — Run `sui move test` for a Move contract inside the container, so developers don't need to `shell` in or use `env run`.
- [x] **`efctl env extension build [contract-path]`** — Compile a Move contract without publishing, catching errors earlier.
- [x] **`efctl env extension list`** — List published extensions with their package IDs, config IDs, and status from the `.env`.

## GraphQL & Chain Interaction

- [ ] **`efctl graphql query [raw-query]`** — Run an arbitrary GraphQL query (from a string or `.graphql` file) against the RPC endpoint, enabling exploration beyond objects/packages.
- [ ] **`efctl graphql transaction [digest]`** — Inspect a specific transaction by digest (sender, gas, effects, events).
- [ ] **`efctl graphql events [filter]`** — Subscribe to or tail recent chain events, useful for debugging Smart Gate interactions.

## Developer Experience

- [x] **`efctl env init`** — Scaffold a new project directory with a starter `efctl.yaml`, Move contract template, and directory structure, reducing manual boilerplate for new builders. (Implemented as `efctl init`).
- [x] **`efctl doctor`** — Comprehensive diagnostic that checks prerequisites, port conflicts, Docker daemon health, Sui client config, disk space, and version compatibility — then outputs a shareable report.
- [x] **`efctl completion`** — Shell completion generation (bash/zsh/fish/powershell) via Cobra's built-in `GenBashCompletion` etc.

## Deployment & Networking

- [ ] **Testnet/Devnet deployment flow** — Currently `env up` is localnet-only. A `--network testnet` flag (or `efctl env deploy --network testnet`) would bridge the gap for staging.
- [x] **`efctl sui faucet [address]`** — Request test tokens from the localnet or testnet faucet without leaving the CLI. (Implemented as `efctl env faucet`).

## Operational

- [ ] **Windows self-update** — The `update` command is currently disabled on Windows via build tags. Adding Windows support (atomic rename via `MoveFileEx`) would cover all platforms.
- [ ] **`efctl config show`** — Print the resolved configuration (merged defaults + `efctl.yaml` + env vars) to help debug configuration issues.
- [ ] **`efctl config validate`** — Validate the `efctl.yaml` file and report issues without running anything.
- [ ] **Telemetry opt-in / usage analytics** — Anonymous usage stats to understand which commands are used most, where failures happen, and what to prioritize.

## CI / Automation

- [ ] **Non-interactive / `--yes` mode** — Skip all confirmation prompts for use in CI pipelines and scripts.
- [ ] **JSON/machine-readable output** — A `--output json` flag on key commands (`status`, `graphql`, `version`) for programmatic consumption.
- [ ] **`efctl env wait`** — Block until the environment is fully ready (all containers healthy, chain producing blocks), useful as a CI gate step.
