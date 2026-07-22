## Why

OpenSSF Scorecard detects GO-2026-5970 because efctl selects `golang.org/x/text` v0.38.0, which is below the first fixed release. Although current symbol-level analysis finds no reachable vulnerable call, retaining the affected module version creates latent denial-of-service risk and lowers the repository's Scorecard Vulnerabilities result.

## What Changes

- Upgrade the selected `golang.org/x/text` module to a release that contains the upstream fix for GO-2026-5970, targeting the current v0.40.0 release and treating v0.39.0 as the minimum secure version.
- Refresh and verify Go module checksums without broad, unrelated dependency upgrades.
- Validate compilation, existing behavior, module integrity, and vulnerability-scan results through the repository's established quality gates.
- Confirm that post-merge OpenSSF Scorecard output no longer reports GO-2026-5970.

## Capabilities

### New Capabilities

- `go-dependency-security`: Defines the security and verification requirements for selecting a remediated Go dependency version and preventing GO-2026-5970 from remaining in efctl's dependency graph.

### Modified Capabilities

None.

## Impact

- Affects `go.mod` and `go.sum`; no application API or CLI behavior is intended to change.
- Updates the indirect `golang.org/x/text` dependency used through PTerm and fuzzysearch.
- Exercises existing Makefile, pre-commit, test, module-verification, govulncheck, and OpenSSF Scorecard security surfaces.
- Requires Go 1.25 or later for the selected `golang.org/x/text` release; efctl already declares Go 1.26.5.
