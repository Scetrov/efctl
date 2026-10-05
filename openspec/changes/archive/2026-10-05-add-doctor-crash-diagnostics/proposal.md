## Why

A Sui startup SIGILL is usually a background child of `sui-playground`, so `efctl env up` often reports the shell's RPC-timeout exit rather than signal 4, and the container log stream may never contain "Illegal instruction". Default `efctl doctor` correctly refuses to analyse core dumps. Operators still need an explicit, local, credential-safe way to ask whether the host recorded a matching crash and, if so, what little evidence can be shown without extracting a core.

## What Changes

- Add an opt-in `efctl doctor --crash` mode that performs host crash-dump checks after the existing doctor report, without changing default `efctl doctor`.
- Detect likely swallowed SIGILL evidence already available to efctl: managed-container exit code `132`, and bounded log text matching an illegal-instruction message. Report that the container exit code is the shell's status, not proof of the child signal.
- On a local Linux engine only, query `coredumpctl` for a recent dump whose cgroup or container metadata matches the inspected `sui-playground` container ID. Print an allowlisted summary (signal, timestamp, executable, cgroup/container identity) or an explicit unavailable reason.
- When a matched dump is readable, optionally collect a bounded non-interactive unwind: signal, instruction pointer, one disassembled instruction, and module+offset. Do not write a core file into the workspace or repository.
- Refuse the crash path for remote engines, Docker Desktop/Podman machine contexts where `coredumpctl` is not on the same kernel as the container, macOS, and Windows. Do not prompt for polkit; a permission failure is an unavailable reason.
- Never print core-dump environment, process maps, open file descriptors, or raw core bytes. Never enable core dumps, raise `RLIMIT_CORE`, start `sui-playground`, or relax the existing isolated version/CPU probe (which keeps `--ulimit core=0:0`).

## Capabilities

### New Capabilities

- `doctor-crash-diagnostics`: Opt-in host crash correlation and bounded unwind for the managed Sui container, with explicit unavailable reasons and a credential boundary that forbids core extraction and environment disclosure.

### Modified Capabilities

- None. Default `efctl doctor` collection, isolation, budgets, and the rule against diagnosing SIGILL from version or architecture evidence remain unchanged. `--crash` is a separate capability and does not run those probes' core-dump settings in reverse.

## Impact

- CLI: `cmd/doctor.go` flag, help text, and report rendering. Generated command docs and `docs/doctor-support.md` must describe the opt-in path, its platform limits, and the review-before-sharing warning.
- Code: new focused package logic under `pkg/doctor` (or a sibling package) with injected command runners. No new module dependencies. No container create/start changes and no `builder-scaffold` entrypoint changes.
- Systems: local systemd-coredump and, for the unwind, a host `gdb` or `eu-stack` if already installed. Absence of either tool is a reported gap, not an install.
- Security: crash metadata can still identify a local process; environment and core bytes are secret-bearing and out of scope. Polkit prompts must not block the command.
