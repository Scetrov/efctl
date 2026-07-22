## Context

Project 13754 currently reports a 67% OpenSSF Best Practices score. The preceding attestation change supplies a repository-owned manifest for all 43 Passing MUST controls, offline validation, release history, and an owner workflow. The service treats that manifest as a proposal: an authorized editor must review and save the values before the live score changes. The remaining gap may therefore be service-state drift, a rejected proposal, or a real missing repository control.

## Goals / Non-Goals

**Goals:**

- Reconcile every score-relevant live answer with the repository attestation and retain an auditable record of the comparison.
- Provide a deterministic owner procedure for accepting truthful proposals and verifying the resulting score and badge.
- Turn each repository-controlled discrepancy into a bounded evidence, documentation, or quality-gate follow-up with offline validation.
- Preserve a clear rollback and recurring verification process.

**Non-Goals:**

- Assert that a control is met before an authorized owner confirms the underlying facts.
- Automate authenticated writes to bestpractices.dev or store editor credentials.
- Pursue Silver or Gold criteria, alter efctl runtime behavior, or add dependencies without a separately justified discrepancy.

## Decisions

### Treat the live project as the reconciliation source of truth

Implementation will fetch or export the project 13754 answers for review, compare them to `.bestpractices.json`, and classify differences as pending acceptance, inaccurate repository evidence, or a service-side discrepancy. The repository manifest remains the version-controlled proposal; copying mutable live project metadata into it is rejected because it would obscure reviewable evidence.

### Keep privileged service actions manual and record their outcome

An authorized owner will review and save only the evidence-backed proposals through the service UI. Documentation will capture the date, editor role, project URL, score, badge result, and unresolved controls without recording credentials or private report details. Browser automation is rejected because authenticated service actions require an accountable human decision.

### Close gaps with evidence first, enforcement second

For each unresolved repository-controlled criterion, implementation will first correct the status or add precise public evidence. A new quality gate or code change will be introduced only when the criterion requires enforceable behavior that the current repository cannot demonstrate. This avoids speculative scope expansion while keeping every accepted claim supportable.

### Make regression detection explicit

Offline tests will continue to validate manifest structure and repository-controlled evidence. Documentation will require re-checking the live project after changes to the manifest, security policy, release process, or badge service criteria. A live score regression creates a tracked follow-up rather than silently changing a claim.

## Risks / Trade-offs

- **[The 67% score is stale or proposals are not imported]** → Record the live audit, save the reviewed proposals, and verify the project endpoint and badge after propagation.
- **[A control requires a private fact]** → Use `N/A` or retain an unresolved follow-up until an authorized maintainer provides a truthful attestation.
- **[A service criterion changes]** → Compare the current criterion metadata during each reconciliation and update the pinned offline audit deliberately.
- **[Manual acceptance is not repeatable]** → Document the exact owner procedure and outcome record, including rollback steps.
- **[Score cannot reach 100% truthfully]** → Document the blocking control and implement only the narrowly scoped remediation it requires.

## Migration Plan

1. Merge the existing attestation change and verify the repository manifest is available from the default branch.
2. Capture the live project answers and compare them with the manifest and current Passing criteria.
3. Have an authorized editor review and save the accurate proposals, then record the resulting score and badge state.
4. Implement and validate any repository-controlled remediation identified by the audit.
5. Re-verify the live project after service propagation; if needed, correct saved answers or revert repository claims and create a follow-up.

Rollback consists of reverting any repository remediation and correcting any saved project answer in bestpractices.dev. Repository rollback alone does not undo a service-side save.

## Open Questions

- Which exact controls account for the currently observed 67% live score after the attestation PR is merged and proposals are imported?
- Does the service expose all necessary reconciliation data through an unauthenticated project endpoint, or must the owner export some fields from the editor UI?
- What propagation delay should the owner workflow allow before classifying a score update as failed?
