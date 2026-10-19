## Why

The live OpenSSF Best Practices project still reports a 67% score. The repository now contains a complete, tested 43-control attestation, but repository proposals do not affect bestpractices.dev until an authorized editor reviews and saves them. The remaining score gap must be reconciled against the live project state and closed without making unsupported claims.

## What Changes

- Audit the live project 13754 answers against the merged repository attestation and identify every control preventing a 100% Passing score.
- Add an auditable owner-run acceptance and verification procedure that applies repository proposals, records the resulting service state, and captures follow-up work for any control that cannot truthfully be marked `Met` or `N/A`.
- Strengthen offline validation and evidence only where the live reconciliation demonstrates a repository-controlled gap.
- Document an explicit rollback and periodic re-verification process so future score regressions are detected and handled.

## Capabilities

### New Capabilities

- `best-practices-score-completion`: Reconciles the repository attestation with the live bestpractices.dev assessment and governs the owner acceptance, verification, follow-up, and rollback workflow required to achieve and retain the full Passing score.

### Modified Capabilities

- `best-practices-attestation`: Extend the manual badge-acceptance requirement with a verifiable live-score reconciliation and follow-up policy for unresolved controls.

## Impact

- Affects `.bestpractices.json`, attestation tests, contributor/owner documentation, and bestpractices.dev project 13754 administration.
- May add evidence or quality gates only after the live-control audit identifies a specific repository-owned gap.
- Does not change efctl runtime behavior, public APIs, or dependencies unless a separately justified quality gap requires it.
