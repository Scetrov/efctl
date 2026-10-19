## Context

`pkg/doctor/doctor.go` gathers report sections sequentially. `gatherSystem()` reports `runtime.GOOS/GOARCH`, OS name, and WSL detection; that platform is not necessarily the physical CPU or container server platform. The runtime gatherer uses the CLI's `--version`, not server metadata. The environment gatherer counts running containers and only includes logs for running containers. Host Sui client configuration has a separate existence guard. `cmd/doctor.go` renders human-readable text.

The reported SIGILL occurs inside Sui, after successful key creation and PostgreSQL readiness. A successful `sui --version` does not establish that all node-start code paths are CPU-compatible. Upstream installs Sui without selecting a fixed version, making the local image's actual binary and identity important evidence. The affected user's root cause remains unknown.

Existing container APIs include context-aware inspect and execution internally, but some convenience methods hide errors. New doctor probes need structured results, command deadlines, and captured rather than streamed output. Docker and Podman, including remote servers and Docker-compatible Podman wrappers, must remain distinguishable.

## Goals / Non-Goals

**Goals:**
- Make plain `efctl doctor` collect the evidence needed for Sui startup triage, including after the managed Sui container exits.
- Separate efctl build platform, local machine CPU view, runtime server platform, image platform, and execution-time CPU view.
- Preserve the existing report and client-config behavior; make missing evidence explicit rather than fatal.
- Keep new diagnostics bounded, credential-safe, and independent of internet access.
- Align efctl's minimum Go version, CI/release toolchain selection and source-build guidance on Go 1.27.1, with compatible security scanning.

**Non-Goals:**
- Fix the reported SIGILL, select a Sui release, pin upstream installations, or change `env up`.
- Assert a Sui AVX/AVX2 requirement, prove that a VM masks flags, or infer Rosetta/QEMU solely from an architecture mismatch.
- Run `sui start`, client initialization, genesis, funding, database operations, or core-dump analysis.
- Introduce JSON report output, telemetry, dependencies, or a general runtime abstraction rewrite.

## Decisions

### 1. Add source-labelled data rather than one ambiguous architecture field

Extend report types with CPU information, server information, and a separate Sui runtime diagnostic section. Fields carry availability/error reasons and provenance; unavailable flags differ from a known empty feature set. Normalize common aliases (`x86_64`/`amd64`, `aarch64`/`arm64`) for comparisons while retaining reported values where useful.

Use standard read-only OS sources: `/proc/cpuinfo` and machine architecture on Linux; selected `sysctl`/`uname` values on macOS; bounded, fixed PowerShell/CIM queries on Windows. Unsupported sources produce unavailable reasons, not guessed data. Sort/deduplicate instruction flags and, on heterogeneous Linux CPUs, use the intersection of reported per-CPU flags when complete; do not claim uniform support from one core. Clearly label local CPU data as the view visible to efctl, which may itself run inside WSL or a VM.

Alternative: report only efctl's Go architecture and selected AVX booleans. Rejected because it hides host/server boundaries and fails to support ARM investigations.

### 2. Inspect server and actual image metadata with explicit field selection

Query structured Docker/Podman version and info responses through the selected engine. Report client and server versions separately, server OS/architecture, and virtualization/remote indicators only when evidence supports them. Do not dump engine info, environment variables, registry authentication, or connection URLs.

Inspect `sui-playground` even when stopped: state, recorded exit code, image reference and immutable image ID. Inspect that image ID for OS/architecture and repository digests. Local images may have no repository digest; display `not recorded`, not a fabricated checksum. If the managed container does not exist, resolve the existing local `localhost/efctl-sui-dev` image and label it as a fallback, not the image proven to have crashed.

Alternative: inspect only the current image tag. Rejected because rebuilding can make a tag refer to a different image than the failed container used.

### 3. Default to bounded version/CPU probes, with an isolated local-image fallback

Host Sui version uses only `sui --version` and is labelled `host`. It does not depend on client configuration and never calls `sui client`; existing guarded client gatherers remain unchanged.

If the managed Sui container is running, capture separate read-only version, machine-architecture, and CPU-source probes through exec. Label this data `managed container`. If it races with exit, fall back once to the inspected immutable image ID.

For a stopped/absent managed container with an existing local image, create a uniquely named disposable probe using that image ID. Override its entrypoint with a fixed diagnostic command, disable networking, make the root filesystem read-only, drop capabilities, set no-new-privileges and resource/PID limits, and disable core dumps. Do not attach workspace/config/database volumes, supply credentials, or execute the image's normal entrypoint. Enforce no image pulls. Capture independent results so a failed Sui version command does not suppress CPU information. Label this data `local-image probe`: it represents the current server execution environment, not necessarily historical container state or a volume-mounted replacement binary.

The sandbox is a transient runtime object, not a restart of the development environment. Track its unique name before creation and force-remove only that probe on normal completion, failure, timeout, and cancellation; report cleanup failure with the precise safe removal target. Never remove existing containers. If the engine cannot enforce isolation or execute the probe, skip it and explain why instead of weakening protections. Volumes declared in the image must not create persistent anonymous storage; reject such images for probing unless suppression/cleanup is verified.

Alternative: use exec only, reporting the version unavailable when stopped. Rejected because startup failures commonly leave no executable managed container, defeating the one-command support goal. An opt-in debug flag is unnecessary for these bounded diagnostics; disclose the transient probe in help/docs.

### 4. Bound and sanitize new diagnostic subprocesses

Use an injectable context-aware runner that captures exit status, timeout, limited stdout/stderr, and signal information when available. Use a shared 20-second budget for new external probes, a maximum five seconds per command, and a separate maximum three-second cleanup allowance. Existing gatherers are not silently claimed to meet this new budget. Propagate command cancellation into the new probes.

Cap captured output at 64 KiB per stream, handle overlong input explicitly, and render concise normalized errors (at most 512 characters). Select structured fields instead of printing entire inspect responses. Redact secret-bearing values and escape terminal control characters before new fields/errors reach the report. Omit unrestricted probe stderr when safe redaction cannot be assured. No shell interpolation of user values: engine arguments are structured; any in-container shell snippet is fixed diagnostic code.

Alternative: reuse error-swallowing convenience methods or print raw stderr. Rejected because absence, crashes, timeouts, and unsupported commands would be indistinguishable or expose sensitive data.

### 5. Interpret evidence cautiously and retain existing CLI behavior

Present new information alongside the existing sections without removing fields or making unavailable diagnostics fail the overall command. A server/image architecture mismatch can produce `possible emulation; not confirmed`. Missing instruction data never means `AVX unsupported`. A version probe that receives SIGILL reports that observation; it does not diagnose the instruction or equate a successful version probe with node health. A managed container's exit code is container state, not necessarily the child Sui process's exit status.

Tests use injected command fixtures and parsers for high-volume unit coverage, rendering tests for user-facing distinctions, and a small Docker/Podman integration suite for running/stopped/image-only cases and sandbox cleanup. Tests must also assert absence of managed mutations, image pulls, client initialization, and secrets.

### 6. Upgrade the project baseline to Go 1.27.1 (approved scope extension)

Set `go 1.27.1` in `go.mod` and update the README source-build requirement. Existing setup-go steps already consume `go.mod`; retain that single source of truth rather than introduce duplicate version strings or a redundant toolchain directive. The directive sets the minimum supported build toolchain; developers can use compatible newer releases. This does not change the independently built upstream Sui images or claim to fix SIGILL.

Go 1.27 maintains the Go 1 compatibility promise, but its standard library uses new language features, including generic methods, that older source-analysis binaries cannot parse. Verify a current released `govulncheck`, compiled using Go 1.27.1 with checksum-authenticated modules, under repository `./tmp`. Do not overwrite global tools or introduce scanner dependencies into the application module. The isolated hook harness may use a read-only override directory for updated tools while retaining other existing host tooling.

Add test-first guards for the module baseline, matching source-build documentation, and workflow selection from `go.mod`. Run the full suite, race checks, module verification/tidy-diff checks and representative Linux/macOS/Windows cross-compilation inside disposable Bubblewrap source snapshots. Record compiled-target evidence separately from live platform verification. Application module requirements/checksums should remain unchanged apart from the Go directive unless a separately reviewed compatibility issue requires otherwise.

## Risks / Trade-offs

- [Default doctor creates a transient probe] → Document this behavior; use immutable already-local image identity, strict isolation, bounded execution, and verified cleanup. Skip when safety cannot be maintained.
- [Server uses a VM/remote host or CPU emulation] → Keep local/server/image/probe sources separate and report only evidence-supported hints.
- [Managed container uses volume-mounted binaries/configuration] → Label fallback image probes explicitly; do not imply exact reproduction of that execution context.
- [CLI versions differ or a Docker wrapper is Podman] → Parse selected structured fields with tested engine-specific fallbacks; report unsupported metadata rather than substituting client details.
- [CPU flags vary across cores or OS APIs] → Use complete conservative intersections where possible and explicit unavailable/incomplete status otherwise.
- [Version probe itself crashes or tries to write cache files] → Preserve its failure status, forbid writable persistent data/core dumps, and continue CPU probes without relaxing isolation.
- [Logs and existing config can contain secrets] → Preserve existing guards and exercise the complete doctor report with credential fixtures; do not introduce raw inspect/env dumps. Broader unrelated redaction work is out of scope.
- [Killing a runtime CLI does not stop its created container] → Force-remove the tracked probe through a separate cleanup deadline; make failure visible.

## Migration Plan

Additive report changes require no configuration or data migration. Implement tests alongside gatherers, then rendering and documentation. Validate Go tests and pre-commit plus targeted container integration checks before release. Users update efctl and run `efctl doctor` from their workspace, reviewing the report before sharing it. Rollback removes the new gatherers and fields without touching managed images or volumes. The approved Go upgrade raises the source-build minimum to 1.27.1; CI/release follows `go.mod`, with no runtime data migration. A toolchain rollback restores the previous directive/docs after compatibility review, without changing managed images or volumes. Archive this OpenSpec change only after implementation and verification, before the final commit/PR.

## Open Questions

- The failing user's exact Sui version, CPU capabilities, and runtime/emulation environment remain unknown; diagnostics must not encode a hypothesized cause as fact.
- Platform/engine fixtures need to establish which optional VM/remote indicators are reliable. Unsupported indicators remain unavailable and do not block the core report.
