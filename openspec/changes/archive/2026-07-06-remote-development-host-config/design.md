## Context

`efctl` currently publishes development environment ports on localhost-oriented bindings and the dashboard displays hardcoded localhost URLs. PR #49 from `setkeh:add-host-config` introduces a `host` config value and dynamic dashboard labels, which enables remote development over SSH or LAN access. That implementation also applies the configured host to PostgreSQL, so `host: "0.0.0.0"` can expose port 5432 on all interfaces.

This change will create the implementation branch `feat/remote-development`, incorporate the host configuration work, and tighten the design so remote-facing service endpoints can be exposed without accidentally exposing PostgreSQL.

## Goals / Non-Goals

**Goals:**
- Provide a safe remote-development host configuration that defaults to localhost-only behavior.
- Allow RPC, GraphQL, and frontend ports to bind to a configured host such as `127.0.0.1`, `0.0.0.0`, or a specific interface address.
- Keep PostgreSQL bound to localhost by default, regardless of the service host setting.
- Provide an explicit opt-in for PostgreSQL host exposure when needed.
- Render dashboard URLs using display-friendly host resolution.
- Trim and validate host-related config before container creation, before values can be formatted into Docker/Podman `-p` arguments.
- Preserve existing startup timing by reverting the unrelated fixed sleep increase from the upstream branch unless it is replaced by readiness polling in a separate change.
- Cover the behavior with unit tests for config, container args, startup wiring, startup delay preservation, and dashboard output.

**Non-Goals:**
- Adding authentication, TLS, firewall management, or production hardening for exposed services.
- Changing internal container-to-container networking or service discovery.
- Reworking startup readiness beyond reverting or isolating unrelated fixed sleep changes.
- Supporting arbitrary URL schemes; exposed dashboard links remain HTTP.

## Decisions

1. **Separate service host from PostgreSQL exposure.**
   - Decision: Use the configured service host for RPC, GraphQL, and frontend port bindings, but use `127.0.0.1` for PostgreSQL unless an explicit PostgreSQL exposure flag/config is enabled.
   - Rationale: The main remote-development use case needs application endpoints, not database exposure. Database exposure has higher security impact and should require intent.
   - Alternative considered: Apply one `host` setting to all ports. Rejected because it caused accidental PostgreSQL exposure.

2. **Use explicit PostgreSQL opt-in rather than implicit inference.**
   - Decision: Add a clear config field and/or CLI flag for exposing PostgreSQL, with naming that communicates risk, such as `expose-postgres: true` or an equivalent env/flag path accepted by existing CLI patterns.
   - Rationale: Operators must make a deliberate choice before binding port 5432 beyond localhost.
   - Alternative considered: Expose PostgreSQL only when GraphQL is enabled. Rejected because GraphQL requires the database internally, but external host access is not required.

3. **Validate bind hosts during config validation.**
   - Decision: `GetHost()` and related accessors must trim whitespace, and `Validate()` must reject empty-after-trim, malformed, or unsupported host values. Accept `localhost`, valid IPv4 addresses, and valid DNS hostnames. IPv6 may be accepted only if the implementation also formats container port bindings with correct IPv6 bracket syntax; otherwise reject IPv6 with a clear error.
   - Rationale: Failing early produces clearer errors than passing malformed values into `fmt.Sprintf("%s:%d:%d/tcp", host, hostPort, containerPort)` and then receiving Docker/Podman errors.
   - Alternative considered: Let the container engine reject invalid host strings. Rejected because errors would be later and less actionable.

4. **Resolve display host independently from bind host.**
   - Decision: Keep binding semantics separate from dashboard display semantics. `127.0.0.1` displays as `localhost`; `0.0.0.0` displays as a detected non-loopback IPv4 address when available, falling back to `localhost`.
   - Rationale: `0.0.0.0` is a bind address, not a client URL. The dashboard should show a URL users can open.
   - Alternative considered: Display `0.0.0.0` directly. Rejected because it is less useful as a browser/client endpoint.

5. **Keep unrelated startup timing out of this change.**
   - Decision: Revert the accidental `time.Sleep(10 * time.Second)` to `time.Sleep(30 * time.Second)` change from the upstream branch during intake. If startup reliability needs improvement, handle it separately with readiness polling/backoff that does not penalize fast startups.
   - Rationale: A fixed 30-second sleep increases every startup path by 20 seconds and is unrelated to remote host binding.

## Risks / Trade-offs

- **Remote service exposure increases local network attack surface** → Mitigate with localhost default, explicit documentation, and no authentication claims.
- **PostgreSQL may still be intentionally exposed insecurely** → Mitigate by requiring explicit opt-in and documenting the risk in config comments/help text.
- **Host validation may reject edge-case container-engine formats** → Mitigate with focused validation tests and support for common IPv4/hostname/local cases first.
- **Network interface detection for display can pick an unexpected address** → Mitigate by treating it as display-only and allowing custom host values for deterministic output.
- **Bringing in an external branch may include unrelated changes** → Mitigate by reviewing the diff and excluding unrelated startup sleep changes unless separately justified.
