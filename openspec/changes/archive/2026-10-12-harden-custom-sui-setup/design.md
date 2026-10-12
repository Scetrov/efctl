## Context

Custom Sui patching currently deletes recognized setup before locating WORKDIR and validates only file size. Doctor rendering re-gathers host evidence and prints globally. CI has a 35-minute job allowance and should retain a 30-minute test deadline.

## Goals / Non-Goals

**Goals:** preserve Dockerfile/entrypoint on invalid setup, reject LFS pointers, normalize supported instructions independent of comments, provide architecture-safe source-labelled hints, and centralize suite deadlines.

**Non-Goals:** authenticate arbitrary executables, execute binaries during validation, change official repository refs, update product dependencies, or diagnose SIGILL solely from flags.

## Decisions

- Build the Sui transformation in memory and validate before any write; return errors through environment preparation. Unlike warning-only continuation, this prevents building damaged setup.
- Match bounded supported Docker instructions rather than whole comment-labelled blocks. Reject unsupported/multi-stage Sui layouts rather than guess across stages; retained unrelated commands are preserved.
- Read only a short custom-file prefix; reject LFS pointers and invalid files with actionable errors. Absence selects upstream fallback; supplied invalid files do not silently select another executable.
- Consume CPUInfo architecture/flags already gathered. Only AMD64-compatible observed architectures get ADX/BMI2 hints, labelled by execution boundary, with conditional language. No extra probes or global warning writer.
- `E2E_TIMEOUT ?= 60m` is the local suite default; CI invokes the Make target with `E2E_TIMEOUT=30m`, retaining its 35-minute job cap. Per-command deadlines remain unchanged.

## Risks / Trade-offs

- Unsupported Dockerfile variants now fail → explicit errors and preservation tests; support only audited installation instruction shapes.
- Binary format/provenance remains user responsibility → explain LFS checks do not authenticate artifacts.
- Tool/scanner versions can block quality gates → use compatible verified tooling without changing product dependencies; never bypass signing/hooks.

## Migration Plan

No configuration migration. Existing official scaffolds and valid local custom files continue working. Unsupported layouts/invalid files receive preparation errors. Rollback is reverting the follow-up commit.

## Open Questions

None.
