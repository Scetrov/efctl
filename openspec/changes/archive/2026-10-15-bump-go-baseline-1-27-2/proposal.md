# Bump Go baseline to 1.27.2

## Why

Go 1.27.1's standard-library `net/http/internal/http2` carries vulnerabilities that
`govulncheck` (v1.8.0) reports through efctl's `pkg/graphql`, `pkg/env`, and
`cmd/root.go` import traces. Go 1.27.2 fixes those standard-library findings. Raising
the build baseline to 1.27.2 clears them without touching any application dependency.

Two toolchain-compatibility consequences must also be handled so the quality gates
stay green in the hermetic pre-commit sandbox:

- Go 1.27.2 emits version 5 (V5) export data, which the released `gosec` scanner's
  default `golang.org/x/tools` (v0.49.0) cannot decode.
- Go 1.27's `go run pkg@version` deprecation lookup requires a proxy version-list
  query that the hermetic sandbox (`GOPROXY=off`, read-only module cache) cannot satisfy.

## What Changes

- Raise the `go.mod` `go` directive and the toolchain guard from 1.27.1 to 1.27.2.
- Update source-build documentation, the `go-toolchain-baseline` spec, and the guard
  tests to 1.27.2.
- Rebuild `gosec v2.29.0` with an `x/tools v0.52.0` replace so it parses V5 export
  data; record its checksum-authenticated module hash.
- Install `actionlint v1.7.12` as a local binary and invoke it from the pre-commit
  hook instead of `go run pkg@version`.
- Record the updated scanner set and module hashes in `docs/go-toolchain.md`.

## Non-goals

- No application dependency changes.
- No change to Sui image contents or lifecycle behavior.
- No change to CI workflow scanner invocation (CI still uses `go run pkg@version`
  because it has network access).

## Impact

- Affected specs: `go-toolchain-baseline`.
- Affected files: `go.mod`, `toolchain_test.go`, `README.md`,
  `docs/go-toolchain.md`, `.pre-commit-config.yaml`,
  `openspec/specs/go-toolchain-baseline/spec.md`.
