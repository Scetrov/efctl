# Tasks: upgrade-x-text-osv-daily-scan

## 1. Dependency remediation (GO-2026-6629)

- [x] 1.1 Upgrade `golang.org/x/text` to v0.42.0 (`go get golang.org/x/text@v0.42.0 && go mod tidy`)
- [x] 1.2 Verify the `go.mod`/`go.sum` diff is limited to `golang.org/x/text` and `go mod verify` passes
- [x] 1.3 Confirm no selected module remains in the GO-2026-6629 affected range (`go list -m golang.org/x/text`)

## 2. Daily OSV-Scanner workflow

- [x] 2.1 Create `.github/workflows/osv-scan.yml` with `schedule` (daily cron) + `workflow_dispatch` triggers and `permissions: { contents: read, security-events: write }`
- [x] 2.2 Add checkout (SHA-pinned) and the SHA-pinned `google/osv-scanner/actions/osv-scanner` (v2.6.0) step writing `osv-results.sarif` with `--fail`
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
- [ ] 4.4 Optionally run `osv-scanner scan --lockfile go.mod` locally to confirm a clean SARIF

## 5. Ship

- [x] 5.1 Commit (signed, conventional commit, agent/model attribution) and push a branch
- [x] 5.2 Open the PR against `main` referencing Code Scanning alert #7 and GO-2026-6629
- [x] 5.3 After CI passes, confirm the SARIF upload path (via `workflow_dispatch` if possible) and archive the OpenSpec change
