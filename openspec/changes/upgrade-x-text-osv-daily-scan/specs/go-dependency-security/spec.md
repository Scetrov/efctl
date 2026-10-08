## MODIFIED Requirements

### Requirement: Select a remediated text module
The project SHALL select `golang.org/x/text` v0.41.0 or later, and this change SHALL target v0.42.0 without introducing unrelated dependency upgrades.

#### Scenario: Resolve the dependency graph
- **WHEN** the Go module graph is resolved after the remediation
- **THEN** `golang.org/x/text` resolves to v0.42.0 and no selected version is within the GO-2026-6629 affected range

#### Scenario: Review dependency scope
- **WHEN** the remediation's module metadata changes are reviewed
- **THEN** version and checksum changes are limited to `golang.org/x/text` unless an additional change is required and explicitly justified

### Requirement: Eliminate the vulnerability finding
The project's security scans SHALL complete without reporting GO-2026-6629 (CVE-2026-56851).

#### Scenario: Run reachability-aware scanning
- **WHEN** `govulncheck ./...` is run against the remediated module graph
- **THEN** it exits successfully and does not report GO-2026-6629

#### Scenario: Run OpenSSF Scorecard
- **WHEN** OpenSSF Scorecard evaluates the merged remediation on the default branch
- **THEN** its Vulnerabilities check does not report GO-2026-6629
