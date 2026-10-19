## Context

`pkg/setup/docker_patch.go` adapts upstream Dockerfile and `entrypoint.sh` content with literal and regular-expression replacements during environment setup. Several transformations are intentionally idempotent, but a replacement currently returns the original string when its expected source text is absent. That behavior is indistinguishable from an already-applied patch and can silently leave a partially compatible scaffold in place after upstream text changes.

The setup flow already uses `pkg/ui` for operator-facing console messages, including a standard `ui.Warn` printer. Patch diagnostics must remain non-fatal because the current patch flow writes the best available content and subsequent build or runtime validation may still succeed.

## Goals / Non-Goals

**Goals:**
- Distinguish successful, already-applied, and unmatched patch outcomes.
- Emit a standard console warning for each intended semantic patch that is unmatched.
- Include enough context in each warning to identify the affected file and patch operation.
- Preserve current patch output and idempotent re-application behavior.
- Cover the outcome states with focused unit tests and retain integration coverage for the complete Docker preparation flow.

**Non-Goals:**
- Replace the current string-based patching strategy with an AST, template engine, or versioned upstream files.
- Make unmatched patches fatal or change `prepareDockerEnvironment`'s public error behavior.
- Warn for optional cleanup operations whose source text is legitimately absent.
- Change Docker or Podman runtime behavior, scaffold contents, or configuration.

## Decisions

### Track semantic patch outcomes explicitly

Each required transformation will check for an already-patched marker before attempting its source match. If the marker is present, it will remain a quiet no-op. If a supported source form matches, the transformation will apply normally. Only when neither condition holds will it emit a warning.

This semantic three-state check is preferred over comparing the complete before/after strings because some patch helpers perform multiple replacements, some already-patched scripts are intentionally unchanged, and unrelated normalization could otherwise hide a missed target.

### Warn once per intended patch operation

Diagnostics will identify a stable patch name and the relative target (`Dockerfile` or `scripts/entrypoint.sh`). Helpers that use multiple raw replacements to implement one outcome will report at most one warning for that outcome. This keeps warnings actionable and avoids exposing large expected/replacement strings or producing duplicate messages.

A shared small helper may be used for simple literal replacements, while regex and compound transformations can perform the same state checks directly. This is preferred over forcing all transformations through one generic abstraction that would obscure their different match semantics.

### Use the standard console warning channel

Unmatched patches will use `ui.Warn` rather than the standard-library logger. This ensures the diagnostic is visible in normal `efctl env up` console output and follows existing CLI warning formatting. Warning messages will not include scaffold content, environment values, or other potentially sensitive data.

### Preserve non-fatal setup behavior

An unmatched replacement will leave the relevant content unchanged, emit its warning, and allow the remaining patches and file write to continue. This preserves compatibility with the existing best-effort patch process while making fragility observable. Making misses fatal was considered but rejected because the requested behavior is diagnostic and some downstream paths may remain functional.

### Test the three outcome states

Tests will capture warning output and assert that a missing target produces a warning containing the patch identity and file. Separate cases will verify that a successful replacement and an already-patched input remain warning-free. Existing transformation and idempotency assertions will continue to validate resulting content.

## Risks / Trade-offs

- **[Risk] Incomplete already-patched markers could create false warnings** → Use stable markers from the produced replacement and add explicit idempotency tests for every instrumented transformation.
- **[Risk] Broad instrumentation could warn for optional legacy cleanup** → Limit diagnostics to required forward patches; absent deprecated cleanup text remains an expected no-op.
- **[Risk] Multiple misses could create noisy setup output** → Emit one concise warning per semantic patch, with a stable operation name and target file.
- **[Risk] Warnings allow setup to continue with incompatible content** → Keep messages actionable and preserve downstream build/runtime failures; changing failure policy remains a separate decision.
