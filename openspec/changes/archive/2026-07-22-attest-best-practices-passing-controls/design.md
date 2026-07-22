## Context

Bestpractices.dev project 13754 currently has 11 mandatory passing controls marked Met, 31 unanswered, and `release_notes` marked Unmet. Repository inspection found evidence for most unanswered controls in the README, generated CLI documentation, contribution and security policies, Git history, GitHub Issues, tests, security workflows, release signing, and scanning configuration. The notable documentation gap is a complete human-readable release history: some historical GitHub releases contain only a generated comparison link, which does not satisfy the criterion by itself.

A top-level `.bestpractices.json` is recognized by the badge service, but its values are proposals rather than authoritative updates. An authorized project editor must review and save those proposals before the badge state changes. The artifact must therefore be both machine-consumable and independently reviewable, without overstating controls that require private maintainer knowledge.

## Goals / Non-Goals

**Goals:**

- Provide evidence-backed `Met` or justified `N/A` proposals for all 43 mandatory passing controls.
- Close the release-notes and contributor-policy documentation gaps.
- Make incomplete or malformed attestations fail the repository's normal test workflow.
- Preserve a clear manual acceptance and verification step for bestpractices.dev project 13754.
- Keep the live badge state visible after the change merges.

**Non-Goals:**

- Pursue Silver or Gold badge controls, or optional Passing `SHOULD` and `SUGGESTED` controls.
- Change efctl's CLI behavior, public interfaces, cryptographic behavior, or runtime dependencies.
- Automate authenticated writes to bestpractices.dev.
- Claim facts about private vulnerability reports without maintainer confirmation.
- Retroactively rewrite Git tags or release artifacts.

## Decisions

### Maintain a focused mandatory-control manifest

The top-level `.bestpractices.json` will contain the current 43 Passing `MUST` control statuses and their evidence, rather than copying the entire project API response. Each status will be `Met` or `N/A`; unknown and unmet values are prohibited. Justifications will point to repository files, workflows, release pages, issue history, or GitHub security features as appropriate.

Keeping the manifest focused limits drift and avoids making unaudited claims for optional or higher-tier controls. Copying the complete API response was rejected because it includes mutable service metadata and controls outside this change's scope.

### Treat attestations as claims that require evidence

Controls observable from the public repository will use direct URLs and concise explanations. Controls involving private facts will use `N/A` or `Met` only after maintainer confirmation. In particular, `vulnerability_report_response` must reflect whether reports were received during the applicable six-month window; if reports exist, their initial-response times must be checked before the answer is selected.

The implementation will distinguish repository evidence from self-attestation instead of inferring private facts from an empty public advisory list.

### Add one canonical human-readable release history

A top-level `CHANGELOG.md` will provide a human-readable entry for every existing SemVer release and an `Unreleased` section for future work. Entries will summarize user-relevant changes rather than merely reproducing commit logs or comparison links. Publicly known runtime vulnerabilities fixed by a release will be identified; where none apply, the attestation will use the criterion's allowed N/A answer with a truthful justification.

Editing every historical GitHub release was rejected as the primary solution because it is an out-of-band operation that cannot be reviewed in the implementation PR. The existing release workflow may continue generating GitHub release notes, while the changelog remains the repository-owned canonical evidence.

### Put contributor obligations in contributor-facing documentation

`CONTRIBUTING.md` will explicitly require tests for new functionality and bug fixes, list the standard unit/integration/E2E and quality commands, and require human-readable release notes for user-visible changes. Existing agent-only instructions and local hooks remain supporting evidence, but contributor-facing policy becomes authoritative.

### Validate the attestation offline through Go tests

A repository-level Go test will parse `.bestpractices.json` using the standard library and compare its status keys with a pinned set of the 43 mandatory controls audited for this change. It will reject missing or extra mandatory status keys, invalid statuses, missing paired justifications, unjustified N/A answers, and missing evidence URLs for URL-required controls. It will also verify that every repository SemVer tag represented by the audited release set has a changelog heading.

The validator will not fetch bestpractices.dev or evidence URLs during tests. Network validation was rejected because it would make normal tests flaky and would not prove authenticated acceptance on the service. The pinned criterion set intentionally turns upstream criterion changes into an explicit repository update rather than silently changing compliance claims.

### Use existing CI to enforce the invariant

The validator will run under the existing `go test ./...` CI and pre-commit workflows. Existing CodeQL, gosec, govulncheck, gitleaks, dependency review, and release-signing configuration will be cited as evidence rather than duplicated solely for the badge. CI or release configuration will only be changed if implementation reveals that a proposed `Met` answer is not enforced by the cited mechanism.

### Keep service acceptance manual and auditable

Contributor documentation will describe the post-merge sequence: an authorized editor opens project 13754, reviews the repository-proposed values, saves them, and verifies the JSON endpoint and live badge. The README badge will use the service's live badge endpoint, so it accurately reflects pending or passing state without hard-coded success text.

## Risks / Trade-offs

- **[Historical release summaries may be incomplete]** → Reconstruct entries from release pages, merged pull requests, and tagged diffs; keep statements concise and factual.
- **[A self-attested private fact may be wrong]** → Require maintainer confirmation for vulnerability-response claims and document the applicable time window.
- **[The upstream mandatory criterion set may change]** → Pin the audited set in tests so drift causes a deliberate maintenance update rather than silent failure.
- **[Evidence URLs on the moving `main` branch can change]** → Prefer stable repository paths for maintained policies and tagged release URLs for historical evidence; validate URL shape offline.
- **[A merged manifest does not itself award the badge]** → Include explicit owner acceptance and post-save verification tasks.
- **[A large changelog adds maintenance cost]** → Use a standard concise format and require entries only for user-visible or security-relevant changes.

## Migration Plan

1. Add failing attestation-validation tests that encode the audited mandatory control set and changelog expectations.
2. Add contributor and release-note documentation needed to support the proposed answers.
3. Add `.bestpractices.json` and make the validation tests pass.
4. Run repository tests and all pre-commit hooks.
5. Merge the implementation PR.
6. Have an authorized project editor review and save the proposals on bestpractices.dev.
7. Verify project 13754 reports passing status and the README badge resolves correctly.

Rollback consists of reverting the repository PR. Any answers already saved on bestpractices.dev must be reviewed separately because reverting the manifest does not automatically revert service state.

## Open Questions

- The maintainer must confirm whether any private vulnerability reports were received in the six months preceding submission and, if so, whether every initial response was within 14 days.
- During implementation, release history inspection may identify historical GitHub release descriptions worth correcting out of band, but those edits are not required for the repository PR if `CHANGELOG.md` provides complete human-readable evidence.
