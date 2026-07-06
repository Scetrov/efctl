## ADDED Requirements

### Requirement: Configurable service bind host
The system SHALL allow development service endpoints to bind to a configured host address while preserving localhost-only behavior by default.

#### Scenario: Default service host
- **WHEN** no service host is configured
- **THEN** RPC, GraphQL, and frontend host port bindings use `127.0.0.1`

#### Scenario: Remote service host
- **WHEN** the service host is configured as `0.0.0.0`
- **THEN** RPC, GraphQL, and frontend host port bindings use `0.0.0.0`

#### Scenario: Custom service host
- **WHEN** the service host is configured as a valid custom address
- **THEN** RPC, GraphQL, and frontend host port bindings use that custom address

### Requirement: PostgreSQL exposure restricted by default
The system SHALL keep PostgreSQL bound to localhost unless PostgreSQL exposure is explicitly enabled.

#### Scenario: Remote service host without PostgreSQL exposure
- **WHEN** the service host is configured as `0.0.0.0` and PostgreSQL exposure is not enabled
- **THEN** PostgreSQL host port binding uses `127.0.0.1`

#### Scenario: Custom service host without PostgreSQL exposure
- **WHEN** the service host is configured as a custom non-localhost address and PostgreSQL exposure is not enabled
- **THEN** PostgreSQL host port binding uses `127.0.0.1`

#### Scenario: Explicit PostgreSQL exposure
- **WHEN** PostgreSQL exposure is explicitly enabled and the service host is configured as a valid remote address
- **THEN** PostgreSQL host port binding uses the configured remote address

### Requirement: Host configuration validation
The system SHALL trim and validate host-related configuration before creating containers or formatting container port arguments.

#### Scenario: Valid host configuration
- **WHEN** the configured host values are `localhost`, valid IPv4 addresses, valid DNS hostnames, or supported IPv6 values with correct port binding syntax
- **THEN** configuration validation succeeds

#### Scenario: Whitespace host configuration
- **WHEN** a configured host value has leading or trailing whitespace around an otherwise valid value
- **THEN** host accessors return the trimmed value and configuration validation succeeds

#### Scenario: Empty host configuration
- **WHEN** a configured host value is empty after trimming
- **THEN** configuration validation fails with an actionable error before container creation

#### Scenario: Malformed host configuration
- **WHEN** a configured host value is malformed or unsupported
- **THEN** configuration validation fails with an actionable error before container creation

#### Scenario: Unsupported IPv6 host configuration
- **WHEN** a configured host value is IPv6 and the implementation does not support correct IPv6 container port syntax
- **THEN** configuration validation fails with an actionable error before container creation

### Requirement: Dashboard displays reachable endpoint URLs
The dashboard SHALL display endpoint URLs derived from the configured service host rather than hardcoded localhost values.

#### Scenario: Localhost dashboard URLs
- **WHEN** the service host is `127.0.0.1`
- **THEN** the dashboard displays RPC, GraphQL, and frontend URLs using `localhost`

#### Scenario: All-interfaces dashboard URLs
- **WHEN** the service host is `0.0.0.0` and a non-loopback IPv4 address is available
- **THEN** the dashboard displays RPC, GraphQL, and frontend URLs using the detected non-loopback IPv4 address

#### Scenario: All-interfaces dashboard fallback
- **WHEN** the service host is `0.0.0.0` and no non-loopback IPv4 address is available
- **THEN** the dashboard displays RPC, GraphQL, and frontend URLs using `localhost`

#### Scenario: Custom host dashboard URLs
- **WHEN** the service host is a valid custom host
- **THEN** the dashboard displays RPC, GraphQL, and frontend URLs using that custom host

### Requirement: Remote-development implementation branch
The implementation SHALL happen on a repository branch named `feat/remote-development` and incorporate the reviewed host-configuration changes from `setkeh:add-host-config`.

#### Scenario: Implementation branch exists
- **WHEN** implementation starts
- **THEN** the repository has a working branch named `feat/remote-development`

#### Scenario: External host configuration changes incorporated safely
- **WHEN** changes from `setkeh:add-host-config` are brought into the implementation branch
- **THEN** unrelated changes are excluded or separately justified, and PostgreSQL exposure remains restricted unless explicitly enabled

#### Scenario: Unrelated fixed startup delay excluded
- **WHEN** changes from `setkeh:add-host-config` are brought into the implementation branch
- **THEN** the unrelated `time.Sleep(10 * time.Second)` to `time.Sleep(30 * time.Second)` change is not included in this change

#### Scenario: Startup readiness improvement handled separately
- **WHEN** startup reliability needs improvement
- **THEN** it is implemented as readiness polling/backoff in a separate justified change rather than as an unconditional fixed 30-second delay
