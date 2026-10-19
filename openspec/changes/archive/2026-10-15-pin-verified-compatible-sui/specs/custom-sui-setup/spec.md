## MODIFIED Requirements

### Requirement: Transactional Sui preparation
The system SHALL validate a supported WORKDIR insertion point and custom input before removing installation instructions. Invalid preparation MUST preserve the Dockerfile and entrypoint and prevent building. Supported managed installation forms SHALL transition to exactly one selected installation without removing unrelated instructions.

#### Scenario: Insertion point is absent
- **WHEN** a Dockerfile contains working Sui installation but no supported WORKDIR /workspace
- **THEN** preparation fails and both scaffold files remain unchanged

#### Scenario: Supported custom and default installations
- **WHEN** preparation finds a supported layout and a valid custom input or no custom input
- **THEN** it produces exactly one selected installation and remains idempotent despite comment changes, indentation or CRLF
- **AND** the custom executable is selected unambiguously without an installation shadowing it
- **AND** the default path installs the repository-pinned release tag rather than a floating suiup release resolution

#### Scenario: Managed installation transitions
- **WHEN** preparation encounters a legacy floating suiup block, a previously generated pinned block, or a custom installation block
- **THEN** it replaces the managed installation with exactly one currently selected custom or pinned default block
- **AND** adding or removing a valid custom executable switches installation paths without leaving a shadowing installation
- **AND** unrelated Dockerfile instructions remain unchanged

#### Scenario: Invalid custom input or unknown installation layout
- **WHEN** a supplied custom executable is invalid or the Dockerfile contains unsupported Sui installation instructions
- **THEN** preparation fails before building
- **AND** both scaffold files remain unchanged

### Requirement: Reject unhydrated custom inputs
The system SHALL inspect a bounded prefix without executing the supplied artifact. An unhydrated Git LFS pointer or invalid supplied file MUST produce an actionable error rather than fallback. A valid custom executable SHALL retain precedence over the pinned default release and SHALL NOT be represented as a release-pinned artifact.

#### Scenario: Git LFS pointer is supplied
- **WHEN** docker/sui contains the Git LFS pointer header
- **THEN** preparation fails with hydration guidance without modifying scaffold files

#### Scenario: Custom executable is absent
- **WHEN** docker/sui does not exist
- **THEN** preparation selects the pinned default release installation
- **AND** no floating suiup release resolution remains selected

#### Scenario: Valid custom executable is supplied
- **WHEN** docker/sui passes the existing bounded custom-input validation
- **THEN** preparation selects the custom installation without selecting the default release path
- **AND** the default release pin is not misrepresented as validation of the custom executable

#### Scenario: Supplied custom input is invalid
- **WHEN** docker/sui exists but fails existing file validation
- **THEN** preparation fails with an actionable error without modifying scaffold files
- **AND** it does not silently fall back to the default installation
