## 1. Crash result model and fixtures

- [x] 1.1 Add failing unit tests for a crash result model that distinguishes container hints, skipped correlation, unscoped records, and one attributed record.
- [x] 1.2 Add failing tests that a fixture secret in `COREDUMP_ENVIRON`, maps, and core-file paths is dropped before the result is rendered.
- [x] 1.3 Implement the model, allowlisted field copy, and sanitizer using the existing doctor redaction and terminal-escape helpers.

## 2. Container hints

- [x] 2.1 Add failing tests for exit code `132`, one illegal-instruction log excerpt, an absent container, and an image id that must not be used as the container id.
- [x] 2.2 Add a crash-only inspect template for container id, status, exit code, `StartedAt`, and `FinishedAt`, with Docker and Podman fixture coverage.
- [x] 2.3 Implement bounded log scanning and hint labels that state the shell/container boundary.

## 3. Local-kernel gate

- [x] 3.1 Add failing tests that remote, Docker Desktop, unestablished boundary, non-Linux `GOOS`, and missing `coredumpctl` skip helper execution.
- [x] 3.2 Implement the gate using existing engine boundary evidence and local `PATH` lookup, without SSH or a remote context.
- [x] 3.3 Cover pager-disabling environment and deadline cancellation with an injected runner.

## 4. Journal attribution

- [x] 4.1 Add failing parser tests for `coredumpctl --json=short` discovery and field-selective `journalctl` JSON, including malformed input and extra keys.
- [x] 4.2 Add failing correlation tests for full container-id cgroup match, short-id rejection, unscoped `sui` records, missing start time, the six-hour cap, and the 30-second finish grace.
- [x] 4.3 Implement structured, validated helper arguments and the newest-of-many selection rule.
- [x] 4.4 Add a failing test that unsupported field selection does not fall back to `coredumpctl info` or an unfiltered export.

## 5. Bounded unwind

- [x] 5.1 Add failing tests that unwind runs only for one attributed present core when `gdb` is on `PATH`, and that the debugger arguments exclude locals, environment, and an output path.
- [x] 5.2 Implement the single `coredumpctl debug` invocation, 8 KiB capture, 20-line sanitized render, and metadata retention when gdb or the core is absent.

## 6. CLI, docs, and budgets

- [x] 6.1 Add a failing command test that default `efctl doctor` has no crash section and does not invoke crash helpers.
- [x] 6.2 Add `--crash` to the doctor command, render the section only when set, and keep doctor exit status independent of unavailable crash data.
- [x] 6.3 Enforce the separate 20-second budget, five-second metadata deadline, and eight-second debugger deadline.
- [x] 6.4 Update doctor help, generated command docs, and `docs/doctor-support.md` with the opt-in limits and the review-before-sharing warning.

## 7. Verification

- [x] 7.1 Run the new doctor unit tests and the existing doctor suite.
- [x] 7.2 Run `pre-commit` on the changed files and fix any gate it reports.
