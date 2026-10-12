## Why

PR #94 adds custom Sui support but unsupported Dockerfile layouts can lose installation instructions, LFS pointers can be mistaken for executables, and CPU warnings use the wrong architecture/evidence/output stream. Follow-up fixes must preserve official defaults and make E2E suite deadlines configurable without extending CI's budget.

## What Changes

- Validate custom inputs and a supported Dockerfile insertion point before changing any setup instructions; propagate errors and preserve files on failure.
- Normalize supported Sui installation instructions independently of comments, avoid duplicate COPY blocks, and select custom Sui unambiguously.
- Reject unhydrated LFS pointers without executing inputs or silently falling back.
- Render architecture-qualified CPU compatibility hints from collected evidence through the report writer, without new probes.
- Test the built artifact explicitly in documentation.
- Default `E2E_TIMEOUT` to `60m` locally and explicitly select `30m` in CI.

## Capabilities

### New Capabilities
- `custom-sui-setup`: transactional custom/fallback Sui Dockerfile preparation and input validation.
- `e2e-test-timeouts`: configurable local suite deadline with an explicit CI override.

### Modified Capabilities
- `container-patch-diagnostics`: mandatory Sui preparation failures abort instead of silently continuing.
- `doctor-runtime-diagnostics`: architecture-qualified compatibility hints use existing source-labelled evidence and the report writer.

## Impact

Setup patching and regression tests, doctor feature/rendering tests, Makefile and CI, and custom-binary documentation. No product dependency changes or changes to official repository defaults. Unsupported Sui layouts now fail before building rather than generating an invalid image.
