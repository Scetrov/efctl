# Collecting Sui startup diagnostics

1. Update efctl using your normal installation method.
2. From the affected workspace, run `efctl doctor` (or `efctl doctor --workspace <path>`).
3. Review the entire report before sharing it with support. Existing logs, configuration and repository URLs may contain sensitive data; remove any credentials, keys or recovery phrases. New diagnostic metadata is sanitized, but that is not a guarantee about every existing report field.

The report separates efctl's build platform from the local machine architecture and CPU view visible to efctl. WSL, VMs and remote runtimes can expose different views. Runtime client/server versions and server platform are separate; missing server data is not replaced with client or local platform data. Remote/VM hints require observable evidence and may be unavailable.

Linux instruction flags are sorted and deduplicated, using a conservative intersection when all per-CPU sets are complete. Heterogeneous sets are identified. Missing, incomplete or unsupported sources are **unavailable**, not proof that AVX or any other instruction is absent. Windows CIM does not provide a complete instruction feature set. macOS flag sources can also be unavailable, especially on ARM.

For `sui-playground`, doctor reports recorded container state and exit code, image reference and immutable image ID, image platform and recorded repository digests. Container exit code is not necessarily the Sui process's exit status. An image built locally may have no digest. If there is no managed container, the existing `localhost/efctl-sui-dev` image is explicitly labelled a configured-image fallback, not the proven startup image. A rebuilt tag never overrides the recorded image ID of an existing managed container.

## Version and CPU probes

Host Sui is queried with `sui --version`, independently of client configuration. Doctor does not initialize an unconfigured host client or generate keys/recovery phrases. Existing client queries remain guarded by client configuration existence.

A running managed container receives separate read-only Sui version, `uname -m` and `/proc/cpuinfo` queries. Failure of one query does not suppress the others. If it exits during collection, doctor may attempt one local-image fallback.

For a stopped/absent managed container, doctor may create a uniquely named `efctl-doctor-<random>` container using the immutable **already-local** image. It overrides the entrypoint, disables networking/healthchecks/core dumps, uses a read-only root, drops all capabilities, enables no-new-privileges, and limits memory, CPU and PIDs. It attaches no host/workspace/database/configuration volumes and injects no credentials. Images declaring volumes (or with unknown volume metadata) are skipped. Unsupported restrictions cause probing to be skipped rather than weakened. No images are pulled and the managed environment is never started or restarted.

Local-image observations describe the current runtime server execution view, not necessarily the historical failed container or a volume-mounted replacement binary. Server/image architecture mismatch suggests **possible emulation; not confirmed**. Neither mismatch nor a successful version command diagnoses a startup crash. An observed SIGILL is reported as a probe failure, without identifying its cause.

New external probes share a 20-second collection budget, with at most five seconds per command. Captured output is limited to 64 KiB per stream; over-limit sources are unavailable. Error excerpts are limited to 512 characters, omit raw stderr, redact potentially sensitive fields and escape terminal controls. Legacy gatherers are not covered by this new budget.

## Opt-in crash correlation

Default `efctl doctor` does not query systemd, `coredumpctl`, `journalctl`, or a debugger, and it does not add a crash section. `efctl doctor --crash` appends that section after the existing report. Unavailable crash evidence does not by itself change the doctor exit status. The isolated version and CPU probe still disables core dumps; `--crash` does not start `sui-playground`, enable dumps, or raise `RLIMIT_CORE`.

Container exit code `132` and an illegal-instruction log line are shell/container hints, not the background `sui start` child status and not journal attribution. On a proven-local Linux kernel, doctor may read field-limited crash metadata for the inspected `sui-playground` container id. Remote engines, Docker Desktop, an unestablished remote/VM boundary, macOS, Windows, and a missing `coredumpctl` skip that query. Doctor does not SSH into a Podman machine or remote context, prompt for polkit, or install `coredumpctl` or `gdb`.

The crash section does not include crash environment, process maps, aux vectors, resource limits, open file descriptors, or core-file paths. An unwind, when `gdb` is already installed and one attributed core is present, is a non-interactive instruction view and at most eight frames. efctl does not write a core into the workspace or repository. `coredumpctl debug` may still use its own temporary file; efctl does not name or retain that path.

Crash collection has a separate 20-second budget, at most five seconds per metadata command, and at most eight seconds for one debugger invocation. Review the section before sharing it. Do not attach core files, crash environment, or maps to a support request.

## Cleanup

Doctor attempts ownership-verified force-removal of only its uniquely labelled disposable probe on completion, failure, timeout and cancellation, with a separate three-second allowance. Anonymous volumes are not intentionally created. A cleanup failure identifies the precise tracked target. Verify its `org.efctl.doctor-probe` label and identity before manually removing it; never remove `sui-playground` as diagnostic cleanup. If ownership cannot be verified, no object is removed automatically.

## Developer test isolation (Linux)

Run `tools/test-isolated.sh` for the full offline Go suite. It copies tracked and non-ignored working files, without `.git` or managed checkouts, into a disposable Bubblewrap filesystem. Host homes, Git metadata, runtime sockets and container storage are not exposed. Fixtures, HOME and build caches are backed by repository `./tmp`, outside the copied source tree so ancestor config/repository discovery cannot affect the real checkout. Failed scratch directories are retained for inspection. Go modules must already be cached; Bubblewrap and Go are required. `tools/test-isolated.sh --pre-commit` runs the configured hooks in a fresh sandbox checkout using copies of already-cached hook environments and read-only host tooling. This mode allows network access for security scanners, but still exposes no host credentials, checkout metadata or runtime sockets; default test mode remains offline.

Run `tools/test-doctor-runtime-isolated.sh <existing-local-image-id>` for focused doctor integration checks. Use an immutable local Linux image providing `/bin/sh`, `sleep`, `uname` and `cat`. The harness exports it offline, establishes a Podman user namespace with `podman unshare`, then runs Bubblewrap with a dedicated VFS store under `./tmp`. It exercises both Podman and a Docker-compatible Podman wrapper; it does not connect to a host Docker socket. The simulated Sui executable tests collection/cleanup, not upstream Sui compatibility. Direct rootless Podman inside Bubblewrap may fail UID mapping; the explicit outer user namespace avoids that failure without relaxing diagnostic sandbox flags.

See [command help](efctl_doctor.md) for invocation details.
