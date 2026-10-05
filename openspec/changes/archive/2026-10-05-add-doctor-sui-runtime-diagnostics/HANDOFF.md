# Handoff: doctor runtime diagnostics and Go 1.27.1

## Historical handoff

This checkpoint predates completion and archival. See `tasks.md` and `verification.md` for final status.

### Original resume instructions

Read this file, repository/global `AGENTS.md`, the `openspec-apply-change` skill, and this change's `proposal.md`, `design.md`, both `specs/*/spec.md` files, `tasks.md`, and `verification.md` before continuing.

**Change:** `add-doctor-sui-runtime-diagnostics`
**Schema:** `spec-driven`
**Progress:** **25/27 tasks complete**
**Only remaining tasks:** **6.4** (resolve complexity and pass all hooks) and **6.5** (final validation/archive).
**No commit, push, PR or archive has been performed.** All implementation changes are in the working tree, including many untracked files. Do not reset/clean them.

Useful entry commands (run read/query commands through context-mode):

```bash
openspec status --change add-doctor-sui-runtime-diagnostics --json
openspec instructions apply --change add-doctor-sui-runtime-diagnostics --json
```

The most recent strict OpenSpec validation passed. Do not confuse valid/completed *artifacts* with completed implementation: two task checkboxes remain unchecked.

## CRITICAL: protect the host checkout and runtime

**Do not run the full Go suite or pre-commit directly against the real checkout with repository-local TMPDIR. Use the Bubblewrap harnesses below.**

Earlier in this session, legacy Git tests with temporary directories under this repository discovered its parent `.git`. They changed the real origin URL to `evefrontier/world-contracts`, fetched/pulled that repository, changed HEAD and left delete/modify conflicts. The user explicitly authorized recovery. Recovery restored the original efctl commit, canonical efctl remote and implementation files. Subsequent isolated runs confirmed host Git stayed unchanged.

Current checked Git state:

- Working branch: `main`, tracking `origin/main`.
- HEAD: `eca8c618f2b6ff5ca6b436951c7972037b9e6e20` (`chore(deps): bump the low-risk group with 4 updates (#85)`).
- `origin`: `https://github.com/Scetrov/efctl.git`.
- Exact original remote transport URL was not recorded; this canonical HTTPS URL was recovered from local efctl evidence.
- Recovery backups: `tmp/doctor-diagnostics-recovery/`, including post-failure Git metadata in `git-state-after-tests/`. **These backups predate later Go/tooling changes and are not a current source snapshot.** Do not blindly restore them.

The user suggested Bubblewrap. It is now implemented and proven locally. Host homes, checkout Git metadata, runtime sockets and container storage are hidden from the ordinary test sandbox. Only repository-local scratch/cache space is writable. A source snapshot has no inherited host `.git`; TMPDIR sits outside the snapshot source tree to prevent parent config/Git discovery.

All temporary files, tool installations, caches, logs and integration artifacts must stay under repository `./tmp`. Do not edit `builder-scaffold` or `world-contracts`. Never disable security restrictions or commit signing to get past a failure.

## What is implemented

### Doctor collection/reporting

- `pkg/doctor/diagnostic_runner.go` and tests: injectable context-aware command runner, shared 20-second collection budget, five-second command limit, 64 KiB per-stream capture, bounded/sanitized errors, signal observations, cancellation and output-limit handling. Raw stderr is not exposed in diagnostic reports.
- `pkg/doctor/runtime_diagnostics.go` and tests: source-labelled `Evidence`, CPU, server, image and runtime Sui report models; `Options.Context`/`Runner` injection. CLI context is propagated in `cmd/doctor.go`. Legacy gatherers retain their behavior and are **not** claimed to meet the new probe budget.
- `pkg/doctor/cpu_diagnostics.go` and tests: local platform queries, architecture aliases, Linux conservative per-CPU flag intersection with heterogeneity/incompleteness handling, sorted flags, macOS/Windows fixtures, distinct host version-only Sui observation. Actual unconfigured-host fixtures prove no client initialization/key/config generation.
- `pkg/doctor/runtime_metadata.go` and tests: field-selected client/server metadata, stopped/running container state and recorded exit, actual immutable image selection, explicitly labelled local configured-image fallback only after confirming container absence, digest absence, image-volume safety checks.
- `pkg/doctor/runtime_probes.go` and tests: independent managed-container version/architecture/CPU exec probes; at most one image fallback after an observed exit/removal race; isolated uniquely named local-image probe; ownership-labelled cleanup with a separate three-second allowance.
- Disposable probes override entrypoint, prohibit pulls/network/host or managed mounts/credentials, use read-only root, drop capabilities, no-new-privileges, memory/CPU/PID/core limits and disabled healthchecks/logging. Images with declared or unknown volume metadata are skipped.
- `cmd/doctor_diagnostics.go` and tests: additive default rendering, explicit sources/unavailability, actual versus fallback identity, recorded container status versus Sui probe failure, cautious possible-emulation wording, complete-report security regressions.
- Updated command help and regenerated `docs/efctl_doctor.md`; support guidance is in `docs/doctor-support.md`.

Important integration discoveries to preserve while refactoring:

1. Podman image templates require `.ID` inside `json`; the `.Id` alias does not apply there. Native Podman uses the appropriate template, and a Docker-compatible wrapper receives a bounded alternate-template attempt.
2. Podman `rm --force` may wait beyond the cleanup budget. After verifying the random ownership label, doctor sends **KILL only to its owned probe**, then force-removes it, all within the same three-second allowance. A stopped probe may reject KILL; removal must still be attempted. Never kill/remove an unverified or pre-existing object.
3. A failed version probe (including SIGILL) is not proof the container exited; independently available CPU results must remain available. Architecture aliases must not trigger mismatch warnings, and mismatch/version success must not diagnose the reported Sui startup crash.

### Test isolation and Git fixture changes

- `tools/test-isolated.sh`: disposable Bubblewrap source snapshot, default offline Go tests, shared build cache under `tmp/isolated-go-cache`, isolated pre-commit mode, explicit module-audit mode with a writable scratch cache, read-only updated-tool override.
- `tools/test-doctor-runtime-isolated.sh`: offline export of an existing immutable host image into a dedicated scratch VFS store; official `podman unshare` followed by Bubblewrap; both Podman and a Docker-compatible Podman wrapper exercised without host runtime sockets/storage.
- `pkg/doctor/runtime_integration_test.go`: gated local-engine fixture integration; refuses a normal host store. Running, stopped/tag-rebuilt, image-only, hung-version cleanup, declared-volume rejection, managed fixture sentinel and container/volume leak checks. Sui is a simulated version-only shell executable, **not upstream Sui**.
- `pkg/git/git_test.go`: replaced a network-dependent positive test with a local bare remote; negative tests have Git discovery ceilings.
- `pkg/doctor/doctor_test.go`: non-Git fixture has a test-only Git ceiling. Production repository detection was intentionally not changed, per user choice.

### Approved Go upgrade (all tasks 7.1–7.3 complete)

The user explicitly added Go 1.27.1 to this same OpenSpec change.

- `go.mod`: minimum `go 1.27.1` (was `1.26.5`). No redundant toolchain directive.
- README source-build requirement now matches. CI/release/CodeQL already use `go-version-file: go.mod`; their YAML needed no version edits.
- `toolchain_test.go`: test-first module/README/workflow alignment guards.
- Application requirements are unchanged apart from the Go directive. `go.sum` remains byte-for-byte identical to original HEAD.
- `docs/go-toolchain.md`: source-analysis compatibility, repository-local scanner reproduction, authenticated downloads and isolated hook instructions.
- `pkg/doctor/diagnostic_runner_test.go`: helper subprocesses set `GORACE` with `atexit_sleep_ms=0`. Otherwise race-instrumented helpers sleep one second at exit and spuriously exceed the fixture's 100 ms deadline. **Race detection remains enabled; production deadlines are unchanged.**

## Remaining blocker: complexity, NOT govulncheck

The latest isolated pre-commit run passed every hook **except gocyclo**. Do not weaken/skip the threshold or add blanket ignore annotations. Refactor production logic and test fixture handlers into clear small helpers while preserving existing tests and security assertions.

Current `gocyclo -over 15` results (line numbers may move):

| Complexity | Function | Location |
| ---: | --- | --- |
| 47 | `TestDisposableIsolationAndCleanup` | `pkg/doctor/runtime_probes_test.go:102` |
| 37 | `TestDoctorRuntimeIntegration` | `pkg/doctor/runtime_integration_test.go:23` |
| 32 | `TestManagedExitRaceFallsBackOnce` | `pkg/doctor/runtime_probes_test.go:28` |
| 27 | `parseLinuxCPU` | `pkg/doctor/cpu_diagnostics.go:59` |
| 22 | `TestImageMetadata` | `pkg/doctor/runtime_metadata_test.go:96` |
| 17 | `gatherImage` | `pkg/doctor/runtime_metadata.go:103` |
| 16 | `gatherRuntimeServer` | `pkg/doctor/runtime_metadata.go:37` |

Suggested decomposition: separate CPU block parsing/intersection/model selection; isolate version versus server-info parsing and image-resolution/inspection; split test command dispatchers and scenario assertions into named helpers instead of a single large closure. The integration test can separate fixture creation and per-engine/per-scenario verification. Maintain test-first coverage and rerun the live isolated probe checks after cleanup/metadata refactors.

## Scanner compatibility is resolved

The globally installed `govulncheck` was **v1.3.0 built with Go 1.26.2** and could not parse Go 1.27 language features. It was intentionally **not overwritten**.

The official proxy identified released **v1.8.0**. It was built with **Go 1.27.1** under `tmp/go127-tools/bin/govulncheck`, with official module proxy/checksum-database authentication, and **passes the isolated vulnerability hook**. Do not revert to the stale global binary when rerunning hooks.

- Scanner module checksum: `h1:clG4qBU6zH5VKjti8n5j8BBuYzoSha392xXMkXS351U=`.
- Source-processing module: `golang.org/x/tools v0.50.0`, checksum `h1:c2ifzfcuY7L90lZ2aKd8S4K2NpASF08SZx9ZuJkHmSU=`.
- Locally built binary SHA256: `42d0768bede55942485b281bbf9c03d87a918a1e45de6d209f488182db0468e8`.
- Full provenance and remaining checksums are in `verification.md`.
- If `tmp` is cleaned, reproduce the install using `docs/go-toolchain.md`; recheck released versions rather than assuming “latest” forever.

## Commands for the next session

Run commands likely to produce large output through `ctx_execute`/`ctx_batch_execute`, saving verbose logs under `./tmp` and reporting only derived results. Use read/edit for files being changed. The following are commands to run, not instructions to bypass context-mode.

### Full and race tests (safe/offline)

```bash
tools/test-isolated.sh
tools/test-isolated.sh go test -race ./... -count=1 -timeout 5m
```

### All configured hooks with the compatible scanner

```bash
EFCTL_ISOLATED_TOOLBIN="$PWD/tmp/go127-tools/bin" \
  tools/test-isolated.sh --pre-commit
```

Pre-commit mode allows networking for scanners, but not host credentials, host Git state or runtime sockets. It copies only configured cached hook environments and binds existing tools read-only. `EFCTL_ISOLATED_TOOLBIN` must resolve under repository `./tmp`. The harness currently expects cached gocyclo/gitleaks environments matching `.pre-commit-config.yaml`; do not silently change hook pins or download replacements to evade failures.

The installed Git pre-commit hook was absent when checked; the user was already prompted to run `pre-commit install`. Check again when finalizing. The configured hooks must still be run explicitly even if not installed as a Git hook.

### Module audit (authenticated online downloads, scratch only)

```bash
tools/test-isolated.sh --modules
```

Default test mode has a read-only host module cache. `tidy` can need existing test-only modules absent from that cache, so use explicit module mode rather than making the host cache writable. Module mode enables the official proxy/checksum database in a fresh scratch cache and runs download/tidy-diff/verify by default. Successful cleanup makes only its own extracted module directories writable, because Go creates them read-only.

### Runtime integration (dedicated store; no host image pulls)

```bash
tools/test-doctor-runtime-isolated.sh \
  c93bce62b190a2cc5ed3c7f87fdf9832fd37f982f2c941bf6b5b327f65374fbb
```

That base ID was an already-local Linux image with shell/sleep/uname/cat. Inspect its availability first; if absent, select an appropriate existing immutable local image, not a tag or an implicitly pulled image. The harness exports it without changing the host image, loads it into a fresh isolated store and builds a test-only Sui shell fixture. Direct rootless Podman inside Bubblewrap fails UID mapping here; the explicit outer `podman unshare` is required and was tested. The actual Podman executable here is `/usr/local/bin/podman`, not `/usr/bin/podman`; the harness resolves/binds it rather than assuming its location.

The harness also binds required runtime policy/seccomp files read-only and backs `/var/tmp` with repository-local scratch space for Podman build-cache operations. Preserve these when editing the harness.

### Validation/finalization

```bash
bash -n tools/test-isolated.sh tools/test-doctor-runtime-isolated.sh
git diff --check
openspec validate add-doctor-sui-runtime-diagnostics --strict
```

After complexity remediation and **all** hooks pass, immediately check task 6.4. Then follow the `openspec-archive-change` skill for task 6.5, including spec synchronization as required by that skill. Archive only after implementation verification, and before any final commit/PR. Do not manually improvise archive moves or mark task 6.5 complete prematurely.

No commit/PR is requested merely by writing this handoff. If subsequently requested, follow the commit skill and repository constitution: conventional commit, required model/Pi attribution, **signed commits only**. If signing fails, stop and offer to retry; never disable signing. Do not accidentally omit the many currently untracked implementation files when staging.

## Verified evidence and limits

Existing logs (all still present when this handoff was written):

- `tmp/pre-commit-go127.log`: docs/fmt/vet/build/full tests/gosec/govulncheck/gitleaks passed; gocyclo failed with the seven functions above.
- `tmp/go127-race.log`: complete race suite passed after helper exit-delay adjustment.
- `tmp/go127-module-crossbuild.log`: module verification/tidy-diff and metadata comparisons passed; Linux, macOS and Windows amd64/arm64 builds passed with `CGO_ENABLED=0`; scratch cleanup succeeded.
- `tmp/doctor-runtime-integration.log`: Podman and Docker-compatible Podman-wrapper integration scenarios passed with no surviving disposable probes, changed managed fixture sentinel, anonymous volume leaks or image pulls.

Cautions:

- No genuine Docker daemon/Desktop VM or remote service was tested.
- macOS/Windows/ARM runtime sources have fixtures and successful cross-builds, **not live platform verification**.
- No real upstream Sui binary/image was exercised; integration uses a version-only shell fixture. The reported end-user SIGILL root cause remains unknown.
- Live runtime integration passed before the application Go directive was raised; it used the already-installed Go 1.27.1 compiler. Rerun after the final source/cleanup refactor and upgraded baseline rather than treating old evidence as a fresh final integration run.
- Old failed scratch directories/logs are retained alongside current successful evidence. Some failed pipeline logs include already-passing race tests followed by an offline-cache or cleanup failure; use the dedicated successful logs above, not a misleading aggregate status.
- Clearing conversation context does not delete the working tree or `tmp`. Do not remove the scanner/evidence/recovery backups just to clear session memory.

## Optional resume prompt

> Read `openspec/changes/add-doctor-sui-runtime-diagnostics/HANDOFF.md` and resume this OpenSpec change. Refactor the seven complexity violations without weakening gates, rerun tests/hooks/runtime integration using the isolated harnesses and the repository-local compatible scanner, then archive only once verification passes. Preserve the working tree and never run repository-local temporary Git fixtures against the host checkout.
