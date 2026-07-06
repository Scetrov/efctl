## Why

Developers running `efctl` over SSH or from another machine need environment endpoints to bind beyond localhost and for the dashboard to show reachable URLs. The current upstream PR for host configuration provides that flexibility, but it can unintentionally expose PostgreSQL when binding services to all interfaces.

## What Changes

- Create a working branch named `feat/remote-development` from the repository base.
- Bring in the remote-development host configuration changes from `setkeh:add-host-config`.
- Add a configurable host/bind address for externally reachable development services, defaulting to localhost-compatible behavior.
- Update dashboard endpoint labels to display URLs based on the configured host, including resolving `0.0.0.0` to a useful local network address for display.
- Restrict PostgreSQL host exposure by default even when service endpoints are bound to `0.0.0.0` or a custom address.
- Add an explicit opt-in flag/configuration path for exposing PostgreSQL when a developer intentionally needs remote database access.
- Trim and validate host-related configuration so invalid bind addresses fail early with actionable errors before they reach container port argument formatting.
- Reject empty-after-trim, malformed, or unsupported host values; support `localhost`, valid IPv4 addresses, valid hostnames, and IPv6 only if implemented with correct container port syntax.
- Revert the unrelated `10s` to `30s` startup sleep from `setkeh:add-host-config`, unless replacing it with readiness polling/backoff in a separate justified change.
- Add or update tests for default local-only behavior, remote service binding, PostgreSQL restriction, explicit PostgreSQL exposure, host validation, startup delay preservation, and dashboard URL rendering.

## Capabilities

### New Capabilities
- `remote-development-host-config`: Configurable remote-development service binding and dashboard endpoint display with safe PostgreSQL exposure controls.

### Modified Capabilities

## Impact

- Affected code: configuration loading/validation, config template/default YAML, container port binding, service container constructors, environment startup, dashboard environment panel, startup timing preservation, and tests.
- Affected CLI behavior: `efctl env up` and `efctl env dash` should continue to default to localhost-safe behavior while allowing explicit remote service access.
- Security impact: reduces accidental database exposure by keeping PostgreSQL local-only unless explicitly enabled.
- Dependencies: no new third-party dependencies expected.
