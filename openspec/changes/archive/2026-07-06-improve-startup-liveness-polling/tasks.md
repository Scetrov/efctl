## 1. Startup Flow Analysis

- [x] 1.1 Review `pkg/setup/start.go` around Sui development container startup, fixed sleep, running check, and ready-log wait.
- [x] 1.2 Review existing setup/container mocks to identify the smallest test seam for liveness polling.
- [x] 1.3 Confirm existing `WaitForLogs(..., ContainerLogReadyCtx)` readiness behavior remains unchanged.

## 2. Liveness Polling Implementation

- [x] 2.1 Introduce a helper for bounded Sui container liveness polling with configurable grace period, poll interval, and timeout.
- [x] 2.2 Make the helper proceed as soon as `ContainerRunning(container.ContainerSuiPlayground)` returns true.
- [x] 2.3 Make the helper fail with exit code and recent logs when the container exits or fails to become running within the liveness window.
- [x] 2.4 Replace the fixed pre-readiness sleep/running-check block in `startSuiDev` with the liveness polling helper.
- [x] 2.5 Preserve the existing log-based readiness wait and startup timeout after liveness succeeds.

## 3. Tests

- [x] 3.1 Add unit tests for fast-path liveness success without waiting for the full liveness timeout.
- [x] 3.2 Add unit tests for container exit during liveness polling, including exit-code/log diagnostics.
- [x] 3.3 Add unit tests for liveness timeout diagnostics when the container never becomes running.
- [x] 3.4 Add tests proving `WaitForLogs` is still called after liveness succeeds.
- [x] 3.5 Ensure tests avoid real multi-second sleeps by using small durations or a testable polling seam.

## 4. Verification

- [x] 4.1 Run `go test ./...` and fix failures.
- [x] 4.2 Run formatting and repository pre-commit checks.
- [x] 4.3 Manually inspect startup code to verify no unconditional 30-second delay remains.
- [x] 4.4 Confirm diagnostics still include recent logs and exit-code information for early Sui container failures.
