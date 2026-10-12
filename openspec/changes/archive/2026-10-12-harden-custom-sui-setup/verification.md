# Verification

## Reproduction and fixes

Regression tests first reproduced destructive missing-WORKDIR preparation, accepted LFS pointers, comment/CRLF/indentation-sensitive installer removal, duplicate custom instructions, missing report-writer warnings, host re-probing, and hard-coded Make/CI timeout behavior. All now pass. The setup transformation rejects unsupported/multi-stage layouts, invalid regular-file inputs and symlinks before changing the Dockerfile/entrypoint, and environment startup stops before image deletion/network/build on failure.

## Successful checks

- `go test -count=1 ./...` on the project's Go 1.27.1 baseline.
- Unit suite, integration suite, all-package race tests, build and vet using Go 1.27.2.
- E2E version, invalid-command and deadline-helper smoke tests; the missing-runtime test skips because container tools are installed.
- `make fmt-check`, including formatting fixes to tagged E2E and an already-unformatted integration test.
- Offline Podman image builds and isolated executions of both custom and fallback setup, using harmless shell fixtures and a cached image selected by full immutable ID. Changed installer comments are included. No real external Sui artifact is downloaded or executed by this test.
- Every configured pre-commit hook: documentation generation, Go formatting/vet/build/tests, gosec, govulncheck, gocyclo, actionlint, action SHA pinning, and Gitleaks.
- Strict validation of the OpenSpec change and the four synchronized capability specifications.

Archival also repaired the existing container-patch-diagnostics main specification's legacy delta header into a proper main-spec Requirements section, allowing its unchanged existing requirements and this delta to be parsed and synchronized.

The all-hook run used an exact staged patch applied to a clean shared local checkout under repository `tmp/`, so ignored historical scratch source/dependency trees were not mistaken for project code by recursive scanners. Scratch files are isolated as a separate ignored Go module locally. The no-config unit test now uses a unique absent filename to avoid assumptions about ancestor files when TMPDIR is inside the checkout. The Make formatting gate selects tracked and non-ignored untracked Go source instead of scanning scratch/dependency caches.

## Scanner/toolchain provenance

Product Go module dependencies and minimum baseline remain unchanged. Validation uses a checksum-verified downloaded Go 1.27.2 toolchain because Go 1.27.1 has pre-existing standard-library vulnerability findings. Govulncheck reports no vulnerabilities with the patched toolchain; this does not claim that builds using Go 1.27.1 are free of its existing findings.

Local validation tooling: gosec v2.29.0 and govulncheck v1.9.0, queried as current releases and downloaded through Go module checksum verification. Gosec was rebuilt in an ignored standalone tooling module with golang.org/x/tools v0.52.0 to support Go 1.27 export data. No scanner checks were disabled and no changes to security hooks were made. Source-module checksums and tool build metadata are retained in `tmp/followup/`.

## Limitations

The full blockchain/world-deployment E2E lifecycle and native Sui ABI compatibility on older x86/physical ARM hardware were not run. Container-selection tests establish that the real patched instructions choose the intended executable, not that an arbitrary user-supplied binary is trusted or compatible. CPU hints are conditional architecture-scoped observations, not diagnoses. Git LFS checks identify pointers but do not authenticate executable provenance.

## Publication

Archive and sync the specifications before the single signed conventional commit, authored using the repository's configured Scetrov identity, with model and Pi Coding Agent attribution. Open a follow-up PR against main referencing #94.
