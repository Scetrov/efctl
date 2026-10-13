## ADDED Requirements

### Requirement: Prominent CPU warning severity indicator

Doctor SHALL render local and runtime CPU compatibility hints with the shared UI warning badge through the supplied report writer. The badge MUST retain a readable `WARNING` severity label when color is disabled, use the shared warning styling when color is enabled, and separate the warning from adjacent diagnostic rows. Existing evidence gating, local/runtime labels, conditional guidance, and no-probe rendering behavior MUST remain unchanged.

#### Scenario: Missing CPU features with color enabled
- **WHEN** collected AMD64 CPU evidence is missing ADX or BMI2 and color is enabled
- **THEN** the supplied report writer receives the shared styled warning badge and the corresponding local or runtime conditional hint

#### Scenario: Missing CPU features without color
- **WHEN** collected AMD64 CPU evidence is missing ADX or BMI2 and color is disabled
- **THEN** the supplied report writer receives a readable `WARNING` badge and separated warning text without ANSI color sequences

#### Scenario: No CPU warning is warranted
- **WHEN** collected CPU evidence is compatible, non-AMD64, unknown, or incomplete
- **THEN** no CPU warning badge is rendered for that boundary
