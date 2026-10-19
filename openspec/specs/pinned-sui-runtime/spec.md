# pinned-sui-runtime Specification

## Purpose
Define the repository-pinned default Sui runtime: a stable release tag selected through the suiup-managed path, reference digest recording, evidence-backed JSON-RPC compatibility acceptance, and reviewable pin maintenance.

## Requirements
### Requirement: Pinned default release selection

When no custom Sui executable is supplied, the system SHALL generate a default Sui installation that selects a repository-pinned release tag through the suiup-managed path, rather than resolving the latest release at build time. The pinned tag MUST be stable for a given efctl revision regardless of which Sui releases are published upstream.

#### Scenario: Supported default installation
- **WHEN** preparation selects the default installation
- **THEN** the generated block installs the repository-pinned release tag (for example `suiup install sui@testnet-1.81.1`)
- **AND** the block does not contain a floating `suiup install sui` resolution

#### Scenario: A newer upstream release is published
- **WHEN** an unchanged efctl revision prepares a new default installation after another Sui release is published upstream
- **THEN** the selected release tag and the block remain unchanged

### Requirement: Reference digest recording

The pinned default block SHALL record the SHA-256 digests of the official release assets for the supported Linux architectures as reference comments adjacent to the pinned tag. The recorded digests MUST be verified against the official release assets at review time. The block SHALL NOT attempt to enforce those digests at build time, and MUST NOT present the reference digests as build-time verification.

#### Scenario: Pinned block records reference digests
- **WHEN** the pinned default block is generated
- **THEN** it contains the repository-recorded SHA-256 digests for the pinned release as reference comments
- **AND** it states that the digests are a reference and not verified at build time

#### Scenario: Digest reference is not misrepresented as enforcement
- **WHEN** the block is inspected
- **THEN** it does not claim the digests are checked at build time
- **AND** no build-time checksum gate is introduced by the pinned block

### Requirement: Evidence-backed JSON-RPC compatibility

The selected default release SHALL support the fullnode JSON-RPC methods required by the current world configuration scripts. Acceptance of a pinned release MUST include a controlled container build of the actual generated block that reports the pinned release and answers a required JSON-RPC method, including `sui_getNormalizedMoveFunction` for a known published function. An open port, a generic HTTP response, or a skipped lifecycle MUST NOT substitute for this evidence.

#### Scenario: Candidate release is accepted
- **WHEN** a candidate release is pinned as the default runtime
- **THEN** a controlled container build of the generated block installs `sui` and reports the pinned release
- **AND** a required JSON-RPC request succeeds, including `sui_getNormalizedMoveFunction` for a known published function
- **AND** the clean existing full lifecycle passes without adding a new skip pattern to hide the failure

#### Scenario: Candidate lacks the JSON-RPC service
- **WHEN** the runtime responds with HTTP 404 or a JSON-RPC method-unavailable error for a required capability
- **THEN** the candidate fails compatibility acceptance
- **AND** the error is not added to known-drift skip patterns to satisfy acceptance

### Requirement: Reviewable pin maintenance

The pin maintenance documentation SHALL identify the pinned release tag and the reference-digest provenance, explain the deliberate deviation from the repository's build-time hash rule, name the known floating suiup installer component, and require review before changing the pin. Changing the pin MUST NOT rely on a runtime fallback to a different release.

#### Scenario: Maintainer updates the compatibility pin
- **WHEN** a maintainer selects another default Sui release
- **THEN** the new release tag and its reference digests are re-verified against the official release assets
- **AND** the pin update re-runs the JSON-RPC compatibility evidence and the full lifecycle
- **AND** the previous evidence is not reused to claim the replacement was tested

#### Scenario: Document records the known deviation and floating component
- **WHEN** a reviewer opens the pin maintenance document
- **THEN** it states that the digests are reference-only and not enforced at build time
- **AND** it identifies the suiup installer as a known floating component separate from the pinned release
