## 1. Reproduce and harden setup

- [x] 1.1 Add failing regression tests for preservation, LFS inputs, realistic installation blocks and idempotence.
- [x] 1.2 Implement validated in-memory Sui transformation and propagate preparation errors.

## 2. Diagnostics and test configuration

- [x] 2.1 Add architecture/evidence/writer regression tests and remove live gathering during rendering.
- [x] 2.2 Configure E2E_TIMEOUT with 60m local default and explicit 30m CI override; add configuration tests.
- [x] 2.3 Correct artifact startup command and document invalid-input diagnostics and timeout usage.

## 3. Validation and publication

- [x] 3.1 Run unit, integration, race, isolated container regressions, smoke and pre-commit gates; record limitations.
- [x] 3.2 Sync delta specifications and archive the completed change before the signed commit and PR.
