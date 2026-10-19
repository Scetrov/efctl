## ADDED Requirements

### Requirement: Complete mandatory-control attestation
The repository SHALL contain a top-level `.bestpractices.json` that proposes an answer for every mandatory OpenSSF Best Practices Passing control in the audited 43-control set. Every proposed status SHALL be `Met` or `N/A`; the manifest MUST NOT contain unknown or unmet mandatory-control statuses.

#### Scenario: All mandatory controls are represented
- **WHEN** the attestation validator reads `.bestpractices.json`
- **THEN** it finds exactly one status entry for each of the 43 audited mandatory Passing controls

#### Scenario: Incomplete status is rejected
- **WHEN** a mandatory control is missing or has status `?`, `Unknown`, or `Unmet`
- **THEN** repository validation fails and identifies the affected control

### Requirement: Evidence-backed answers
Every mandatory-control answer SHALL have a concise justification grounded in repository evidence or an explicit maintainer attestation. Controls requiring a URL SHALL include an HTTPS evidence URL, and every `N/A` answer SHALL explain why the criterion does not apply.

#### Scenario: Publicly observable control is justified
- **WHEN** a control can be verified from repository files, workflows, releases, issues, or security settings
- **THEN** its justification identifies the relevant evidence and includes an HTTPS URL when the criterion requires one

#### Scenario: N/A answer is justified
- **WHEN** a mandatory control is marked `N/A`
- **THEN** its justification states the factual condition that makes the criterion inapplicable

#### Scenario: Private vulnerability-response claim is selected
- **WHEN** `vulnerability_report_response` is added to the manifest
- **THEN** its status and justification reflect maintainer-confirmed reports and response times for the applicable six-month window

### Requirement: Human-readable release history
The repository SHALL maintain a canonical `CHANGELOG.md` with a human-readable summary for every existing SemVer release and an `Unreleased` section for future changes. Release summaries MUST NOT consist only of raw version-control logs or comparison links, and applicable publicly known runtime vulnerabilities SHALL be identified in the relevant release entry.

#### Scenario: Historical releases are covered
- **WHEN** the release-history validator compares the audited repository tags with `CHANGELOG.md`
- **THEN** every existing SemVer release has a corresponding human-readable changelog section

#### Scenario: Future user-visible change is proposed
- **WHEN** a contribution changes user-visible behavior or fixes a security issue
- **THEN** contributor policy requires an appropriate entry under `Unreleased`

#### Scenario: Release fixes a publicly known runtime vulnerability
- **WHEN** a release fixes a project vulnerability that already has a CVE or similar public identifier
- **THEN** the release notes identify that vulnerability

### Requirement: Contributor test and quality policy
Contributor-facing documentation SHALL require automated tests for new functionality and bug fixes, SHALL identify the standard test and quality commands, and SHALL require all applicable checks to pass before a change is accepted.

#### Scenario: Contributor adds functionality
- **WHEN** a contributor proposes new or changed production functionality
- **THEN** the documented contribution process requires corresponding automated tests at the appropriate unit, integration, or E2E level

#### Scenario: Contributor prepares a pull request
- **WHEN** a contributor follows the documented pull-request process
- **THEN** the documentation directs them to run the applicable formatting, warning, security, and test checks

### Requirement: Offline attestation validation
The repository SHALL provide an automated, network-independent validator for `.bestpractices.json` and its release-history invariants. The validator SHALL run through the existing standard Go test workflow.

#### Scenario: Valid attestation is tested in CI
- **WHEN** `go test ./...` runs locally or in CI
- **THEN** the attestation and changelog invariants are evaluated without external network access

#### Scenario: Required evidence is removed
- **WHEN** a mandatory status, justification, required URL, or audited changelog heading is removed or malformed
- **THEN** the standard test workflow fails with an actionable error

### Requirement: Manual badge acceptance and verification
The repository SHALL document that `.bestpractices.json` supplies proposed answers only and that an authorized project editor must review and save them for bestpractices.dev project 13754. The README SHALL display the service's live project badge rather than a hard-coded passing image or statement. The documented workflow SHALL reconcile the live project answers with the repository attestation, record unresolved discrepancies, and re-verify the live project and badge after relevant policy or attestation changes.

#### Scenario: Implementation is merged
- **WHEN** the implementation PR reaches the default branch
- **THEN** the documented process directs an authorized editor to compare project 13754 with the repository proposals, review and save accurate answers, and verify the resulting project state and live badge

#### Scenario: Badge state changes
- **WHEN** bestpractices.dev updates project 13754 from pending to passing or later changes its status
- **THEN** the README badge reflects the service's current state without a repository content change

#### Scenario: A live discrepancy is discovered
- **WHEN** the project 13754 status or justification differs from the version-controlled attestation
- **THEN** the owner workflow records the affected control and resolves it through corrected evidence, a bounded remediation, or a truthful unresolved status

#### Scenario: Relevant evidence changes
- **WHEN** the attestation, security policy, release process, or cited quality gate changes
- **THEN** the owner workflow re-verifies project 13754 and the live badge after the change merges
