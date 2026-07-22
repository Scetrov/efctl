## Why

Container setup patches depend on exact upstream Dockerfile and entrypoint text, so an upstream format change can turn a replacement into a silent no-op. Operators need an explicit warning when an intended patch cannot find its target so they can diagnose incomplete environment setup before it causes less obvious runtime failures.

## What Changes

- Detect when an intended Dockerfile or entrypoint replacement finds neither its expected source text nor an already-patched equivalent.
- Emit an actionable console warning identifying the skipped patch and affected file.
- Preserve idempotent behavior: content that is already patched does not produce a warning.
- Add focused tests for successful replacement, already-patched input, and missing replacement targets.

## Capabilities

### New Capabilities
- `container-patch-diagnostics`: Defines observable warnings when Docker environment patch targets cannot be found, while preserving quiet idempotent re-application.

### Modified Capabilities

None.

## Impact

- Affects Docker environment preparation in `pkg/setup/docker_patch.go` and its tests in `pkg/setup/docker_patch_test.go`.
- Adds warning output to the existing `efctl env up` setup flow when upstream scaffold content is incompatible with an expected patch.
- Uses the existing console UI logging facilities; no API, configuration, dependency, or persisted-data changes are expected.
