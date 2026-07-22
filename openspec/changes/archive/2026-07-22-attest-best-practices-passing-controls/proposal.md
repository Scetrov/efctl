## Why

Efctl's OpenSSF Best Practices assessment currently records only 11 of 43 mandatory passing controls as met, even though the repository already contains evidence for most unanswered controls. The remaining evidence and documentation gaps must be made explicit, reviewable, and continuously validated so project 13754 can earn and retain the passing badge.

## What Changes

- Add a repository-owned `.bestpractices.json` manifest that proposes evidence-backed answers for every mandatory passing control.
- Add human-readable release notes covering existing releases and establish release-note expectations for future releases, including treatment of publicly known vulnerabilities.
- Strengthen contributor guidance so new functionality and bug fixes require corresponding automated tests and documented quality checks.
- Add automated validation that the attestation is valid JSON, covers every mandatory passing control, contains no unknown or unmet answers, and justifies N/A answers and URL-required controls.
- Document the post-merge owner review required to accept the repository proposals on bestpractices.dev and verify that project 13754 reaches passing status.
- Add the project badge to the README so its live assessment state is visible.

## Capabilities

### New Capabilities
- `best-practices-attestation`: Repository evidence, release-note policy, contributor requirements, validation, and acceptance workflow for maintaining the OpenSSF Best Practices passing badge.

### Modified Capabilities

None.

## Impact

- Adds `.bestpractices.json` and a human-readable release history such as `CHANGELOG.md`.
- Updates contributor and project documentation, including `CONTRIBUTING.md` and `README.md`.
- Adds repository-level tests or quality-gate configuration for validating the attestation artifact.
- May strengthen CI or release checks where existing practice is not sufficiently enforceable evidence.
- Requires an authorized project owner to review and save the proposed answers for bestpractices.dev project 13754 after merge; the repository file cannot award the badge by itself.
- No user-facing CLI behavior, public API, or Go runtime dependency changes are expected.
