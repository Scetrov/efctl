# Proposal: upgrade-x-text-osv-daily-scan

## Why

GitHub Code Scanning alert #7 reports GO-2026-6629 (CVE-2026-56851): a panic via out-of-bounds slice when `golang.org/x/text/secure/precis` transforms crafted input. The affected module `golang.org/x/text` is currently at v0.40.0 and the vulnerability is fixed in v0.41.0. In addition, the repository has no continuous OSV-based dependency scanning; vulnerabilities are only surfaced when Scorecard/CodeQL happen to run, so future findings can sit undetected between scans.

## What Changes

- Upgrade `golang.org/x/text` from v0.40.0 to v0.42.0 (latest release, includes the v0.41.0 fix for GO-2026-6629) with no unrelated dependency changes.
- Add a new GitHub Action `osv-scan.yml` that runs `osv-scanner` on a daily schedule (`cron` on the default branch), emits SARIF to a local artifact, and uploads it to GitHub Code Scanning via `github/codeql-action/upload-sarif@v3`.
- Add a `workflow-linting` gate: `actionlint` (pinned v1.7.12) lints all workflow YAML in pre-commit and CI, a strict check enforces that every `uses:` reference is a full 40-character SHA pin with an adjacent `# v<version>` comment, and CI publishes actionlint results as SARIF to Code Scanning.

## Capabilities

### New Capabilities
- `osv-daily-scanning`: Daily OSV-Scanner dependency scan of the repository that produces SARIF output and publishes it to GitHub Code Scanning.
- `workflow-linting`: Static linting of GitHub Actions workflow files with actionlint (SARIF published to Code Scanning) plus an enforced SHA-pin-with-version-comment policy for all `uses:` references.

### Modified Capabilities
- `go-dependency-security`: Target version for the remediated `golang.org/x/text` module changes from v0.40.0 to v0.42.0; scan-exit criteria must now also cover GO-2026-6629.

## Impact

- `go.mod` / `go.sum`: version and checksum lines for `golang.org/x/text` only.
- `.github/workflows/osv-scan.yml`: new workflow; no changes to CI, CodeQL, or release pipelines.
- `.github/workflows/ci.yml`: new `workflow-lint` job (actionlint gate + SARIF upload + pinning gate).
- `.pre-commit-config.yaml`: new actionlint hook (pinned rev) and local pinning-gate hook.
- `scripts/`: new `check-action-pinning.sh` gate and `actionlint-sarif-template.txt` (SARIF 2.1.0 template from actionlint testdata).
- Security posture: Code Scanning alert #7 (GO-2026-6629) becomes stale/closed after the merge; new findings from OSV appear under the "OSV-Scanner" tool name in Code Scanning.
- No application behavior changes expected: `golang.org/x/text` is an indirect dependency and no `precis` call paths are exercised by efctl code.
