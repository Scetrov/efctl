# Design: Bump Go baseline to 1.27.2

## Context

The baseline is a version-pinning spec plus a Go toolchain guard test. The
1.27.1 → 1.27.2 change is mechanical except for two toolchain-compatibility
consequences that must be handled so the quality gates stay green in the hermetic
pre-commit sandbox.

## Decisions

### Go 1.27.2 V5 export data requires a newer x/tools for gosec

Go 1.27.2 emits V5 export data. `gosec v2.29.0` (the latest release) still requires
`golang.org/x/tools v0.49.0` by default, whose `pkgbits` decoder predates V5 and
cannot parse 1.27.2 packages. A plain `go install gosec@v2.29.0` therefore builds a
scanner that silently skips SSA analysis or fails. We build `gosec` from a
repository-local wrapper module under `tmp/gosec-build` with
`replace golang.org/x/tools => golang.org/x/tools v0.52.0` (the first x/tools release
line with V5 support), producing `tmp/go127-tools/bin/gosec`. The replace is recorded
in the toolchain doc and the gosec module hash is integrity-checked via the build
info.

### `go run pkg@version` deprecation lookup breaks under GOPROXY=off

Go 1.27 makes `go run pkg@version` perform a module deprecation lookup that requires
a proxy version-list query. The hermetic pre-commit sandbox sets `GOPROXY=off` with a
read-only module cache, so the lookup fails with `module lookup disabled by
GOPROXY=off`. `actionlint` is therefore installed as a local binary
(`tmp/go127-tools/bin/actionlint`) and the pre-commit hook invokes `actionlint`
directly, matching the existing `gosec`/`govulncheck` local-binary pattern resolved
via `EFCTL_ISOLATED_TOOLBIN`. CI (`.github/workflows/ci.yml`) still uses
`go run pkg@version` because it has network access.

## Alternatives considered

- Per-hook `env: {GOPROXY: ...}` override in `.pre-commit-config.yaml`: rejected —
  pre-commit does not support `env:` on `language: system` hooks and warns/ignores it.
- Keeping `go run` for actionlint: rejected — fundamentally incompatible with
  `GOPROXY=off` in Go 1.27; no GODEBUG or environment bypass exists for the
  deprecation lookup path.

## Risks / Trade-offs

- `gosec` is built from a replace-pinned `x/tools`; the module hash is recorded so the
  build is reproducible and integrity-checked.
- The scanner set is recorded as a point-in-time "latest released" set; the doc
  instructs re-checking released versions on future updates.
