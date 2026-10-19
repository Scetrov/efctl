## Context

CPU warnings are currently plain report rows. The shared `ui.Warn` printer provides a yellow `WARNING` badge and spacing, but direct printing would bypass the supplied report writer.

## Goals / Non-Goals

**Goals:** Restore a prominent, color-independent badge for both CPU boundaries; retain conditional wording, evidence gating, and writer capture.

**Non-Goals:** Change warning eligibility, probe collection, or global UI styling.

## Decisions

Format warning content with `ui.Warn.Sprint` and send the resulting string through the existing report writer. This reuses shared styling and spacing without direct stdout output. Keep local/runtime row labels and existing cautious guidance. Plain prefixes alone would not restore the shared visual treatment; direct `Println` would break output capture.

Test both color modes and both boundaries using fixed collected evidence. Check the shared readable badge, conditional guidance, surrounding spacing, and absence for compatible/unknown/non-x86 CPUs; preserve existing no-probe tests.

## Risks / Trade-offs

- Global color state can leak between tests → restore the prior state and do not parallelize color-changing tests.
- Shared UI prefix changes could weaken assertions → explicitly assert the readable `WARNING` label rather than deriving the expectation from the implementation.

## Migration Plan

No data or dependency migration. Revert the rendering change to roll back.

## Open Questions

None.
