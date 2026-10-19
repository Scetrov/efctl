## ADDED Requirements

### Requirement: Live score reconciliation
The project SHALL maintain a reviewable reconciliation of bestpractices.dev project 13754 against the version-controlled `.bestpractices.json` attestation and the current Passing criteria. The reconciliation MUST identify every status mismatch and classify it as pending owner acceptance, unsupported repository evidence, or a service-side discrepancy.

#### Scenario: Live project differs from the repository proposal
- **WHEN** an owner audits project 13754 and finds a control status or justification that differs from `.bestpractices.json`
- **THEN** the reconciliation records the control, both values, its classification, and the required follow-up

#### Scenario: Repository evidence cannot support a live answer
- **WHEN** a live answer cannot be justified by public evidence or a confirmed maintainer attestation
- **THEN** the owner workflow does not save it as `Met` and records a remediation or correction

### Requirement: Authorized owner acceptance and verification
The project SHALL document an authorized-editor procedure for reviewing and saving accurate repository proposals in project 13754. The procedure MUST verify the resulting project state and live badge without storing credentials or automating authenticated service writes.

#### Scenario: Accurate proposals are ready for acceptance
- **WHEN** the repository manifest has merged and the reconciliation finds only pending owner acceptance
- **THEN** an authorized editor reviews and saves the proposals in project 13754 and records the resulting score and badge state

#### Scenario: Service update does not propagate
- **WHEN** the saved project state or badge does not reflect the reviewed answers after the documented propagation interval
- **THEN** the workflow records the discrepancy and directs the owner to inspect or correct the service-side project state

### Requirement: Score-gap remediation lifecycle
Every repository-controlled control that prevents the intended Passing score SHALL have a bounded remediation that corrects evidence, documentation, or enforcement and is validated through the standard repository workflow. Controls requiring unavailable or private facts SHALL remain explicitly unresolved rather than receiving an unsupported claim.

#### Scenario: Repository-controlled gap is identified
- **WHEN** reconciliation identifies a control whose required evidence or enforcement is absent from the repository
- **THEN** the follow-up defines the narrow remediation and its offline validation before the control is proposed as `Met`

#### Scenario: Private fact remains unavailable
- **WHEN** a control depends on a fact the maintainer cannot confirm
- **THEN** the attestation retains a truthful `N/A` status where allowed or records the control as unresolved

### Requirement: Periodic live-score re-verification
The project SHALL require re-verification of project 13754 after an attestation, security-policy, release-process, or relevant criteria change. A live-score regression MUST produce a documented reconciliation and follow-up.

#### Scenario: A relevant policy changes
- **WHEN** a change modifies `.bestpractices.json`, the security policy, release process, or a cited quality gate
- **THEN** the owner workflow rechecks the live project and badge after the change merges

#### Scenario: The live score regresses
- **WHEN** project 13754 no longer reports the expected Passing score
- **THEN** the project records the affected controls and opens a remediation follow-up without changing repository claims speculatively
