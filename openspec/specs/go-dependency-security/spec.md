## ADDED Requirements

### Requirement: Select a remediated text module
The project SHALL select `golang.org/x/text` v0.39.0 or later, and the remediation change SHALL target v0.40.0 without introducing unrelated dependency upgrades.

#### Scenario: Resolve the dependency graph
- **WHEN** the Go module graph is resolved after the remediation
- **THEN** `golang.org/x/text` resolves to v0.40.0 and no selected version is within the GO-2026-5970 affected range

#### Scenario: Review dependency scope
- **WHEN** the remediation's module metadata changes are reviewed
- **THEN** version and checksum changes are limited to `golang.org/x/text` unless an additional change is required and explicitly justified

### Requirement: Eliminate the vulnerability finding
The project's security scans SHALL complete without reporting GO-2026-5970.

#### Scenario: Run reachability-aware scanning
- **WHEN** `govulncheck ./...` is run against the remediated module graph
- **THEN** it exits successfully and does not report GO-2026-5970

#### Scenario: Run OpenSSF Scorecard
- **WHEN** OpenSSF Scorecard evaluates the merged remediation on the default branch
- **THEN** its Vulnerabilities check does not report GO-2026-5970

### Requirement: Preserve build integrity and behavior
The dependency remediation SHALL preserve efctl's existing build, test, and command behavior and SHALL retain verifiable Go module checksums.

#### Scenario: Verify module integrity
- **WHEN** `go mod verify` is run after the dependency update
- **THEN** all downloaded modules are verified successfully

#### Scenario: Execute project quality gates
- **WHEN** the repository's required tests and pre-commit hooks run with the remediated dependency
- **THEN** they complete successfully without changes to efctl's application behavior
