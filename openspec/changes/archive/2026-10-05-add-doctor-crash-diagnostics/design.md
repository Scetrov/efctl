## Context

`sui start` is backgrounded by the image entrypoint. The shell then waits for RPC and exits `1` on timeout, so a child SIGILL often never becomes container exit code `132` and may never be printed. `WaitForLogs` only polls container logs for the ready sentinel. Default `efctl doctor` already records container exit code, image identity, CPU flags, and an isolated version probe that sets `--ulimit core=0:0`. Its spec forbids diagnosing SIGILL from version success or architecture mismatch, and its design lists core-dump analysis as a non-goal.

Host crash state is a different evidence source. `systemd-coredump` can record `COREDUMP_SIGNAL`, `COREDUMP_EXE`, `COREDUMP_CGROUP`, and `COREDUMP_CONTAINER_CMDLINE` (cgroup and container fields since systemd 248). It also records `COREDUMP_ENVIRON` and process maps. `coredumpctl info` and an unfiltered journal export are not safe to print. Unprivileged `coredumpctl` only sees the caller's journal entries. A dump from a rootless container uid is often invisible. Docker Desktop and remote engines crash on a different kernel from the machine running efctl.

## Goals / Non-Goals

**Goals:**

- Add opt-in `efctl doctor --crash` that appends a crash section without changing default report collection or output.
- Say when container exit code or logs suggest an illegal instruction, and that this is not the child process status.
- On a proven-local Linux kernel, attribute at most a bounded set of journal crashes to the inspected `sui-playground` container id.
- When one attributed dump is readable, collect a short non-interactive instruction view without efctl writing a core file.
- Make every skip, permission failure, missing tool, and empty result explicit.

**Non-Goals:**

- Fix SIGILL, change `env up`, change the entrypoint, enable core dumps, or raise `RLIMIT_CORE`.
- Run crash queries from default doctor, from the isolated version/CPU probe, or by starting `sui-playground`.
- Extract, copy, or render a core file. Print environment, maps, file descriptors, or stack locals.
- Install `systemd-coredump`, `gdb`, or `eu-stack`. Infer AVX, QEMU, or Rosetta from a crash record.
- Support remote engines, Docker Desktop, Podman machine, macOS, or Windows by shelling into a VM.
- Add a JSON report mode or a new module dependency.

## Decisions

### 1. Keep crash collection off the default path

Add a boolean `Crash bool` to doctor options, set only by `--crash`. `Gather` stays responsible for the existing report. A separate function runs only when the flag is set and its result is rendered only then. Default doctor must not spawn `coredumpctl`, `journalctl`, or a debugger, and must not inspect extra container fields.

Crash collection uses its own 20-second budget and 5-second per-command deadline, plus at most 8 seconds for one debugger invocation. It does not consume or extend the existing probe budget. Unavailable crash data does not change doctor's exit status.

Alternative: fold the checks into every `efctl doctor`. Rejected because the archived diagnostics spec excludes core-dump analysis from the default command, and a journal query can hang on a pager or permissions.

### 2. Treat container text and exit code as hints, not attribution

When `--crash` is set, inspect `sui-playground` with a crash-only template that selects container id (`.Id`, not the image id), status, exit code, `StartedAt`, and `FinishedAt`. Decode exit code `132` as "container status compatible with SIGILL". Scan a bounded log tail for a case-insensitive `illegal instruction` line and quote at most one sanitized 512-character excerpt.

Both hints must be labelled as container/shell evidence. They do not by themselves select a journal entry. If the container is absent, say so and skip journal correlation rather than matching any recent `sui` dump.

Alternative: watch logs during `env up`. Rejected for this change. The entrypoint swallows the child status, so the reliable host evidence is the journal, queried after the fact.

### 3. Query only when the kernel boundary is already proven

Require all of the following before any crash helper runs:

- `runtime.GOOS` is `linux`.
- The selected engine's existing boundary evidence does not say remote or Docker Desktop.
- That boundary is established. "not established" skips the query.
- `coredumpctl` is found on `PATH` of the efctl process. Do not SSH, `podman machine ssh`, or use a remote Docker context.

Set `SYSTEMD_PAGER=cat` and pass coredumpctl's no-pager option. Kill the process at the deadline so a pager or agent cannot block the CLI. A non-zero status or timeout is an unavailable reason, not a retry with elevated privileges.

Alternative: run `coredumpctl` whenever the binary exists. Rejected because a local binary can see the wrong kernel's journal, or no journal, and a false "no crash" is worse than a skip.

### 4. Attribute with an allowlisted journal query, not `coredumpctl info`

Use `coredumpctl --json=short list` only as a discovery index inside the container's start window. Parse it as untrusted JSON and copy only allowlisted fields into memory: time, pid, signal, executable, core-file presence, and size. Discard every other key, including any environment or map payload, before rendering.

Correlation uses `journalctl` with `MESSAGE_ID=fc2e22bc6ee647b6b90729ab34a250b1`, `--output=json`, and `--output-fields` limited to:

- `COREDUMP_TIMESTAMP`
- `COREDUMP_PID`
- `COREDUMP_SIGNAL`
- `COREDUMP_SIGNAL_NAME`
- `COREDUMP_EXE`
- `COREDUMP_COMM`
- `COREDUMP_CGROUP`
- `COREDUMP_CONTAINER_CMDLINE`
- `COREDUMP_FILENAME` presence only, not contents
- `COREDUMP_HOSTNAME`

The window is `StartedAt` minus 2 seconds through `FinishedAt` plus 30 seconds, or now if the container is still running. Cap the window at 6 hours. Missing `StartedAt` skips correlation.

A dump is attributed only when `COREDUMP_CGROUP` contains the full container id. A 12-character prefix is not enough. `COREDUMP_CONTAINER_CMDLINE` or `COMM=sui` without that cgroup match is reported as unscoped and is not unwound. If field-selective `journalctl` is unsupported, stop. Do not fall back to `coredumpctl info` or an unfiltered export.

If several dumps attribute, report the count and unwind only the newest. Other signals on an attributed dump are shown as observed, not relabelled SIGILL.

Alternative: print `coredumpctl info`. Rejected because its output includes environment and maps and is not field-selective.

### 5. Unwind through `coredumpctl debug` without an output path

If exactly one attributed dump is selected, `gdb` is on `PATH`, and the index says a core file is present, run:

`coredumpctl debug COREDUMP_PID=<pid>` with `--debugger=gdb` and `--debugger-arguments` containing only `-batch`, pagination off, `x/i $pc`, and a backtrace limited to 8 frames.

Do not pass `--output`, do not call `dump`, and do not choose a path under the workspace or `./tmp`. Do not ask gdb for locals, `bt full`, memory strings, or environment. Capture at most 8 KiB and render at most 20 sanitized lines. Missing gdb, a missing core, or a debugger failure leaves the metadata section intact and marks the unwind unavailable.

Alternative: `eu-stack` after `coredumpctl dump -o`. Rejected because that writes a secret-bearing core to a path efctl would own.

### 6. Bound and redact before render

Reuse the existing diagnostic sanitizer for control characters and secret-shaped tokens. Crash rendering has a hard denylist: environment, maps, limits, auxv, open file descriptors, and any line that looks like a core-file path under `/var/lib/systemd/coredump`. Tests must feed fixtures containing a known secret in `COREDUMP_ENVIRON` and assert the secret never appears in the rendered section or in error strings.

No shell interpolation. Container id, pid, and timestamps are validated before they become arguments (hex id, decimal pid, RFC3339 timestamp).

## Risks / Trade-offs

- [Rootless dump owned by a mapped uid] → Report permission or empty journal explicitly; do not escalate.
- [systemd older than 248 has no cgroup field] → Skip attribution; do not match on executable name.
- [coredumpctl JSON shape differs by version] → Allowlist parse; malformed index is unavailable, not a guessed row.
- [Async coredump arrives after `FinishedAt`] → 30-second grace, then stop. Do not poll.
- [`coredumpctl debug` may still create a tool temporary core] → efctl must not name or retain that path; document the helper's behavior.
- [Gdb backtrace can still contain secret stack strings] → No locals, 8 frames, 20 rendered lines, sanitizer. This remains a residual risk of the opt-in unwind.
- [Docker info does not currently select a remote flag] → Reuse existing boundary evidence. If it is only "not established", skip rather than adding a new default-doctor field.
- [Default report users never learn the flag exists] → Document it in command help and `docs/doctor-support.md`. Do not add a hint line to the default report.

## Migration Plan

Additive flag and docs. No config, volume, or image migration. Rollback removes the flag and crash collector; default doctor behavior is unchanged. Users who want the section pass `--crash`, review it, and do not attach core files to support requests.

## Open Questions

- None that block implementation. A live spike on an affected host can later show whether that host's journal is readable, but the command must already behave correctly when it is not.
