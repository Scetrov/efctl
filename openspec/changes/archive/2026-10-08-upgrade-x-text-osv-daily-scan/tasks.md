# Tasks: upgrade-x-text-osv-daily-scan

## 1. Dependency remediation (GO-2026-6629)

- [x] 1.1 Upgrade `golang.org/x/text` to v0.42.0 (`go get golang.org/x/text@v0.42.0 && go mod tidy`)
- [x] 1.2 Verify the `go.mod`/`go.sum` diff is limited to `golang.org/x/text` and `go mod verify` passes
- [x] 1.3 Confirm no selected module remains in the GO-2026-6629 affected range (`go list -m golang.org/x/text`)

## 2. Daily OSV-Scanner workflow

- [x] 2.1 Create `.github/workflows/osv-scan.yml` with `schedule` (daily cron) + `workflow_dispatch` triggers and `permissions: { contents: read, security-events: write }`
- [x] 2.2 Add checkout (SHA-pinned) and the SHA-pinned `google/osv-scanner-action/osv-scanner-action` (v2.6.0) step scanning `go.mod`, writing `osv-results.sarif` with `--output-file`, and using the default non-zero exit on vulnerabilities
- [x] 2.3 Add `github/codeql-action/upload-sarif@2892aa5e19bbd11bc0cff5427e3b750a04d9e3c2` with `if: always()`, `sarif_file: osv-results.sarif`, `category: osv-scan`, and a cleanup step removing the SARIF
- [x] 2.4 Validate workflow syntax (e.g., `actionlint` or `ruby -ryaml -e` parse) and confirm all `uses:` entries are full SHA pins

## 3. Workflow linting gate (actionlint + SHA-pin policy)

- [x] 3.1 Vendor the actionlint SARIF 2.1.0 template into `scripts/actionlint-sarif-template.txt`
- [x] 3.2 Add `scripts/check-action-pinning.sh` (fail on any `uses:` not matching `@<40-char-sha> # v<version>`), executable
- [x] 3.3 Add actionlint (rev v1.7.12) + pinning-gate hooks to `.pre-commit-config.yaml`
- [x] 3.4 Add `workflow-lint` job to `ci.yml`: pinning gate, actionlint gate, SARIF generation (`if: always()`), `upload-sarif` with `category: workflow-lint` and `security-events: write`

## 4. Verification

- [x] 4.1 Run `go build ./...`, `go test ./...`, and `govulncheck ./...` (confirm GO-2026-6629 absent)
- [x] 4.2 Run `scripts/check-action-pinning.sh` and actionlint locally against all workflows
- [x] 4.3 Run `pre-commit run --all-files` and fix any findings
- [x] 4.4 Run the pinned OSV action image locally against `go.mod` and confirm clean SARIF (2.1.0, zero findings)

## 5. Ship

- [x] 5.1 Commit (signed, conventional commit, agent/model attribution) and push a branch
- [x] 5.2 Open the PR against `main` referencing Code Scanning alert #7 and GO-2026-6629
- [ ] 5.3 After CI passes, confirm the SARIF upload path (via `workflow_dispatch` if possible) and archive the OpenSpec change

## Finalization evidence

- Implementation merged in PR #91; initial spec sync/archive merged in PR #92, which retained a stale active copy.
- Alert #7 (GO-2026-6629) is fixed in GitHub Code Scanning.
- Manual run https://github.com/Scetrov/efctl/actions/runs/37782620217 exposed unsupported `--fail` and `go.sum` inputs before SARIF generation. Both failures were reproduced locally with the v2.6.0 action image (digest `sha256:13cef841c7b8de79248e572de0c278d64eaf0a4006e2973fa690b777be30eee4`).
- Corrected arguments produced SARIF 2.1.0 with zero findings locally. A regression test rejects the original invalid arguments.
- The three synced main specs now have required Purpose/Requirements headers and pass OpenSpec validation. The stale active copy is removed; this archive is retained as the canonical record.
- Hosted verification of the corrected workflow remains pending.
