# Sui Release Pin (Default Container Runtime)

This document records how the **default** Sui runtime inside efctl environment images is selected and how to change that selection. It applies to the generated `Dockerfile` block that efctl writes into `builder-scaffold/docker/` during environment preparation.

## Current pin

| Field | Value |
| --- | --- |
| Release tag | `testnet-1.81.1` |
| Install command | `suiup install sui@testnet-1.81.1` |
| x86_64 SHA-256 (reference) | `d921910fdedf5ae2bc3f08a7236c57819a457ce53a8cc60c85f3653c7831b5d4` |
| aarch64 SHA-256 (reference) | `4c2d948417af9f55cdd1f2f5d8e558533502672004c34f6429aa6cb114a51583` |
| Verified release string | `sui 1.81.1-e04cf9aa915c` |

`testnet-1.81.1` is the latest published Sui testnet release that still serves the **fullnode JSON-RPC** endpoint. Its successor, `testnet-v1.82.1`, removed that endpoint, which broke the world-configuration scripts that call methods such as `sui_getNormalizedMoveFunction`.

## Why a tag pin, not a hash-enforced download

The project security rule asks that dependencies be pinned with a cryptographic hash rather than a tag or version. This pin deliberately deviates from that rule, and the deviation is recorded here on purpose:

- The default block installs Sui **through the suiup installer**, not by downloading a known file. Enforcing a hash at build time would require downloading the release asset directly and verifying it before extraction — a larger change with its own failure surface.
- Instead, the pin records the release **tag** (which makes the selection deterministic per efctl revision) and the **SHA-256 digests of the official release assets as reference comments**. The digests are verified against the official release at review time and re-verified on every pin update. They are a provenance record, not a build-time gate.
- If build-time hash enforcement is later required, the direct-download path is the designated follow-up. It is not part of the current pin.

## Known floating component: the suiup installer

The block still bootstraps the suiup CLI with:

```
curl -sSfL https://raw.githubusercontent.com/MystenLabs/suiup/main/install.sh | sh
```

This tracks the suiup `main` branch. It is a **separate, pre-existing** component from the pinned Sui release and is a known floating point in the supply chain. Pinning the Sui release tag does not make the suiup installer deterministic. Hardening the installer (pinning its version and digest) is out of scope for the release pin but should be tracked if the installer ever regresses.

## Custom Sui binaries are not this pin

If `builder-scaffold/docker/sui` exists and passes validation, efctl selects **that** executable and does not use the default release path. A custom binary is a user-supplied artifact: it is not release-pinned and the reference digests recorded above do not apply to it. See [Create Your Own Sui Binary](create-your-own-sui-binary.md) for the custom-binary workflow.

## Rebuilding local images

Changing the pin (or upgrading efctl across a pin change) does not migrate existing images. Rebuild the environment image to acquire the pinned release:

```
efctl env down
efctl env up
```

Existing scaffolds that still contain the legacy floating block (`suiup install sui`) migrate automatically on the next preparation: efctl recognizes the legacy block and rewrites it to the pinned block in place. The transition is one-shot and idempotent.

## Release-update checklist

When selecting a new default Sui release, do all of the following before committing the new pin. Do not reuse evidence from a previous pin to claim the replacement was tested.

1. **Confirm compatibility.** Verify the candidate release still serves the fullnode JSON-RPC endpoint and answers a required method — `sui_getNormalizedMoveFunction` for a known published function — not just an open port or a generic HTTP response.
2. **Verify the reference digests.** Download the official `x86_64` and `aarch64` release assets into `./tmp`, compute their SHA-256 digests, and record the values in the pinned block comments and this document.
3. **Run the one-shot container build.** Stage a clean builder-scaffold checkout at `./tmp/pin-validation/builder-scaffold` with no custom `docker/sui` artifact. On amd64 with Podman and network access, run the opt-in validation to build the actual generated block and assert the installed `sui` reports the exact pinned release. Disable test caching so each acceptance run builds and executes the image:

   ```
   EFCTL_PIN_VALIDATE=1 go test ./pkg/setup/ -run TestManualPinValidation -count=1 -v
   ```

   This test checks the build and version only; it does not start a fullnode or verify JSON-RPC. Separately record a successful live-node `sui_getNormalizedMoveFunction` response for a known published function (step 1), then run the full lifecycle (step 4). A passing version check alone is not compatibility acceptance.

4. **Run the full lifecycle.** Run the existing E2E lifecycle (`go test -tags e2e -run TestE2E_FullLifecycle ./tests/e2e/`) and confirm the required subtests execute and pass. Do not add a new skip pattern to hide a compatibility failure.
5. **Review security advisories.** Check for open advisories affecting the candidate release and note the pin is a compatibility pin, not a security pin.
6. **Update the managed block and tests.** Change the tag in `globalSuiBlock`, update the reference-digest comments, and update the unit/container expectations and the manual test's exact expected version so the pinned-not-floating, legacy-migration, and version checks pass.

## Rollback

Rolling back the pin is a one-token revert of the release tag in `globalSuiBlock` plus restoring the prior reference digests. If the goal is to restore floating (latest) behaviour, also remove the `legacyGlobalSuiBlock` managed-instruction recognition. No data migration is involved; images are rebuilt, not migrated.
