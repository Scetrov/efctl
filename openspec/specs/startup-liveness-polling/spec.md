# startup-liveness-polling Specification

## Requirements

### Requirement: Sui startup liveness polling
The system SHALL use bounded liveness polling before waiting for Sui development container readiness.

#### Scenario: Fast liveness success
- **WHEN** the Sui development container is observed running during the liveness polling window
- **THEN** startup proceeds to the existing log-based readiness wait without waiting for the full liveness window

#### Scenario: Initial grace period
- **WHEN** the Sui development container has just been started
- **THEN** the system waits only a short grace period before beginning liveness polling

#### Scenario: Container exits during liveness polling
- **WHEN** the Sui development container exits before it is observed running
- **THEN** startup fails with the container exit code and recent container logs

#### Scenario: Liveness timeout
- **WHEN** the Sui development container is not observed running before the liveness polling window expires
- **THEN** startup fails with diagnostics that include recent container logs and exit-code information when available

### Requirement: Log-based readiness preserved
The system SHALL preserve the existing log-based Sui readiness gate after liveness is established.

#### Scenario: Liveness succeeds before readiness
- **WHEN** the Sui development container is observed running
- **THEN** the system continues to wait for the existing ready-log sentinel before treating Sui startup as ready

#### Scenario: Ready-log timeout
- **WHEN** the ready-log sentinel is not observed before the configured startup timeout
- **THEN** startup fails using the existing readiness timeout diagnostics

### Requirement: No unconditional extended startup delay
The system SHALL NOT add an unconditional extended fixed delay before Sui readiness waiting.

#### Scenario: Avoid fixed 30-second delay
- **WHEN** the Sui development container is observed running quickly
- **THEN** startup does not wait an unconditional 30 seconds before starting the ready-log wait

#### Scenario: Preserve fast startup path
- **WHEN** liveness is established quickly and the ready-log sentinel appears quickly
- **THEN** startup completes without waiting for any unused liveness budget
