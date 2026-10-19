## Purpose

Ensure the Sui development image supplies the runtime libraries required by project-selected pnpm.

## Requirements

### Requirement: Sui development image supplies pnpm runtime libraries
The system SHALL build the `sui-playground` image with the minimal operating-system runtime libraries required to start the project-selected pnpm executable. For Debian- or Ubuntu-based scaffold images using the affected `@pnpm/exe` distribution, the image MUST install `libatomic1` with no recommended packages and MUST retain package-index cleanup.

#### Scenario: Fresh image executes affected standalone pnpm
- **WHEN** a fresh Sui development image is built from the patched scaffold and the supported world project selects `pnpm@11.9.0`
- **THEN** the selected pnpm executable starts successfully and reports its version without a `libatomic.so.1` loader error

#### Scenario: Runtime dependency remains minimal
- **WHEN** the Dockerfile is prepared for image construction
- **THEN** it installs `libatomic1` through the existing runtime package installation layer without adding compiler toolchains or recommended packages
- **AND** apt package indexes are removed from the resulting image layer

### Requirement: Runtime dependency patch is durable and idempotent
The system SHALL encode the pnpm runtime dependency in the existing source-controlled scaffold patching flow. Repeated environment preparation MUST NOT duplicate the package entry, and a changed upstream Dockerfile that cannot be patched MUST produce a standard patch warning.

#### Scenario: Dockerfile is patched for the first time
- **WHEN** the cloned scaffold Dockerfile contains a supported package-list insertion point and does not contain `libatomic1`
- **THEN** environment preparation adds exactly one `libatomic1` package entry

#### Scenario: Dockerfile is prepared repeatedly
- **WHEN** environment preparation runs against a Dockerfile that already contains `libatomic1`
- **THEN** the Dockerfile remains unchanged and no missing-target warning is emitted for the runtime dependency

#### Scenario: Upstream Dockerfile has no supported insertion point
- **WHEN** the cloned scaffold Dockerfile contains neither `libatomic1` nor a supported package-list insertion point
- **THEN** the system emits a standard warning identifying the unapplied `libatomic1` runtime patch

### Requirement: Container boundaries remain isolated from the host
The system MUST satisfy the pnpm runtime dependency within the image that executes world deployment. It MUST NOT require `libatomic1` on the host and MUST NOT rely on manually installing the package in a running container.

#### Scenario: Linux host does not provide libatomic to containers
- **WHEN** a user creates a fresh environment on Linux with Docker and the host has no relevant library mounted into the container
- **THEN** world deployment succeeds using only libraries supplied by the `sui-playground` image

### Requirement: Partial environments remain recoverable
The system SHALL support normal teardown and recreation when a prior world deployment stopped after containers were started.

#### Scenario: Recreate after loader failure
- **WHEN** an environment is partially initialized following a pnpm loader failure
- **THEN** `efctl env down` can remove the managed containers, images, networks, and volumes according to existing cleanup semantics
- **AND** a subsequent clean `efctl env up` using the fixed candidate can recreate the environment and complete world deployment

### Requirement: Runtime fix is validated on real Docker execution
The implementation MUST include automated behavioral coverage that invokes pnpm through the same project-version selection mechanism used during world deployment. Validation on the provided Linux Mint/Docker host SHALL continue until the affected deployment path succeeds or a critical external blocker is recorded with evidence.

#### Scenario: Container smoke test exercises project-selected pnpm
- **WHEN** the regression smoke test runs against a freshly built patched image
- **THEN** it invokes pnpm from a project declaring the affected package-manager version
- **AND** the command exits successfully without a missing shared-library error

#### Scenario: Remote end-to-end validation
- **WHEN** a candidate `efctl` is transferred to `scetrov@172.29.117.128` for acceptance testing
- **THEN** validation uses normal teardown and clean creation workflows without installing host packages or mutating a running container as the fix
- **AND** world deployment completes past the pnpm step that previously exited with status 127
