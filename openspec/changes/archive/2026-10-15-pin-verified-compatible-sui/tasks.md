## 1. Establish candidate and failure evidence

- [x] 1.1 Recheck the official Sui release history and JSON-RPC removal notes; confirm `testnet-1.81.1` is the latest testnet release that still serves fullnode JSON-RPC. (Red signal source: PR #98 lifecycle failure — HTTP 404 from `sui_getNormalizedMoveFunction` on testnet-v1.82.1.)
- [x] 1.2 Download the official amd64 and aarch64 release assets into `./tmp` and verify their SHA-256 digests: x86_64 `d921910fdedf5ae2bc3f08a7236c57819a457ce53a8cc60c85f3653c7831b5d4`, aarch64 `4c2d948417af9f55cdd1f2f5d8e558533502672004c34f6429aa6cb114a51583`.
- [x] 1.3 Run the controlled one-shot container build of the actual generated block (opt-in `EFCTL_PIN_VALIDATE=1`): image builds, `sui --version` reports `sui 1.81.1-e04cf9aa915c`, and a live node answers JSON-RPC — `sui_getChainIdentifier` returns a chain id and `sui_getNormalizedMoveFunction` returns a full function definition for `coin::balance`.

## 2. Add failing regression coverage (test-first)

- [x] 2.1 Update unit tests to expect the pinned block and demonstrate the old floating implementation fails them (red).
- [x] 2.2 Add regression tests for the pinned-not-floating default block and legacy floating-block migration to the pinned block; cover idempotence and custom precedence alongside the existing transition coverage.

## 3. Implement the pinned default installation

- [x] 3.1 Change `globalSuiBlock` to install `sui@testnet-1.81.1` through the existing suiup-managed path, with the verified reference digests recorded as non-enforced comments.
- [x] 3.2 Add `legacyGlobalSuiBlock` and register it in managed-instruction recognition so existing scaffolds migrate in place on the next preparation.
- [x] 3.3 Run focused unit and container tests to green and confirm the full suite (`go vet ./...`, `go test ./...`) passes with no unrelated changes.

## 4. Validate and document

- [x] 4.1 Verify the pinned release in a real container build (task 1.3 evidence), keeping temporary artifacts under `./tmp`.
- [x] 4.2 Run the existing full E2E lifecycle (`TestE2E_FullLifecycle`, `-tags e2e`) against the pinned release and confirm required lifecycle subtests execute and pass without adding a new skip pattern. Result: `TestE2E_FullLifecycle` PASS in 879.83s — all 11 subtests (env_up, extension_init/build/test/publish/idempotent, env_run, faucet_config, sui_doctor, env_down) pass, zero skips, on a build that installed `sui@testnet-1.81.1` through the pinned block.
- [x] 4.3 Write the pin maintenance document: pinned tag and reference-digest provenance, the deliberate deviation from build-time hash enforcement, the known floating suiup installer component, and the release-update checklist (re-verify digests, version, JSON-RPC compatibility, full lifecycle, security advisories).
- [x] 4.4 Run all pre-commit hooks. Result: every hook passes except two **pre-existing, environment-only** failures that reproduce identically on clean `HEAD` (verified in a throwaway worktree): `gosec` exits 1 with `0 Issues` because the installed `gosec` binary was built with Go 1.26 while the module requires Go 1.27 (SSA build skew); and `govulncheck` reports 8 standard-library `net/http/internal/http2` vulnerabilities from the local Go 1.27.1 toolchain (fixed in 1.27.2) via traces through `pkg/graphql`, `pkg/env`, and `cmd/root.go`, none of which this change touches. This change introduces no new hook failures.

## 5. Finalize

- [x] 5.1 Review implementation and specs for consistency; confirm no floating fallback, no enforced-digest claim, and no unrelated transport/readiness changes.
- [x] 5.2 After all required tasks pass, archive this change before the final commit or PR; keep unavailable verification tasks unchecked and report blockers truthfully.
