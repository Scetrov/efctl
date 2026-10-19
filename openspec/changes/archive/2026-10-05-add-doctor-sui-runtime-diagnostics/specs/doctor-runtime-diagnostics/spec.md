## ADDED Requirements

### Requirement: Source-labelled CPU diagnostics

The default `efctl doctor` report SHALL include the local CPU model, observed machine architecture, and available instruction flags, separately from efctl's build platform. It SHALL label the source, normalize architecture aliases for comparison, sort and deduplicate flags, and distinguish unavailable/incomplete data from confirmed absence. For heterogeneous Linux CPUs, a uniform-support feature set SHALL use the intersection of complete per-CPU flag sets rather than a single core's flags.

#### Scenario: Linux CPU information is available
- **WHEN** doctor reads complete Linux CPU metadata and machine architecture
- **THEN** it reports the CPU model, observed architecture, and deterministic instruction flags with local source labels
- **AND** it retains the existing efctl build-platform and WSL fields

#### Scenario: CPUs expose different instruction sets
- **WHEN** complete per-CPU flag sets differ
- **THEN** the uniform-support feature set contains only flags present in every CPU set
- **AND** doctor identifies that per-CPU feature sets differ

#### Scenario: Platform metadata is unavailable
- **WHEN** a CPU source is missing, incomplete, unsupported, or fails on Linux, macOS, or Windows
- **THEN** doctor reports the affected field unavailable or incomplete with a concise reason
- **AND** it does not interpret missing metadata as absence of AVX or any other instruction

### Requirement: Runtime client and server identity

Doctor SHALL report container client and server versions separately and collect server OS/architecture through the selected Docker or Podman engine. It SHALL identify remote/virtualization indicators only when supported by collected evidence and SHALL NOT substitute the local platform or client version for missing server information.

#### Scenario: Server is reachable
- **WHEN** the selected engine returns version and server metadata
- **THEN** doctor reports client version, server version, server OS, and server architecture separately

#### Scenario: Server is unreachable or incompatible
- **WHEN** the engine client exists but the server query fails or lacks expected fields
- **THEN** doctor retains available client information and marks unavailable server fields with reasons
- **AND** other report sections still render

### Requirement: Failed container and local image identity

Doctor SHALL inspect the managed Sui container regardless of running state and report its state, available recorded exit code, image reference, and immutable image ID. It SHALL inspect the actual image ID for OS/architecture and recorded repository digests. If the container is absent, it SHALL attempt metadata inspection of the already-local configured Sui image, explicitly identifying this as a fallback. No diagnostic SHALL pull an image.

#### Scenario: Sui container has exited
- **WHEN** the managed Sui container exists but is stopped
- **THEN** doctor reports its state, available exit code, and actual image identity/platform
- **AND** it does not start the managed container or treat its exit code as the child Sui process's exit code

#### Scenario: Image tag has changed
- **WHEN** the failed container's image ID differs from the current image tag target
- **THEN** doctor uses the container's recorded image ID for metadata and fallback probing

#### Scenario: Only a local image exists
- **WHEN** no managed container exists but the configured Sui image is local
- **THEN** doctor reports its immutable image identity with a fallback source label
- **AND** it does not claim this image was used by the failed startup

#### Scenario: Image has no recorded digest or is missing
- **WHEN** an inspected image has no repository digest or the relevant local image cannot be inspected
- **THEN** doctor explicitly reports the digest not recorded or the image unavailable, respectively
- **AND** it neither invents a digest nor pulls an image

### Requirement: Distinct host and runtime Sui executable diagnostics

Doctor SHALL collect host Sui version through a version-only command independently of host client configuration. It SHALL separately collect Sui version, observed machine architecture, and CPU instruction information from the running managed Sui container or the isolated local-image fallback. Each result SHALL identify its execution source; a failure of one probe SHALL NOT suppress independently available probe results. Existing host client-configuration guards SHALL remain in force.

#### Scenario: Host client is unconfigured
- **WHEN** the host Sui executable exists but its client config does not
- **THEN** doctor can report its version using only `sui --version`
- **AND** it does not invoke host client commands or create config, keys, or recovery phrases

#### Scenario: Host Sui is absent but the managed container runs
- **WHEN** no host Sui executable is found and the managed Sui container is running
- **THEN** host version is reported absent and container version/CPU probes are attempted independently
- **AND** successful runtime results are labelled as managed-container observations

#### Scenario: Sui version probe crashes
- **WHEN** a version-only probe fails with an observed signal or nonzero exit
- **THEN** doctor reports a bounded, sanitized failure and observed signal/status when available
- **AND** it still reports independently collected CPU and architecture information

### Requirement: Isolated local-image probe after startup failure

When the managed Sui container is absent, stopped, or exits before exec and its selected image is already local, default doctor SHALL attempt a disposable probe of that immutable image ID. It MUST override the entrypoint, disable networking and core dumps, use a read-only root filesystem, drop capabilities, prevent privilege escalation, and impose resource/PID limits. It MUST NOT mount managed or host data, inject credentials, run the normal entrypoint, start Sui services, initialize a client, or alter databases/volumes. It SHALL label observations as a local-image probe rather than the historical managed execution context.

#### Scenario: Managed Sui container is stopped
- **WHEN** its actual image is local and isolation is supported
- **THEN** doctor creates a uniquely named isolated probe to collect version-only and CPU/architecture evidence
- **AND** it leaves the managed environment, images, workspace, keys, and volumes unchanged

#### Scenario: Container exits during collection
- **WHEN** managed-container exec fails because the container exited
- **THEN** doctor attempts at most one local-image fallback using the inspected image ID
- **AND** it preserves source labels and the exec failure reason

#### Scenario: Isolation cannot be guaranteed
- **WHEN** the engine cannot enforce sandbox restrictions or an image would introduce unsafely persistent anonymous volumes
- **THEN** doctor skips the disposable probe with a reason
- **AND** it does not weaken isolation or start the managed environment instead

#### Scenario: Disposable probe fails or is interrupted
- **WHEN** a probe completes, fails, times out, or doctor is cancelled
- **THEN** cleanup attempts to force-remove only the uniquely tracked disposable probe
- **AND** cleanup failure is reported with the precise probe removal target without removing any pre-existing container

### Requirement: Bounded and credential-safe diagnostic collection

New external diagnostic probes SHALL obey a shared maximum 20-second collection budget and a maximum five-second per-command deadline, with at most three additional seconds for cleanup. Captured output SHALL be limited to 64 KiB per stream; over-limit data SHALL be handled explicitly. Rendered error excerpts SHALL be limited to 512 characters, redact secrets, and escape terminal control characters. New diagnostics SHALL select metadata fields rather than dump raw environment, inspect, engine-info, or credential data. Errors/timeouts SHALL NOT prevent available existing report sections from rendering or independently cause a nonzero doctor exit.

#### Scenario: New diagnostic command hangs
- **WHEN** a new probe exceeds its command deadline or shared budget
- **THEN** doctor cancels that probe and reports timeout or budget exhaustion
- **AND** cleanup gets its separate bounded allowance and the remaining report renders

#### Scenario: Probe output is oversized or malicious
- **WHEN** a response exceeds capture limits or contains credentials or terminal escape sequences
- **THEN** memory consumption remains bounded and the affected result is truncated or marked unavailable explicitly
- **AND** new output does not expose credentials or executable terminal controls

### Requirement: Evidence-based interpretation and support guidance

Doctor SHALL retain its existing report sections and provide the new diagnostic fields without requiring a debug flag. It SHALL describe server/image architecture mismatch as possible emulation, not confirmation. It SHALL NOT diagnose the reported SIGILL solely from version success, architecture mismatch, or missing CPU metadata. Help and support documentation SHALL disclose transient isolated probing, explain unavailable fields and source boundaries, and instruct users to review reports before sharing them.

#### Scenario: Architecture aliases match
- **WHEN** server reports `amd64` and image/probe reports `x86_64`, or equivalently `arm64` and `aarch64`
- **THEN** doctor treats the normalized architectures as equal
- **AND** it does not emit a mismatch warning solely because spelling differs

#### Scenario: Image and server architectures differ
- **WHEN** normalized image and server architectures differ
- **THEN** doctor reports both observed values and identifies possible emulation as unconfirmed
- **AND** it does not assert that the mismatch caused the Sui crash

#### Scenario: User requests startup support
- **WHEN** the user runs plain `efctl doctor` after a failed startup
- **THEN** the report includes available new diagnostics alongside existing sections, with source labels and reasons for missing data
- **AND** documented guidance explains how to review and share that report without exposing secrets
