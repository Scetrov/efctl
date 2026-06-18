## 1. Sui Config Guard — pkg/sui/ helper

- [x] 1.1 Add `SuiConfigPath()` function to `pkg/sui/sui.go` that resolves `~/.sui/sui_config/client.yaml` using `os.UserHomeDir()`
- [x] 1.2 Add `SuiConfigExists()` function to `pkg/sui/sui.go` that returns true if the config file exists
- [x] 1.3 Add unit tests for `SuiConfigPath()` and `SuiConfigExists()` in `pkg/sui/sui_test.go`

## 2. Sui Config Guard — pkg/doctor/doctor.go

- [x] 2.1 Update `gatherSuiClient()` to check `sui.SuiConfigExists()` before running any `sui client` subcommand
- [x] 2.2 When config does not exist: set `Found: true`, `ActiveEnv: "not configured"`, skip all sui commands
- [x] 2.3 Add unit tests for the config pre-check in `pkg/doctor/doctor_test.go`

## 3. Sui Config Guard — pkg/setup/summary.go

- [x] 3.1 Update `resolveAddress()` to check `sui.SuiConfigExists()` before running `sui client addresses --json`
- [x] 3.2 When config does not exist: return empty string (caller falls through to `deriveAddress()`)
- [x] 3.3 Add unit tests in `pkg/setup/summary_test.go`

## 4. PNPM Diagnostics — pkg/setup/constants.go

- [x] 4.1 Update `CmdDeployWorld` to prepend: pnpm version echo, workspace yaml cat (with "not found" fallback), and `pnpm approve-builds esbuild 2>/dev/null || true` before the existing install + deploy commands
- [x] 4.2 Verify the updated command string is valid shell syntax and does not introduce pipeline failures
- [x] 4.3 Add unit test for command format in `pkg/setup/constants_test.go`

## 5. Release

- [x] 5.1 Bump version to v0.3.2 (update version constant in `cmd/version.go` or equivalent)
- [x] 5.2 Run full test suite: `go test ./...`, `gosec -quiet ./...`
- [x] 5.3 Run `go build` and verify binary version output
- [x] 5.4 Create git tag `v0.3.2` and release notes mentioning issue #38 + security fix
