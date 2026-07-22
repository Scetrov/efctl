## Context

efctl currently selects `golang.org/x/text` v0.38.0 as an indirect dependency. PTerm and fuzzysearch lead to the affected `golang.org/x/text/unicode/norm` package, but `govulncheck` finds no current call path from efctl to the vulnerable symbols. OpenSSF Scorecard uses module-version matching through OSV and therefore reports GO-2026-5970 regardless of present symbol reachability.

The upstream fix is available from v0.39.0. The current release is v0.40.0, both fixed releases require Go 1.25, and efctl declares Go 1.26.5. Temporary module-file simulations with v0.39.0 and v0.40.0 compiled all project test packages and produced a clean govulncheck result without modifying the working tree.

## Goals / Non-Goals

**Goals:**

- Remove the vulnerable `golang.org/x/text` version from efctl's selected module graph.
- Clear GO-2026-5970 from govulncheck and OpenSSF Scorecard results.
- Keep the remediation narrowly scoped to module metadata and checksums.
- Preserve current CLI behavior and compatibility with direct dependencies that consume `golang.org/x/text`.
- Verify the change with existing tests and security gates.

**Non-Goals:**

- Refactor or replace PTerm, fuzzysearch, or other consumers of `golang.org/x/text`.
- Add an OSV suppression for GO-2026-5970.
- Perform a broad Go dependency refresh.
- Change efctl APIs, commands, output, or supported workflows.
- Backport or maintain a fork of the upstream security patch.

## Decisions

### Select `golang.org/x/text` v0.40.0 directly

Update the existing indirect requirement from v0.38.0 to v0.40.0 and regenerate its checksums. v0.39.0 is the security floor, while v0.40.0 is the current release and has already passed temporary compilation and vulnerability-scan checks against this repository.

A targeted update is preferred over `go get -u ./...` because it minimizes unrelated module movement and simplifies review. Selecting v0.39.0 was considered as the smallest possible security delta, but it would leave the repository immediately behind the current release without reducing the relevant compatibility requirement.

### Remediate rather than suppress

Do not add `osv-scanner.toml` or otherwise ignore GO-2026-5970. Current symbol reachability reduces immediate exploitability, but the affected package remains in the build graph and future call paths could make the vulnerable symbols reachable. Removing the affected version provides a durable result across both reachability-aware and version-based scanners.

### Use existing quality gates as acceptance evidence

No application-level regression test is needed for an upstream dependency implementation defect that efctl does not directly invoke. Verification will instead combine focused module-diff review, `go mod tidy`, `go mod verify`, compilation and existing tests, `govulncheck`, and the repository-required pre-commit hooks. The post-merge Scorecard run provides external confirmation because the workflow executes on pushes to `main` and on its schedule.

### Preserve the indirect dependency classification

The requirement remains marked `// indirect` because efctl does not import `golang.org/x/text` directly. The root module requirement deliberately raises the version selected by Minimal Version Selection for transitive consumers without introducing a new application import.

### Pin PTerm's synchronization fix

The initial race-enabled verification exposed unsynchronized spinner lifecycle access in the latest released PTerm version (`v0.12.83`). The relevant synchronization fix exists on the upstream default branch but has not been released, so the remediation pins PTerm to pseudo-version `v0.12.84-0.20260711211409-bacb2fc434b3`. This is an explicitly justified exception to the narrow dependency-update scope: it is required to make the mandated race-enabled test suite pass without masking the race. The resulting `atomicgo.dev/keyboard` update and removal of unused `github.com/gookit/color` entries are transitive module-graph effects of this targeted PTerm update.

## Risks / Trade-offs

- [A newer v0 minor release can contain compatibility changes] → Keep the update isolated, inspect the module diff, and run compilation, unit, integration, and pre-commit gates.
- [A broad tidy operation could move unrelated modules] → Review `go.mod` and `go.sum` and reject unrelated version changes; the expected semantic change is only `golang.org/x/text`.
- [Scorecard confirmation is delayed until a `main` push or scheduled run] → Use local govulncheck as immediate evidence and verify the next published Scorecard result after merge.
- [Rolling back to v0.38.0 would reintroduce the finding] → If v0.40.0 causes an unforeseen regression, move to the minimum fixed v0.39.0 rather than restoring the vulnerable version.

## Migration Plan

1. Update the indirect `golang.org/x/text` requirement to v0.40.0.
2. Run `go mod tidy` and verify that only the intended module metadata and checksum entries change.
3. Run module verification, tests, govulncheck, and all pre-commit hooks.
4. Merge through the normal review process and inspect the post-merge Scorecard result.
5. If v0.40.0 introduces a confirmed incompatibility, switch to v0.39.0 and rerun the same gates.

## Open Questions

None. The fixed version, target version, compatibility floor, and verification path are known.
