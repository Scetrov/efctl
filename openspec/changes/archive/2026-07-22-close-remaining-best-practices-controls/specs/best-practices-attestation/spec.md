## MODIFIED Requirements

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
