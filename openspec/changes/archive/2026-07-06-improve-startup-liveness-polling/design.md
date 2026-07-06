## Context

`pkg/setup/start.go` currently starts the Sui development container, normalizes scripts, sleeps for 10 seconds, checks whether the container is still running, and then waits for the existing ready-log sentinel. PR #49 attempted to increase the fixed sleep to 30 seconds to reduce startup flakiness, but that adds an unconditional 20 seconds to every successful fast startup.

The better model separates liveness from readiness. Liveness means the container process has started and has not immediately exited. Readiness remains the existing `WaitForLogs(..., ContainerLogReadyCtx)` gate.

## Goals / Non-Goals

**Goals:**
- Replace the fixed pre-readiness sleep with a small grace period followed by bounded polling.
- Continue as soon as the Sui development container is observed running.
- Fail early with exit code and recent logs if the container exits before readiness waiting begins.
- Preserve the existing log-based readiness wait and startup timeout behavior.
- Make the behavior testable without slowing tests.

**Non-Goals:**
- Proving RPC, faucet, gas, or GraphQL readiness.
- Changing `ContainerLogReadyCtx` semantics.
- Adding new container engine dependencies or external health probes.
- Changing frontend or PostgreSQL startup behavior unless the same helper is later reused intentionally.

## Decisions

1. **Use liveness polling before existing readiness detection.**
   - Decision: Replace the fixed 10-second sleep/check block with a short grace period and a bounded liveness polling loop that checks whether `ContainerRunning` is true or whether the container has exited.
   - Rationale: The container can proceed immediately on fast machines while still catching immediate failures with useful diagnostics.
   - Alternative considered: Increase the fixed sleep to 30 seconds. Rejected because it penalizes all successful startups.

2. **Keep readiness based on logs.**
   - Decision: After liveness is established, keep calling `WaitForLogs` with `ContainerLogReadyCtx` and the existing `startupTimeoutFromEnv()` context.
   - Rationale: Running is not ready. The existing log sentinel remains the semantic readiness gate for Sui dev startup.
   - Alternative considered: Replace log readiness with RPC/gas probing. Rejected for this change because the original scope is avoiding fixed delay, not redefining full Sui readiness.

3. **Use a small grace period plus polling window.**
   - Decision: Allow a small initial grace period, such as 500ms to 1s, before polling. Poll at a short interval, such as 250ms to 1s, up to the existing 10-second liveness budget.
   - Rationale: This accounts for Docker/Podman state propagation without imposing a long fixed delay.
   - Alternative considered: Poll immediately with no grace period. Rejected because a tiny grace period is a low-cost hedge against container-engine timing quirks.

4. **Make timing injectable or helper-scoped for tests.**
   - Decision: Isolate liveness waiting in a helper whose timeout/interval can be tested without real sleeps, either by passing durations or by using a minimal test-controlled polling path.
   - Rationale: Tests should verify fast-path, exit, and timeout behavior without adding wall-clock delays.
   - Alternative considered: Test only through `StartEnvironment`. Rejected because full startup tests would be slow and brittle.

## Risks / Trade-offs

- **Polling may proceed earlier than the old fixed sleep and expose latent assumptions** → Mitigate by preserving `WaitForLogs` immediately after liveness.
- **Container engine state may briefly report not running even though startup is progressing** → Mitigate with the bounded polling window and small grace period.
- **Failure diagnostics could change** → Mitigate by preserving exit code and recent log collection when liveness fails.
- **Overly broad readiness claims** → Mitigate by naming this liveness polling, not full readiness probing.
