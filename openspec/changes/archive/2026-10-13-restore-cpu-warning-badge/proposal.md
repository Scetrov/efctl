## Why

PR #95 retained CPU compatibility warning text but removed the shared UI warning badge, making the warning easy to miss. Existing tests covered evidence and wording but not the visual severity indicator.

## What Changes

- Restore the shared warning badge for local and runtime CPU compatibility hints while preserving supplied-writer output.
- Add regression coverage for visible severity labels with and without color, warning spacing, and compatible/unknown CPU cases.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `doctor-runtime-diagnostics`: Require prominent CPU warning badges, including a readable non-color severity label.

## Impact

`cmd/doctor_diagnostics.go` and CPU warning tests; reuse `pkg/ui` without dependency changes or new probes.
