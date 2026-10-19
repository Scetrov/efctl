## ADDED Requirements

### Requirement: Architecture-qualified CPU compatibility hints
Doctor SHALL derive compatibility hints from already-collected CPUInfo evidence and render them through the supplied report writer. Hints MUST identify the local or runtime boundary, gate ADX/BMI2 advice on confirmed AMD64 architecture and available flags, and describe a possible affected-build issue rather than establish a crash diagnosis. Rendering MUST NOT launch probes.

#### Scenario: ARM or unknown architecture
- **WHEN** CPU architecture is ARM or unavailable
- **THEN** no x86 ADX/BMI2 compatibility hint is rendered

#### Scenario: Known AMD64 flags are incomplete for affected builds
- **WHEN** collected AMD64 flags omit ADX or BMI2
- **THEN** the supplied writer receives a source-labelled conditional compatibility hint

#### Scenario: Report is captured or rendered remotely
- **WHEN** a report is rendered to a buffer with recorded local and runtime evidence
- **THEN** hints use that evidence only and do not re-probe the current host or write outside the buffer
