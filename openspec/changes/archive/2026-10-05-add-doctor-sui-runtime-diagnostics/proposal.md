## Why

An end-user reports `sui start` terminating with `Illegal instruction` during `efctl env up`. The current doctor report identifies efctl, the OS, and the container CLI, but lacks the Sui binary version, CPU capabilities, and runtime/image architecture needed to distinguish binary defects, CPU feature masking, and possible emulation. Collecting this evidence in one command reduces repeated support requests without claiming an unverified root cause.

## What Changes

- Extend the default `efctl doctor` report with CPU model, machine architecture, and available CPU instruction flags, distinguishing these from efctl's build platform.
- Report container client/server versions and runtime server OS/architecture, including partial failures and remote/VM boundaries where observable.
- Report the actual Sui container's state, exit code, image identity/digests, image OS/architecture, and explicitly sourced Sui version.
- Collect Sui version, machine architecture, and CPU flags from a running managed container; when it is not running, use a short-lived, isolated probe of its already-local image without starting the managed environment, pulling images, or accessing its volumes.
- Distinguish host Sui from container/image Sui, and preserve host Sui client configuration guards.
- Show bounded, sanitized diagnostic failures and availability reasons; report architecture mismatch as possible emulation rather than proof of the crash's cause.
- Document the new fields, probe behavior, limitations, and support workflow; cover success and failure paths with tests.
- Raise the efctl project minimum Go version to 1.27.1, aligning source-install documentation and CI/release selection through `go.mod`.
- Verify compatible released vulnerability-scanning tooling built with Go 1.27.1 in repository-local scratch space; keep application dependencies unchanged unless verification demonstrates a necessary change.

## Capabilities

### New Capabilities

- `doctor-runtime-diagnostics`: Safe, bounded collection and presentation of host CPU, runtime server, Sui image, and Sui executable diagnostics for startup failure investigation.
- `go-toolchain-baseline`: Consistent Go 1.27.1 source-build, CI/release and security-tool verification requirements.

### Modified Capabilities

None. Existing `sui-config-guard` requirements remain in force; version-only probes do not invoke client commands or generate keys.

## Impact

- `pkg/doctor/doctor.go` and its tests: report types, gatherers, platform parsing, availability/error handling, and injectable command execution.
- `cmd/doctor.go` and command rendering tests: additive human-readable diagnostic fields and help text.
- `pkg/container/container.go` and tests as needed: bounded, structured inspect/probe access supporting Docker and Podman, without broad lifecycle changes.
- `docs/efctl_doctor.md` and relevant support documentation: regenerated CLI documentation and safe report-sharing guidance.
- `go.mod`, `README.md`, toolchain regression tests, isolated test tooling and verification notes: Go 1.27.1 baseline and compatible security-scanner verification. Existing workflows already select the toolchain through `go-version-file: go.mod`.
- No new production dependency is planned; use Go's standard library and existing container runtime executables. No Sui upgrade, image pinning change, startup behavior fix, or diagnosis of a mandatory AVX requirement is included.
