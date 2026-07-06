## Why

The Sui development container startup path uses a fixed pre-readiness sleep before checking whether the container is still running. Increasing that fixed delay improves slow-start tolerance at the cost of penalizing every fast startup, so the startup path should proceed as soon as the container is observed alive while still failing quickly if it exits.

## What Changes

- Replace the fixed pre-readiness startup delay with a short grace period followed by bounded liveness polling.
- Proceed immediately once the Sui development container is observed running.
- Fail early with exit code and recent logs if the Sui development container exits during the liveness window.
- Preserve the existing log-based readiness gate after liveness is established.
- Keep startup readiness and liveness distinct: liveness means the container process is alive; readiness remains based on the existing ready-log sentinel.
- Add tests for fast-path startup, early container exit, and liveness timeout behavior.

## Capabilities

### New Capabilities
- `startup-liveness-polling`: Bounded Sui development container liveness polling before existing log-based readiness detection.

### Modified Capabilities

## Impact

- Affected code: Sui development environment startup flow in `pkg/setup/start.go`, setup mocks/tests, and any helper functions introduced for polling behavior.
- Affected behavior: fast successful starts no longer wait an unconditional fixed delay, while failed starts still report useful diagnostics.
- Dependencies: no new third-party dependencies expected.
