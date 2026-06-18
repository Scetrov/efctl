## Why

`efctl env up` fails on fresh WSL2 Ubuntu 24.04 environments because pnpm v11 (shipped in the upstream sui-dev Docker image) blocks esbuild build scripts by default, preventing world-contracts deployment. The fix (pnpm-workspace.yaml allowBuilds config) exists in code but is unreleased and unverified in e2e CI. Additionally, when `efctl env up` fails partway through and the user runs `efctl doctor`, the sui CLI auto-creates a wallet and leaks the BIP-39 mnemonic (secret recovery phrase) into the doctor output — a critical security vulnerability.

## What Changes

- **Harden `CmdDeployWorld`** to emit pnpm diagnostic info (version, workspace yaml contents) before install, and run `pnpm approve-builds esbuild` as a belt-and-suspenders fallback before `pnpm install`.
- **Add config existence pre-check in `gatherSuiClient()`** (`pkg/doctor/doctor.go`): before running any `sui client` subcommand, verify `~/.sui/sui_config/client.yaml` exists. If it does not, report `not configured` and skip all sui commands entirely — preventing auto-wallet-creation and mnemonic leak.
- **Apply the same pre-check to `resolveAddress()`** (`pkg/setup/summary.go`) to prevent the same leak during deployment summary output.
- **Add unit tests** for the config pre-check logic and the pnpm diagnostic command format.
- **Bump version** to v0.3.2 and create a release so affected users receive the fix.

## Capabilities

### New Capabilities
- `sui-config-guard`: Pre-check for `~/.sui/sui_config/client.yaml` existence before running `sui client` subcommands on the host, preventing wallet auto-creation and credential leakage.
- `pnpm-diagnostics`: Diagnostic logging and fallback approval before `pnpm install` in the deploy-world command, providing clear failure diagnostics when pnpm/esbuild issues recur.

### Modified Capabilities
<!-- (none — no existing specs) -->

## Impact

- **pkg/setup/constants.go**: `CmdDeployWorld` string gains diagnostic echo + approve-builds step.
- **pkg/doctor/doctor.go**: `gatherSuiClient()` gains config file existence check.
- **pkg/setup/summary.go**: `resolveAddress()` gains the same guard.
- **pkg/setup/constants_test.go** (new or updated): Tests for command format.
- **pkg/doctor/doctor_test.go**: Tests for config pre-check behavior.
- **pkg/setup/summary_test.go**: Tests for resolveAddress guard.
- **Version**: Bump to v0.3.2; affects `cmd/version.go` or equivalent + release tag.
- Security: Eliminates plaintext mnemonic exposure in doctor output.
- Reliability: Diagnostic logging makes future pnpm/esbuild failures immediately diagnosable without requiring users to run efctl again.
