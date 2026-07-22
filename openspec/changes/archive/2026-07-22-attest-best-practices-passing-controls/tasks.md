## 1. Confirm Evidence Baseline

- [x] 1.1 Record the audited list of 43 mandatory Passing controls and map each control to its public repository evidence or required maintainer attestation.
- [x] 1.2 Confirm whether private vulnerability reports were received during the applicable six-month window and select an accurate `vulnerability_report_response` status and justification.
- [x] 1.3 Audit every existing SemVer tag and release page to identify the human-readable summary and any publicly known runtime vulnerability fixed by each release.

## 2. Add Failing Compliance Tests

- [x] 2.1 Add a failing repository-level Go test that requires `.bestpractices.json`, exactly covers the audited mandatory status keys, permits only `Met` or `N/A`, and requires paired justifications.
- [x] 2.2 Extend the failing test to require HTTPS evidence for URL-required controls and a non-empty applicability explanation for every `N/A` answer.
- [x] 2.3 Add a failing release-history test that requires an `Unreleased` section and a changelog heading for every audited SemVer release.

## 3. Close Documentation Gaps

- [x] 3.1 Add `CHANGELOG.md` with concise human-readable summaries for every existing release and an `Unreleased` section for future user-visible and security changes.
- [x] 3.2 Update `CONTRIBUTING.md` to require tests for new functionality and bug fixes and to document the standard formatting, warning, security, unit, integration, and E2E checks.
- [x] 3.3 Document contributor expectations for adding human-readable release notes and identifying applicable publicly known runtime vulnerabilities.
- [x] 3.4 Document the post-merge bestpractices.dev owner workflow, including proposal review, save, project 13754 verification, and rollback considerations.

## 4. Create the Passing-Control Attestation

- [x] 4.1 Add top-level `.bestpractices.json` with evidence-backed `Met` or justified `N/A` proposals for all 43 mandatory Passing controls.
- [x] 4.2 Ensure repository-observable claims cite precise maintained-file, workflow, issue, release, tag, or security-feature evidence and that private claims match the maintainer confirmation from task 1.2.
- [x] 4.3 Reconcile any claim that existing repository enforcement cannot support by strengthening the relevant documentation or quality gate, or by correcting the proposed status.
- [x] 4.4 Make the compliance and release-history tests pass without network access.

## 5. Expose and Verify the Badge

- [x] 5.1 Add the live bestpractices.dev project 13754 badge to `README.md` without hard-coding a passing state.
- [x] 5.2 Run `go test ./...` and verify the new invariant tests fail clearly when a mandatory status, justification, required URL, or changelog heading is intentionally omitted.
- [x] 5.3 Run all pre-commit hooks and resolve every formatting, lint, security, generated-documentation, and test failure.
- [x] 5.4 Review the implementation diff to confirm it contains no optional Silver/Gold claims, no unverifiable private assertions, and no runtime behavior or dependency changes outside the approved scope.
