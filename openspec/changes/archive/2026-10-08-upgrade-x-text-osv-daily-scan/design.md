# Design: upgrade-x-text-osv-daily-scan

## Context

efctl's dependency graph resolves `golang.org/x/text` (indirect) to v0.40.0. Code Scanning alert #7 reports GO-2026-6629 (CVE-2026-56851): a panic (out-of-bounds slice) in `golang.org/x/text/secure/precis` when transforming crafted input, fixed in v0.41.0. The latest release is v0.42.0.

The repository already runs Scorecard (which includes an OSV-derived Vulnerabilities check) and CodeQL on push/PR, but has no dedicated, frequently-scheduled dependency scanner, so newly published advisories for the dependency graph are only caught at the next push/PR. A previous change (`go-dependency-security` spec) already remediated GO-2026-5970 by pinning x/text to v0.40.0; this change supersedes that version target.

Constraints: all GitHub Actions are SHA-pinned in this repository; commits must be signed; `pre-commit` must pass; no unrelated dependency churn; temporary files go in `./tmp`.

## Goals / Non-Goals

**Goals:**
- Raise `golang.org/x/text` to v0.42.0 so GO-2026-6629 is out of the affected range, with `go.mod`/`go.sum` changes limited to that module.
- Add a daily `osv-scanner` job that emits SARIF and publishes it to GitHub Code Scanning via `upload-sarif`.

**Non-Goals:**
- Remediation of any other currently-unreported vulnerabilities.
- Changing CodeQL, Scorecard, CI, or release workflows.
- Fail-closed PR gating on OSV findings in pull_request events (out of scope; scan is scheduled only).

## Decisions

1. **Upgrade to v0.42.0, not the minimum fix v0.41.0.**
   Constitution requires the latest applicable released version, and v0.42.0 is a superset of the v0.41.0 fix. Alternative considered: v0.41.0 to minimize churn — rejected because it would immediately be one release behind.

2. **Use the official `google/osv-scanner-action` Docker action, SHA-pinned.**
   Pin `google/osv-scanner-action/osv-scanner-action@a345acffa64b0eaede81a3d9aae6141214d9c8fc` (tag v2.6.0, verified via `git/ref/tags`), which runs the `ghcr.io/google/osv-scanner-action:v2.6.0` image, so the action ref and the image tag are released together. Scan arguments are passed via the action's `scan-args` input (`--lockfile go.mod --lockfile go.sum --format sarif --output osv-results.sarif --fail`). Alternative considered: the legacy `google/osv-scanner/actions/scanner` action — rejected (upstream marks it experimental/legacy and redirects to `google/osv-scanner-action`).

3. **Reuse the already-pinned `github/codeql-action/upload-sarif@2892aa5e19bbd11bc0cff5427e3b750a04d9e3c2` (v4.38.2).**
   Same action/SHA already used in `codeql.yml`, so no new action provenance. Use `category: osv-scan` so OSV alerts are grouped separately from CodeQL alerts.

4. **Scan runs with `--fail`; `upload-sarif` runs with `if: always()`.**
   A failing scan marks the scheduled run red (visible in Actions) while the SARIF is always uploaded so Code Scanning alerts are always created/updated. Alternative considered: never fail — rejected because silent green runs reduce detection signal.

5. **Least-privilege token permissions on the workflow.**
   `permissions: { contents: read, security-events: write }` — nothing else. The scheduled GITHUB_TOKEN is not needed for writing contents or checking out beyond read.

6. **Triggers: `schedule` (daily, `cron: "37 8 * * *"`) plus `workflow_dispatch`.**
   Daily cadence offsets the exact hour to reduce runner contention; manual dispatch allows on-demand rescans. No `pull_request` trigger per scope decision.

7. **SARIF written to the workspace, not a path outside it.**
   Output to `osv-results.sarif` in the job workspace, uploaded to Code Scanning, then removed (not retained as an Actions artifact) to avoid unauthenticated artifact exposure of vulnerability data.

8. **Enforce workflow quality with actionlint (pinned v1.7.12), in pre-commit and CI.**
   actionlint is a static checker for workflow YAML (syntax, `${{ }}` expression typing, action inputs/outputs, cron syntax, script-injection checks). It has no SARIF flag, so SARIF output uses its Go-template `-format` option with the SARIF 2.1.0 template (vendored from actionlint's `testdata/format/sarif_template.txt` at tag v1.7.12 into `scripts/actionlint-sarif-template.txt`). Both pre-commit and CI invoke `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12` (module-hash-verified by the Go toolchain, no binary download). The official pre-commit hook (`repo: rhysd/actionlint`, `id: actionlint`) was rejected because its `entry: actionlint` points at the root *library* package while the main package lives at `cmd/actionlint`, so `language: golang` cannot build it; a `language: system` local hook keeps the same pinned invocation and avoids depending on pre-commit 3.0 behavior.

9. **SHA-pin policy as an explicit gate, not a lint rule.**
   actionlint treats tag references (`actions/checkout@v4`) as syntactically valid and will not fail them. The pinning policy (every `uses:` must match `@<40-char sha> # v<version>`) is therefore a small dedicated script `scripts/check-action-pinning.sh`, wired both as a local pre-commit hook (scoped to `.github/workflows/`) and as the first step of the CI `workflow-lint` job so the policy holds even on fresh clones.

10. **actionlint SARIF published to Code Scanning under its own category.**
    The CI `workflow-lint` job: (1) runs the pinning gate, (2) runs actionlint as the gate (non-zero exit fails the job), (3) with `if: always()` regenerates the SARIF file, and (4) uploads it via the already-pinned `upload-sarif@v4.38.2` with `category: workflow-lint`. The job declares `security-events: write` (workflow keeps its `contents: read` default).

## Risks / Trade-offs

- [New advisories published between now and merge] → the daily scan will surface them as fresh alerts; triage follows the repo's reproduce-then-fix loop rather than blind upgrades.
- [`x/text` v0.42.0 pulls new transitive checksums] → constrained by checking the `go.mod`/`go.sum` diff; `go mod verify` plus full test suite and `pre-commit` gate the merge.
- [Daily `--fail` runs create noise for long-unfixed findings] → acceptable: red scheduled runs are the intended visibility channel; alerts deduplicate by location.
- [osv-scanner false positives on lockless module scanning] → `osv-scanner` reads `go.mod`/`go.sum` for Go projects and reports the selected graph; findings are advisory until confirmed.
- [SARIF template is maintained upstream in actionlint testdata] → template is vendored into `scripts/` and only updated when actionlint is upgraded; a mismatch would fail the `upload-sarif` step, making drift visible.
- [actionlint false positives on existing workflows] → the new CI job runs on the current tree during PR validation; any findings must be fixed (or `-ignore`d with justification) before merge, keeping the gate clean at introduction.

## Migration Plan

1. Merge the dependency bump + workflow in a single PR (as requested).
2. After merge, wait for the first scheduled run (or trigger `workflow_dispatch`) to confirm SARIF upload and alert-tool registration in Code Scanning.
3. Confirm alert #7 (GO-2026-6629) transitions to stale/closed on next Scorecard/CodeQL re-evaluation.
4. Rollback: revert the single commit; the module returns to v0.40.0 (re-opening the finding until re-bump).

## Open Questions

- None blocking. Follow-ups (separate changes): `pull_request` triggers for the OSV scan (fail-closed PR gating); `actionlint` config for self-hosted runner labels if self-hosted runners are adopted.
