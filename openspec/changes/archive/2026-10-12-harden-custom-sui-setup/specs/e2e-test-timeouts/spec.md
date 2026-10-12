## ADDED Requirements

### Requirement: Configurable E2E suite deadline
The Make E2E target SHALL default E2E_TIMEOUT to 60m and accept environment or command-line overrides. CI MUST explicitly select 30m using the same target without increasing its 35-minute job limit.

#### Scenario: Local default and override
- **WHEN** make test-e2e is invoked with no override or E2E_TIMEOUT=90m
- **THEN** go test receives respectively -timeout 60m or -timeout 90m

#### Scenario: CI runs E2E tests
- **WHEN** the E2E job runs
- **THEN** it calls the shared Make target with E2E_TIMEOUT=30m
