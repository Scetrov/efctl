# OpenSSF Best Practices Reconciliation

This record compares the version-controlled [`.bestpractices.json`](../.bestpractices.json) proposal with [bestpractices.dev project 13754](https://www.bestpractices.dev/projects/13754). It contains only public repository evidence and service-visible results; do not add credentials, private reports, or private maintainer communications.

## 2026-07-22 audit

- **Attestation merge:** PR [#64](https://github.com/Scetrov/efctl/pull/64), merged into `main` as `a0ccdaa`.
- **Assessment:** project 13754 reported `in progress`, **67%**; the [live badge](https://www.bestpractices.dev/projects/13754/badge) matched that state.
- **Scope:** all 43 Passing controls in `bestpractices_test.go` were compared with the live project JSON endpoint and the current Passing criteria.
- **Result:** 31 controls matched exactly. Eleven controls had matching statuses but service-generated justifications different from the repository proposal. `release_notes` was the only status mismatch. The service also reported the derived `achieve_passing` and `achieve_silver` controls as `Unmet`; neither is a repository claim or a target of this change.

| Control(s) | Repository proposal | Live result | Classification | Required follow-up |
| --- | --- | --- | --- | --- |
| `contribution`, `floss_license`, `license_location`, `documentation_basics`, `sites_https`, `discussion`, `repo_public`, `repo_track`, `report_process`, `build`, `delivery_mitm` | `Met`, with repository-specific public evidence | `Met`, with service-generated justification | Service-side discrepancy | An authorized editor must review the public evidence and save only the accurate repository proposals. If the service retains a different accurate justification, record that outcome in the next audit rather than changing the repository claim speculatively. |
| `release_notes` | `Met`; [`CHANGELOG.md`](../CHANGELOG.md) supplies human-readable release summaries | `Unmet`; "No release notes file found." | Pending owner acceptance | An authorized editor must verify the changelog evidence and save `Met` only if the service accepts it. If it does not, keep the live result unresolved and open a bounded follow-up for the service's required evidence format. |
| `achieve_passing`, `achieve_silver` | Not proposed | `Unmet` | Service-side derived result | Recheck after the accurate Passing proposals are saved. Do not add Silver claims or change runtime behavior to affect this result. |

An authorized maintainer confirmed the private-report fact supporting the existing `vulnerability_report_response: N/A` proposal. No report details or maintainer identity are recorded.

## 2026-07-22 expanded Passing-control proposal

A subsequent editor review identified 21 additional unanswered score-relevant controls. The repository now proposes the following evidence-backed values; each is classified as **pending owner acceptance** until an authorized editor reviews and saves it.

| Controls | Proposed value | Public evidence |
| --- | --- | --- |
| `contribution_requirements`, `english`, `version_semver`, `version_tags`, `report_tracker`, `enhancement_responses` | `Met` | `CONTRIBUTING.md`, `README.md`, `CHANGELOG.md`, Git tags, and the public issue tracker |
| `build_floss_tools`, `test_invocation`, `test_most`, `test_continuous_integration`, `tests_documented_added`, `warnings_strict` | `Met` | `CONTRIBUTING.md`, the Go test suite, Make targets, pre-commit, and GitHub Actions |
| `vulnerabilities_critical_fixed`, `static_analysis_common_vulnerabilities`, `static_analysis_often` | `Met` | `SECURITY.md`, CodeQL, gosec, govulncheck, go vet, CI, and pre-commit |
| `crypto_call`, `crypto_weaknesses`, `crypto_pfs`, `dynamic_analysis`, `dynamic_analysis_unsafe`, `dynamic_analysis_enable_assertions` | `N/A` | efctl does not implement the cryptographic protocols or dynamic-analysis tooling to which these controls apply |

No repository-controlled enforcement gap was identified by this audit. The remaining score gap is pending authorized-editor review and acceptance; the expanded manifest proposals have not yet been saved to the service.
