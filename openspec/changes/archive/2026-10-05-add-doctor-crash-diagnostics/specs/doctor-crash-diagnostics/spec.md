## ADDED Requirements

### Requirement: Crash checks are opt-in

Default `efctl doctor` SHALL NOT query systemd, `coredumpctl`, `journalctl`, or a debugger, and SHALL NOT add a crash section to the report. `efctl doctor --crash` SHALL run crash collection after the existing report and SHALL render a crash section even when every crash field is unavailable. Crash collection failure SHALL NOT by itself change the doctor exit status. The existing isolated version and CPU probe SHALL continue to disable core dumps and SHALL NOT be started, altered, or replaced by the crash path.

#### Scenario: Default doctor does not read crash state

- **WHEN** the user runs `efctl doctor` without `--crash`
- **THEN** no crash helper is executed
- **AND** the rendered report has no crash section

#### Scenario: Opt-in section is always present

- **WHEN** the user runs `efctl doctor --crash` and crash collection cannot obtain evidence
- **THEN** the report still contains the existing sections
- **AND** the crash section states the unavailable reason
- **AND** doctor does not exit nonzero solely because crash evidence is unavailable

### Requirement: Container signal hints stay labelled as shell evidence

When `--crash` is set and `sui-playground` exists, doctor SHALL read that container's id, exit code, start time, and finish time from selected inspect fields. It SHALL NOT use the image id as the container id. Exit code `132` SHALL be described as compatible with SIGILL at the container status boundary. A bounded log tail containing a case-insensitive illegal-instruction message SHALL be reported with at most one sanitized excerpt. The crash section SHALL state that container exit code and log text are not the child Sui process status and do not by themselves attribute a journal crash.

#### Scenario: Container exit code is 132

- **WHEN** `--crash` is set and the managed container's recorded exit code is `132`
- **THEN** the crash section reports that container status as compatible with SIGILL
- **AND** it does not claim the background `sui start` child was the process that exited

#### Scenario: Logs contain an illegal-instruction line

- **WHEN** `--crash` is set and the bounded container log tail contains an illegal-instruction message
- **THEN** the crash section quotes at most one sanitized excerpt of that message
- **AND** it labels the excerpt as container log text

#### Scenario: Managed container is absent

- **WHEN** `--crash` is set and `sui-playground` cannot be inspected
- **THEN** journal correlation is skipped
- **AND** the crash section says the managed container identity is unavailable

### Requirement: Journal queries require a local Linux kernel

Doctor SHALL run `coredumpctl` or `journalctl` only when efctl itself is on Linux, the selected engine boundary is established, and that boundary is neither remote nor Docker Desktop. It SHALL look up `coredumpctl` on the local `PATH` and SHALL NOT reach a container host through SSH, a Podman machine, or a remote engine context. It SHALL disable pager interaction and enforce a command deadline. A missing helper, unsupported platform, unestablished boundary, timeout, or permission failure SHALL be an unavailable reason with no privilege escalation and no retry.

#### Scenario: Engine is remote or Docker Desktop

- **WHEN** `--crash` is set and existing engine evidence reports a remote service or Docker Desktop
- **THEN** doctor does not run `coredumpctl` or `journalctl`
- **AND** the crash section explains that the container kernel is not the local journal

#### Scenario: Boundary is not established

- **WHEN** `--crash` is set and the engine remote/VM boundary is not established
- **THEN** doctor does not query crash helpers
- **AND** the crash section reports that skip explicitly

#### Scenario: coredumpctl is absent

- **WHEN** `--crash` is set on a qualifying local Linux engine and `coredumpctl` is not on `PATH`
- **THEN** the crash section reports the helper as unavailable
- **AND** doctor does not install or download it

### Requirement: Attributed crash metadata is field-limited

On a qualifying local engine, doctor SHALL search crash records only inside a window beginning two seconds before the container start time and ending thirty seconds after the container finish time, or at the current time if the container is running. The window SHALL NOT exceed six hours. Missing start time SHALL skip correlation. A record SHALL be attributed only when `COREDUMP_CGROUP` contains the full inspected container id. Matches on executable name, command name, hostname, or container cmdline without that cgroup match SHALL be reported as unscoped and SHALL NOT be unwound. The rendered metadata SHALL be limited to timestamp, pid, signal name and number, executable, cgroup, container cmdline, and whether a core file is present. If more than one record attributes, doctor SHALL report the count and select only the newest for unwind.

#### Scenario: Cgroup contains the container id

- **WHEN** a crash record in the start window has `COREDUMP_CGROUP` containing the full managed container id
- **THEN** the crash section reports that record's allowlisted metadata
- **AND** it identifies the record as attributed to `sui-playground`

#### Scenario: Only the executable name matches

- **WHEN** a recent crash record names `sui` but its cgroup does not contain the full container id
- **THEN** the crash section reports the record as unscoped
- **AND** it does not unwind that record

#### Scenario: Start time is absent

- **WHEN** the managed container inspect has no start time
- **THEN** doctor does not search the journal
- **AND** the crash section reports why correlation was skipped

#### Scenario: Journal query cannot exclude fields

- **WHEN** the local `journalctl` cannot emit the allowlisted crash fields without the rest of the record
- **THEN** doctor does not fall back to `coredumpctl info` or an unfiltered export
- **AND** metadata is unavailable with that reason

### Requirement: Unwind does not extract a core

Doctor SHALL invoke a debugger only for one attributed record whose index says a core file is present, and only when `gdb` is already on `PATH`. The invocation SHALL be `coredumpctl debug` in batch mode with arguments limited to the instruction at the program counter and a backtrace of at most eight frames. Doctor SHALL NOT call `coredumpctl dump`, SHALL NOT pass an output path, and SHALL NOT write a core into the workspace or repository. Debugger output SHALL be capped and sanitized before rendering. Missing `gdb`, a missing core, or debugger failure SHALL leave attributed metadata in place and mark only the unwind unavailable.

#### Scenario: Attributed core and gdb are available

- **WHEN** one attributed record has a core file and `gdb` is on `PATH`
- **THEN** doctor runs one non-interactive debugger invocation for that record
- **AND** the rendered unwind contains at most the instruction view and eight frames
- **AND** no core file is created by efctl under the workspace or repository

#### Scenario: gdb is not installed

- **WHEN** an attributed record has a core file and `gdb` is absent
- **THEN** allowlisted metadata is still rendered
- **AND** the unwind is reported unavailable
- **AND** doctor does not install a debugger

### Requirement: Crash output is credential-safe and bounded

Crash helper stdout and stderr SHALL be captured under an explicit byte limit. Rendered crash excerpts SHALL be at most 512 characters, except the unwind block which SHALL be at most 20 sanitized lines. Doctor SHALL NOT render crash environment variables, process maps, aux vectors, resource limits, open file descriptors, or core-file paths. Arguments built from container id, pid, and timestamps SHALL be validated before execution. Crash collection SHALL complete within a 20-second budget, with at most five seconds per metadata command and at most eight seconds for the debugger. Cancellation SHALL stop in-flight crash helpers.

#### Scenario: Journal fixture contains an environment secret

- **WHEN** a crash fixture includes a secret in `COREDUMP_ENVIRON` or another non-allowlisted field
- **THEN** the rendered crash section and its errors do not contain that secret
- **AND** the secret is not retained in the crash result model

#### Scenario: A crash helper hangs

- **WHEN** `coredumpctl`, `journalctl`, or `gdb` exceeds its deadline
- **THEN** doctor cancels that helper
- **AND** the crash section reports timeout without waiting for a pager or privilege prompt

#### Scenario: Helper output is hostile

- **WHEN** crash helper output contains terminal control characters or exceeds the capture limit
- **THEN** rendered text is escaped or truncated
- **AND** the overrun is explicit rather than silently concatenated into the report
