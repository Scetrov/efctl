## 1. Regression Tests

- [x] 1.1 Add failing unit coverage showing Dockerfile preparation inserts exactly one `libatomic1` entry into the existing `--no-install-recommends` package list and retains apt-list cleanup
- [x] 1.2 Add unit coverage for an already-patched Dockerfile and for a missing insertion target using the standard warning channel
- [x] 1.3 Add or extend deployment-command tests for the `sui-playground` identity, OS/architecture, pnpm launcher, project `packageManager`, pnpm version, workspace configuration, and preserved failure output
- [x] 1.4 Add a Docker smoke test that builds the patched Sui image and runs pnpm from a project selecting `pnpm@11.9.0`, reproducing the pre-fix loader failure

## 2. Image and Diagnostic Fix

- [x] 2.1 Extend `patchDockerfile` to add only `libatomic1` through the existing apt package layer, preserving idempotency, `--no-install-recommends`, cleanup, and unmatched-target diagnostics
- [x] 2.2 Extend the pre-deployment diagnostic command to identify `sui-playground`, report available OS/architecture and pnpm selection context, and keep nonessential probes non-blocking
- [x] 2.3 Run the focused unit and Docker smoke tests and confirm `pnpm@11.9.0` starts without a `libatomic.so.1` loader error

## 3. Environment Validation

- [x] 3.1 Build a fresh candidate `efctl` and inspect the resulting `localhost/efctl-sui-dev` image to confirm `libatomic1` is installed and the affected standalone executable resolves `libatomic.so.1`
- [x] 3.2 Inspect the existing state on `scetrov@172.29.117.128`, transfer the candidate to a safe temporary location, and tear down the partially initialized environment using normal `efctl env down`
- [x] 3.3 Recreate the remote Linux/Docker environment from scratch and verify world deployment passes the project-selected pnpm step without any host package installation or running-container mutation
- [x] 3.4 Repeat diagnosis, test-first remediation, candidate transfer, clean rebuild, and remote deployment validation until the complete deployment succeeds or a critical external blocker is captured with exact evidence
- [x] 3.5 Verify the resulting environment can be torn down and recreated normally after successful deployment
- [x] 3.6 Record x86_64 evidence and run equivalent arm64 smoke validation where execution infrastructure is available; document any architecture that remains unverified

## 4. Documentation and Quality Gates

- [x] 4.1 Update the toolchain/developer documentation with the correct `sui-playground` execution boundary, pnpm dispatcher behavior, image-owned runtime dependency, and supported teardown/recreation workflow
- [x] 4.2 Run all relevant Go unit, integration, and e2e tests and record the exact validation commands and outcomes
- [x] 4.3 Run configured pre-commit hooks and resolve all failures without unrelated refactoring
- [x] 4.4 Review the final diff for minimal scope, ensure no host-install or `docker exec apt-get install` workaround is documented, and record genuine follow-up work separately
