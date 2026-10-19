## 1. Establish Baseline and Update Dependency

- [x] 1.1 Run `govulncheck -show verbose ./...` before the update and record that GO-2026-5970 is present at package level but has no reachable vulnerable symbol path.
- [x] 1.2 Update only the indirect `golang.org/x/text` requirement from v0.38.0 to v0.40.0 and run `go mod tidy` to refresh module metadata and checksums.
- [x] 1.3 Review the `go.mod` and `go.sum` diff and remove or explicitly justify any dependency changes unrelated to `golang.org/x/text`.

## 2. Verify Security and Compatibility

- [x] 2.1 Run `go mod verify` and confirm `go list -m golang.org/x/text` resolves to v0.40.0.
- [x] 2.2 Run the standard and race-enabled Go test suites and confirm existing efctl behavior remains unchanged.
- [x] 2.3 Run `govulncheck -show verbose ./...` against the updated graph and confirm it succeeds without reporting GO-2026-5970.
- [x] 2.4 Run `pre-commit run --all-files` and resolve every formatting, linting, security, and test failure attributable to the change.

## 3. Confirm Published Security Result

- [x] 3.1 After the remediation reaches the default branch, inspect the next OpenSSF Scorecard result and confirm its Vulnerabilities check no longer reports GO-2026-5970.
