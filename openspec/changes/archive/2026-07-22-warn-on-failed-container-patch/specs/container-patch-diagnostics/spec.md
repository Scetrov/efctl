## ADDED Requirements

### Requirement: Warn when a required container patch target is absent

When Docker environment preparation attempts a required text patch, the system SHALL emit a console warning if neither a supported source pattern nor an already-patched marker is present. The warning SHALL identify the semantic patch operation and affected scaffold file without including file contents or environment values.

#### Scenario: Literal replacement target is missing
- **WHEN** a required literal patch is applied to scaffold content that contains neither the expected source text nor the replacement marker
- **THEN** the system emits a warning identifying the unapplied patch and target file
- **AND** the system leaves that content unchanged and continues processing remaining patches

#### Scenario: Regular-expression replacement target is missing
- **WHEN** a required regular-expression patch matches no supported source form and its replacement marker is absent
- **THEN** the system emits a warning identifying the unapplied patch and target file

#### Scenario: Compound patch target is missing
- **WHEN** multiple raw replacements implement one required semantic patch and none of their supported source forms or replacement markers are present
- **THEN** the system emits no more than one warning for that semantic patch attempt

### Requirement: Successful and idempotent patches remain warning-free

The system MUST distinguish an unmatched required patch from a successful or already-applied patch. It SHALL NOT emit a missing-target warning when the source match is replaced successfully or when the intended result is already present.

#### Scenario: Replacement succeeds
- **WHEN** a required patch finds a supported source pattern
- **THEN** the system applies the replacement and emits no missing-target warning for that patch

#### Scenario: Patch is already applied
- **WHEN** the content already contains the marker produced by a required patch
- **THEN** the system leaves the content unchanged and emits no missing-target warning for that patch

#### Scenario: Docker preparation is rerun
- **WHEN** Docker environment preparation runs against files previously patched by the system
- **THEN** the files remain unchanged and no missing-target warnings are emitted for those already-applied patches

### Requirement: Patch diagnostics use the standard CLI warning channel

The system SHALL publish missing-target diagnostics through the existing standard console warning facility so they are visible during the normal environment setup flow.

#### Scenario: Environment setup encounters an unmatched patch
- **WHEN** `efctl env up` prepares Docker scaffold files and a required patch cannot be applied
- **THEN** the diagnostic appears in the command's console output with the standard warning presentation
