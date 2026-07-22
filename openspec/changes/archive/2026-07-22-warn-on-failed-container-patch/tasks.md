## 1. Add Patch Diagnostic Tests

- [x] 1.1 Add warning-output test support in `pkg/setup/docker_patch_test.go` using the existing console UI facilities
- [x] 1.2 Add failing-first tests proving unmatched literal, regular-expression, and compound semantic patches emit one warning containing the patch identity and target file
- [x] 1.3 Add tests proving successful and already-applied patches emit no missing-target warning and preserve idempotent content

## 2. Implement Patch Outcome Detection

- [x] 2.1 Add a small internal mechanism in `pkg/setup/docker_patch.go` to distinguish applied, already-applied, and unmatched required patch outcomes and publish concise `ui.Warn` diagnostics
- [x] 2.2 Instrument required Dockerfile transformations with stable operation names and already-patched markers, excluding optional legacy cleanup
- [x] 2.3 Instrument required entrypoint literal, regular-expression, and compound transformations while limiting output to one warning per semantic patch attempt
- [x] 2.4 Ensure unmatched patches remain non-fatal, continue processing subsequent patches, and never include scaffold contents or environment values in warning messages

## 3. Verify Behavior and Quality Gates

- [x] 3.1 Run focused `pkg/setup` tests and confirm warning, success, and idempotency cases pass
- [x] 3.2 Run the full Go test suite and resolve regressions
- [x] 3.3 Run `pre-commit run --all-files` and resolve all formatting, linting, security, and repository quality-gate failures
