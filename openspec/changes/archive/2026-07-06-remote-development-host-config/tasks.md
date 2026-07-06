## 1. Branch and Upstream Change Intake

- [x] 1.1 Create or switch to repository branch `feat/remote-development` from the current base branch.
- [x] 1.2 Fetch `setkeh:add-host-config` and inspect the diff against the base branch.
- [x] 1.3 Incorporate the host configuration, dashboard URL, container binding, and test changes from `setkeh:add-host-config`.
- [x] 1.4 Exclude or revert the unrelated `time.Sleep(10 * time.Second)` to `time.Sleep(30 * time.Second)` startup delay change from the upstream branch.

## 2. Configuration Model and Validation

- [x] 2.1 Add service host configuration with default `127.0.0.1` in the config model and template YAML.
- [x] 2.2 Add explicit PostgreSQL exposure configuration or flag that defaults to disabled.
- [x] 2.3 Trim service host and PostgreSQL exposure host values in config accessors before use.
- [x] 2.4 Validate service host and PostgreSQL exposure host values during config validation, accepting `localhost`, valid IPv4 addresses, and valid hostnames.
- [x] 2.5 Reject empty-after-trim, malformed, unsupported, and IPv6 values unless IPv6 port formatting is explicitly implemented.
- [x] 2.6 Add config tests for default host, trimmed host, remote host, custom hostname, explicit PostgreSQL exposure, invalid host values, and unsupported IPv6 behavior.

## 3. Container Binding Behavior

- [x] 3.1 Thread service host configuration into RPC, GraphQL, and frontend container port bindings.
- [x] 3.2 Keep PostgreSQL host binding on `127.0.0.1` when PostgreSQL exposure is not enabled.
- [x] 3.3 Bind PostgreSQL to the configured remote host only when explicit exposure is enabled.
- [x] 3.4 Add container argument tests covering service ports, default PostgreSQL restriction, and explicit PostgreSQL exposure.

## 4. Dashboard Endpoint Display

- [x] 4.1 Add display-host resolution that renders `127.0.0.1` as `localhost`.
- [x] 4.2 Render `0.0.0.0` dashboard URLs using a detected non-loopback IPv4 address with a localhost fallback.
- [x] 4.3 Render custom host dashboard URLs without replacing the configured custom host.
- [x] 4.4 Add dashboard tests for RPC, GraphQL, frontend, disabled optional services, and display-host fallback behavior.

## 5. Environment Startup Wiring

- [x] 5.1 Pass validated service host and PostgreSQL exposure settings through environment startup to service constructors.
- [x] 5.2 Ensure internal container networking and internal RPC fetch paths continue to use existing local/container network assumptions.
- [x] 5.3 Add setup tests or mock assertions proving PostgreSQL remains local-only unless exposure is enabled.

## 6. Verification

- [x] 6.1 Run `go test ./...` and fix failures.
- [x] 6.2 Run formatting and pre-commit checks required by the repository.
- [x] 6.3 Verify the Sui startup fixed sleep remains at the original 10 seconds unless replaced by separately scoped readiness polling.
- [x] 6.4 Manually review generated container port args for localhost default, `0.0.0.0` service exposure, explicit PostgreSQL exposure, and rejected invalid hosts.
- [x] 6.5 Document any user-facing config comments or help text warning that remote service exposure increases local network attack surface.
