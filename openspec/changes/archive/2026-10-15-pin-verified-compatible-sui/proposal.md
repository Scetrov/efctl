## Why

PR #98's E2E lifecycle fails with HTTP 404 when world-configuration scripts use Sui JSON-RPC; Sui testnet-v1.82.1 removed that service after the preceding successful main run. The scaffold installs Sui through unversioned `suiup install sui` at build time, so otherwise unchanged efctl builds can acquire an incompatible runtime without any reviewed dependency change.

This change is deliberately minimal: the fix is to pin the existing suiup-managed installation to a reviewed release tag instead of resolving "latest" at build time.

## What Changes

- The generated default Sui block changes from `suiup install sui` to `suiup install sui@testnet-1.81.1`, the latest published testnet release preceding JSON-RPC removal.
- The pinned block carries the verified SHA-256 digests of the official amd64 and aarch64 release assets as reference comments, verified once at review time and re-verified on every pin update (see deviation note in design.md).
- The legacy floating block is recognised as a managed instruction, so existing scaffolds migrate in place on the next environment preparation; transitions remain idempotent.
- Custom `docker/sui` precedence, custom-input validation, transactional preparation, and unknown-layout rejection are unchanged.
- Add a pinned-sui-runtime spec, a pin maintenance document with the deviation and known-floating-suiup notes, and regression tests for the pinned block and legacy migration.
- Out of scope (removed from the earlier heavier design): direct release-asset download, build-time digest enforcement, single-member extraction, build-time version verification, TARGETARCH selection, dual-arch container builds, the seven-case container fixture matrix, and transport migration. The floating `suiup` installer itself (`curl install.sh | sh` from main) is a pre-existing accepted risk, not a new surface introduced here.

## Capabilities

### New Capabilities

- `pinned-sui-runtime`: Deterministic default Sui release selection via a pinned suiup release tag, reference digests, evidence-backed JSON-RPC compatibility acceptance, and a reviewable pin-update checklist.

### Modified Capabilities

- `custom-sui-setup`: The "custom executable absent" default path now selects the pinned release tag rather than floating suiup installation, and the managed-installation transition set gains the legacy floating block as a recognised, replaced form.

## Impact

- Primary implementation seam: `pkg/setup/custom_sui.go` (block constants and managed-instruction recognition) plus its unit, container, and validation tests.
- Provenance still flows through the official suiup installer and Sui releases; the change replaces floating tag resolution with a repository-owned tag pin. No new Go dependency or user configuration surface.
- Local images must be rebuilt to acquire the pinned release. Existing scaffolds with the floating block migrate automatically on the next environment preparation (one-shot transition, idempotent thereafter).
- Valid custom executables remain explicitly user-supplied overrides; they are not represented as release-pinned artifacts.
- This is a compatibility bridge for existing JSON-RPC clients, not a long-term substitute for transport migration.
