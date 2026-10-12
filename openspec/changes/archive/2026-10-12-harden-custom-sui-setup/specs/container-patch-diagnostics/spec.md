## MODIFIED Requirements

### Requirement: Warn when a required container patch target is absent

When Docker environment preparation attempts a required text patch, the system SHALL emit a console warning if neither a supported source pattern nor an already-patched marker is present. For mandatory Sui preparation, the system SHALL also return an error before modifying scaffold files or building. Other unmatched patches remain non-fatal. The warning SHALL identify the semantic patch operation and affected scaffold file without including file contents or environment values.

#### Scenario: Literal replacement target is missing
- **WHEN** a required literal patch is applied to scaffold content that contains neither the expected source text nor the replacement marker
- **THEN** the system emits a warning identifying the unapplied patch and target file
- **AND** the system leaves that content unchanged and continues processing remaining patches unless mandatory Sui preparation is unsupported

#### Scenario: Regular-expression replacement target is missing
- **WHEN** a required regular-expression patch matches no supported source form and its replacement marker is absent
- **THEN** the system emits a warning identifying the unapplied patch and target file

#### Scenario: Compound patch target is missing
- **WHEN** multiple raw replacements implement one required semantic patch and none of their supported source forms or replacement markers are present
- **THEN** the system emits no more than one warning for that semantic patch attempt
