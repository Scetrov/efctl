## Context

The scaffold installs Sui with `suiup install sui`, which resolves the **latest** testnet release at image build time. Sui testnet-v1.82.1 removed the fullnode JSON-RPC service that efctl's world-configuration scripts call (including `sui_getNormalizedMoveFunction`), so an otherwise-unchanged efctl build acquired an incompatible runtime with no reviewed dependency change.

The previous iteration of this change proposed a heavy-weight "verified" runtime: a direct HTTPS download of a fixed release asset, build-time SHA-256 enforcement, single-member archive extraction, build-time version verification, TARGETARCH-aware selection, and a seven-case container fixture matrix. Under review, that was judged over-engineered for the actual defect. The defect is **floating release resolution**, not **unverified bytes**. Pinning the release tag addresses the defect directly.

## Goals / Non-Goals

**Goals**

- Make the default Sui release deterministic per efctl revision: pin a reviewed release tag in the generated block.
- Migrate existing scaffolds that still contain the floating block, in place, on the next environment preparation.
- Keep the change small, reviewable, and inside the existing transactional preparation seam.
- Record enough provenance for a future maintainer to re-verify the pin and to change it deliberately.

**Non-Goals**

- Downloading release assets directly or enforcing digests at build time.
- Build-time version verification, single-member extraction, or TARGETARCH-based architecture selection.
- Dual-architecture container builds or a container fixture matrix beyond the existing harness.
- Migrating world scripts off JSON-RPC, broadening E2E skip patterns, or hardening unrelated container dependencies.
- Changing the floating `suiup` installer itself (see Decisions 3).

## Decisions

### 1. Pin the release tag through the existing suiup-managed path

The default block changes only the resolution step:

```
suiup install sui
```

becomes

```
suiup install sui@testnet-1.81.1
```

Everything else about the block — the suiup installer bootstrap, the `PATH` line, the relocation of the installed binary to `/usr/local/bin/sui` — is unchanged. This keeps the surface of change to a single token plus provenance comments, and reuses the existing managed-instruction recognition, idempotence, and transition logic already covered by the existing test suite.

`testnet-1.81.1` is the latest published testnet release that still serves the fullnode JSON-RPC endpoint. It is a compatibility pin, not a security pin; the goal is determinism and known-good behaviour, not the newest or the most patched release.

### 2. Record reference digests as comments, not enforced at build time

The pinned block carries the SHA-256 digests of the official `amd64` and `aarch64` release assets as reference comments:

```
# Reference SHA-256 (not verified at build time):
#   x86_64  d921910fdedf5ae2bc3f08a7236c57819a457ce53a8cc60c85f3653c7831b5d4
#   aarch64 4c2d948417af9f55cdd1f2f5d8e558533502672004c34f6429aa6cb114a51583
```

These are a **non-enforced integrity reference**: a maintainer updating the pin re-verifies them against the official release before committing the new tag, and the recorded values let a reviewer check that the committed pin matches the published asset. They are deliberately not enforced at build time, because the block installs through suiup rather than downloading a known file; build-time enforcement would require the direct-download path this change drops.

**Deviation from the repo's cryptographic-hash rule:** the project's security rule asks that dependencies be "pinned with a cryptographic hash, not a tag or version." This change pins a release tag and records the hash as a reference rather than enforcing it. That is a conscious, documented deviation: it matches the actual transport (suiup), keeps the change minimal, and the enforcement gap is explicitly acknowledged in the pin maintenance document. If the maintainer later wants enforced digests, the direct-download path is the follow-up, not part of this change.

### 3. Leave the floating suiup installer as-is

The block still installs the suiup CLI with `curl -sSfL .../suiup/main/install.sh | sh`, which tracks the `main` branch. This is a pre-existing accepted risk and a separate component from the Sui release this change pins. Hardening the installer is out of scope; the pin maintenance document records it as a known floating component so it is not mistaken for a regression.

### 4. Recognise the legacy floating block as a managed, replaced form

The pre-pin floating block is kept as `legacyGlobalSuiBlock` and added to `knownSuiInstructions()` so preparation can find it in an existing scaffold and replace it with the pinned block. This makes the migration a one-shot, idempotent transition: the first `env up` after upgrading efctl rewrites the block, and subsequent preparations are no-ops. No separate migration tooling is introduced.

### 5. Verify integrity and compatibility at separate test layers

- **Reference digests** are verified once at review time against the official release assets (recorded in the block and this document), and re-verified on every pin update.
- **Compatibility** is verified by a controlled one-shot container build of the actual generated block: it must install `sui`, report the pinned release, and answer a required JSON-RPC method (`sui_getNormalizedMoveFunction`) against a known published function. A red signal (the 404 / method-unavailable from 1.82.1) and the corresponding candidate result are captured with actual runtime versions recorded.
- Unit tests cover the pinned block constant, the legacy-block migration, idempotence, custom precedence, and unknown-layout rejection in the existing suites.

## Risks / Trade-offs

- **Reference-only digests are not build-time protection.** A tampered or mis-served release asset would not be caught at build time. This is the accepted cost of keeping the suiup transport; the pin maintenance checklist requires re-verification at review time, and the direct-download path is the designated follow-up if stronger enforcement is needed.
- **The suiup installer is still floating.** A future suiup `main` regression could break the block even with a pinned Sui tag. Recorded as a known component; out of scope here.
- **The pin is a compatibility pin, not a security pin.** `1.81.1` may lag security fixes present in later releases until JSON-RPC support returns or the world scripts migrate transports. Acceptable for the scope of this change.

## Migration Plan

1. Ship the pinned block and legacy-block recognition.
2. Existing scaffolds migrate on the next `env up` / environment preparation (one-shot, idempotent). No user action or manual Dockerfile edit required.
3. Local images are rebuilt to acquire the pinned release. No migration of existing images is performed; users rebuild.
4. Rollback is a one-token revert of the tag pin plus removal of the legacy-block constant if the floating behaviour is restored.

## Open Questions

- None blocking. The only open follow-up, deliberately out of scope, is whether to later add build-time digest enforcement via a direct-download path, and whether to pin the suiup installer release.
