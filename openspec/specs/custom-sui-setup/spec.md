# custom-sui-setup Specification

## Purpose
TBD - created by archiving change harden-custom-sui-setup. Update Purpose after archive.
## Requirements
### Requirement: Transactional Sui preparation
The system SHALL validate a supported WORKDIR insertion point and custom input before removing installation instructions. Invalid preparation MUST preserve the Dockerfile and entrypoint and prevent building.

#### Scenario: Insertion point is absent
- **WHEN** a Dockerfile contains working Sui installation but no supported WORKDIR /workspace
- **THEN** preparation fails and both scaffold files remain unchanged

#### Scenario: Supported custom and default installations
- **WHEN** preparation finds a supported layout and a valid custom input or no custom input
- **THEN** it produces exactly one selected installation and remains idempotent despite comment changes, indentation or CRLF
- **AND** the custom executable is selected unambiguously without a release installer shadowing it

### Requirement: Reject unhydrated custom inputs
The system SHALL inspect a bounded prefix without executing the supplied artifact. An unhydrated Git LFS pointer or invalid supplied file MUST produce an actionable error rather than fallback.

#### Scenario: Git LFS pointer is supplied
- **WHEN** docker/sui contains the Git LFS pointer header
- **THEN** preparation fails with hydration guidance without modifying scaffold files

#### Scenario: Custom executable is absent
- **WHEN** docker/sui does not exist
- **THEN** preparation retains the upstream suiup installation workflow
