## Context

efctl orchestrates a local Sui development environment via Docker containers. Two critical problems affect users on first-run or partial-init scenarios:

1. **pnpm/esbuild build blocker**: The sui-dev Docker image uses Node.js 24 with pnpm v11, which blocks esbuild build scripts by default. The in-code fix (`pnpm_patch.go` creating `pnpm-workspace.yaml` with `allowBuilds`) exists but has never been validated in e2e CI. Users encounter `ERR_PNPM_IGNORED_BUILDS` and have no diagnostic information to understand why.

2. **Mnemonic leak via `sui client`**: The `gatherSuiClient()` function in `efctl doctor` and `resolveAddress()` in summary output run `sui client` subcommands without checking whether `~/.sui/sui_config/client.yaml` exists. When no config file is present, the sui CLI auto-creates a new wallet and prints the BIP-39 mnemonic to stdout, which gets captured and displayed. This is a critical security vulnerability — the mnemonic appears in the user's terminal and any log capture.

Current code paths under risk:
- `pkg/doctor/doctor.go:562` — `gatherSuiClient()` runs 4 `sui client` subcommands
- `pkg/setup/summary.go:256` — `resolveAddress()` runs `sui client addresses --json`

## Goals / Non-Goals

**Goals:**
- Add pre-flight diagnostic logging (pnpm version, workspace yaml) to `CmdDeployWorld` for immediate issue diagnosis
- Add `pnpm approve-builds esbuild` fallback step as belt-and-suspenders with the existing `allowBuilds` config
- Guard all host-side `sui client` subcommand invocations with a config-existence check
- Eliminate mnemonic/credential leakage from `efctl doctor` and deployment summary output
- Bump version and release so affected users receive the fix

**Non-Goals:**
- BIP-39 mnemonic regex redaction/sanitization layer (defense-in-depth not needed; pre-check alone prevents the leak)
- Changing `ExecCapture` to suppress stderr globally (surgical pre-check at call sites is sufficient)
- Container-side pnpm/esbuild changes (handled by existing `pnpm_patch.go`; this change only hardens diagnostics and fallback)
- Automated e2e test runs (release validation is manual for now)

## Decisions

### 1. Config pre-check at call sites (not a shared wrapper)

**Decision**: Add an `os.Stat` check for `~/.sui/sui_config/client.yaml` directly in `gatherSuiClient()` and `resolveAddress()` rather than introducing a shared wrapper function.

**Alternatives considered**:
- Shared `suiClientGuard()` function: More abstraction but only 2 call sites — over-engineering for a simple check.
- Modify `ExecCapture` to auto-detect and suppress mnemonic output: Invasive, couples the executor to sui CLI semantics, and the user decided against the defense-in-depth layer.

**Rationale**: Two call sites, one simple check. A helper function is warranted only if a third site appears. Keep the fix minimal and localized. Extract the sui config path resolution into a small helper (`sui.ConfigPath()`) so both call sites stay consistent.

### 2. Diagnostic logging as command string (not Go-side orchestration)

**Decision**: Append the pnpm diagnostic echo and approve-builds to the `CmdDeployWorld` shell string rather than adding a separate exec phase in Go.

**Alternatives considered**:
- Go-side pre-exec phase in `DeployWorld()`: More control over output formatting, but changes the orchestrator flow and requires touching the deploy logic.
- Dockerfile-level patch to pin pnpm version: Invasive and fragile — the upstream Dockerfile is already heavily patched.

**Rationale**: `CmdDeployWorld` runs inside the container in one shot. Appending diagnostic echo + approve-builds as shell commands keeps the change in one place (the constant string), requires no new Go code in the orchestrator, and makes the diagnostic output visible in container logs regardless of how efctl captures them.

### 3. Config path resolution via `os.UserHomeDir` + known sui convention

**Decision**: Resolve the sui config path as `~/.sui/sui_config/client.yaml` using `os.UserHomeDir()`. This matches the sui CLI default and the existing `efctl doctor` output which references this path.

**Rationale**: This is the documented and only suconfig path used by the sui CLI. No need to read env vars or config files — it's a fixed convention.

### 4. Version bump to v0.3.2

**Decision**: Bump patch version since this is a security fix + bug fix with no API or behavior changes for correctly-functioning setups.

**Rationale**: Follows semver. The security fix justifies a patch release without waiting for a minor cycle.

## Risks / Trade-offs

| Risk | Mitigation |
|------|-----------|
| `pnpm approve-builds esbuild` may hang or prompt interactively | Redirect stderr to `/dev/null` and use `|| true` to ensure it never blocks the deploy pipeline |
| `~/.sui/sui_config/client.yaml` path could change in future sui CLI versions | Path is a well-established convention; if it changes, it's an obvious single-location fix |
| Diagnostic echo in `CmdDeployWorld` adds noise to container logs | Output only printed on first-run or failure scenarios; bounded to ~5 lines |
| Version bump without automated e2e validation in CI | Manual e2e test pass required before tag; document in release notes |
