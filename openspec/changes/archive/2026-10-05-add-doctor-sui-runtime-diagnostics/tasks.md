## 1. Diagnostic execution and report model

- [x] 1.1 Add test-first fixtures for a context-aware diagnostic runner covering success, missing executables, malformed output, nonzero exit, observed signals, timeout, cancellation, output limits, secret redaction, and terminal control characters.
- [x] 1.2 Implement injectable bounded command capture with a shared 20-second new-probe budget, five-second per-command deadline, 64 KiB per-stream capture limits, and 512-character sanitized error excerpts.
- [x] 1.3 Extend doctor report types with source-labelled CPU, runtime-server, image, and Sui executable results, including explicit availability reasons; propagate command context without breaking existing gatherers.

## 2. Local CPU and host Sui evidence

- [x] 2.1 Add parser tests for Linux x86/ARM CPU sources, heterogeneous and incomplete per-CPU flags, macOS/Windows fixtures, architecture aliases, and unavailable platform sources.
- [x] 2.2 Implement standard-library/read-only-command CPU and machine-architecture gatherers with deterministic feature sets, conservative per-CPU intersection, and explicit source/availability labels.
- [x] 2.3 Add host Sui version-only tests and gathering, distinguishing missing binary from failed version command and proving that no client configuration, keys, or recovery phrases are generated when config is absent.

## 3. Runtime server and actual image metadata

- [x] 3.1 Add Docker, Podman, and Docker-compatible Podman-wrapper fixtures for client/server version and server OS/architecture, including unavailable server and partial metadata cases.
- [x] 3.2 Implement bounded, field-selected runtime metadata gathering with separate client/server identity and only evidence-supported remote/VM indicators.
- [x] 3.3 Add inspect tests for running/stopped/missing managed Sui containers, container/image identity mismatch after tag rebuild, local-image-only fallback, absent digests, and missing images.
- [x] 3.4 Implement inspect-based state/exit/image reporting using the managed container's immutable image ID when present, with explicitly labelled configured-image fallback and no image pulls.

## 4. Runtime executable and CPU probing

- [x] 4.1 Add exec probe tests covering independent Sui version/CPU/architecture results, host Sui absence, version command SIGILL, and the managed container exiting during collection.
- [x] 4.2 Implement running-container version-only and CPU/architecture probes with source labels and at most one isolated image fallback after an exec race.
- [x] 4.3 Add sandbox command/cleanup tests proving entrypoint override, no pulls, no network, read-only root, capability dropping, no-new-privileges, resource/PID/core-dump limits, no host/managed mounts, no credentials, and safe handling of image-declared volumes.
- [x] 4.4 Implement uniquely named disposable probes of existing immutable local images, skipping unsupported isolation rather than weakening it and preserving independently collected results when the Sui binary fails.
- [x] 4.5 Implement cleanup for completion, failure, timeout, and cancellation with a separate maximum three-second allowance, including safe reporting of cleanup failure and proof that pre-existing containers are never removed.

## 5. Rendering, interpretation, and documentation

- [x] 5.1 Add report-rendering tests covering source distinctions, explicit unavailable reasons, deterministic flags, actual versus fallback image identity, normalized architecture comparisons, and cautious possible-emulation warnings.
- [x] 5.2 Render the new fields in default `efctl doctor` while retaining existing sections and exit behavior; distinguish container exit codes from observed Sui probe failures and avoid unsupported CPU/crash diagnoses.
- [x] 5.3 Update command help and regenerate `docs/efctl_doctor.md`; document transient probe isolation/cleanup, remote/VM source boundaries, unavailable fields, and the update-then-doctor support workflow.
- [x] 5.4 Add complete-report security regression tests preserving `sui-config-guard` behavior and preventing credentials or unsafe terminal controls from new metadata/error paths.

## 6. Verification and release preparation

- [x] 6.1 Run targeted unit/rendering tests and the full Go test suite; keep all fixtures and temporary integration artifacts under repository `./tmp`.
- [x] 6.2 Run focused local container integration checks with available Docker/Podman engines for running, stopped, image-only, and timed-out probes; verify no image pulls, no managed data changes, no anonymous volume leaks, and no surviving disposable containers.
- [x] 6.3 Review representative successful and partial reports for support usefulness; document untested platform/engine combinations without presenting them as verified.
- [x] 6.4 Run repository pre-commit hooks and resolve all failures before finalizing implementation; request `pre-commit install` if configured hooks are not installed.
- [x] 6.5 Validate and archive the completed OpenSpec change after implementation verification and before any final signed conventional commit/PR, including required model and Pi attribution.

## 7. Go 1.27.1 baseline (approved scope extension; complete before archive)

- [x] 7.1 Add regression tests for the Go baseline/documentation/workflow alignment, then raise `go.mod` and source-build documentation to Go 1.27.1 while retaining setup-go's module-file selection.
- [x] 7.2 Verify and build a compatible released `govulncheck` with checksum-authenticated modules under `./tmp`, support read-only updated tooling in isolated hook runs, and record scanner version/checksums without changing application dependencies.
- [x] 7.3 Verify Go 1.27.1 with isolated full/race tests, module verification/tidy-diff checks and Linux/macOS/Windows cross-builds; record results, unchanged application dependency metadata and live-platform limitations.
