## ADDED Requirements

### Requirement: Consistent Go 1.27.1 baseline

The efctl application module SHALL require Go 1.27.1 as its minimum supported build toolchain. Source-build documentation SHALL match that baseline. CI, release and CodeQL setup-go steps SHALL select the version from `go.mod` rather than independently hardcode an older version. Compatible newer developer toolchains SHALL remain permitted.

#### Scenario: Source or automated build selects Go
- **WHEN** a user follows the source-install instructions or an automated workflow selects Go
- **THEN** the documented minimum and workflow version selection agree with `go 1.27.1` in `go.mod`
- **AND** upgrading efctl's build baseline does not alter upstream Sui image contents or diagnose a startup crash

### Requirement: Toolchain-compatible security verification

Go 1.27.1 verification SHALL use a released vulnerability scanner capable of processing the selected toolchain, compiled with compatible Go source-processing packages. Tool downloads SHALL use checksum-authenticated modules and their selected versions/checksums SHALL be recorded. Temporary tools and test/build artifacts SHALL remain under repository `./tmp`; tests SHALL not mutate the real checkout or managed runtime store. Updating scanner tooling SHALL NOT add application production dependencies.

#### Scenario: An older scanner cannot parse the selected Go version
- **WHEN** a scanner fails package loading because it predates Go 1.27 language features
- **THEN** verification updates/rebuilds a compatible released scanner in repository-local scratch space and reruns scanning
- **AND** the package-loading failure is not described as a project vulnerability or successful vulnerability scan

#### Scenario: Upgraded baseline is verified
- **WHEN** the Go baseline changes
- **THEN** regression tests check documentation and CI selection, and isolated Go tests/race checks, module-integrity checks and representative cross-builds are run
- **AND** application dependency changes, untested live platforms and unresolved quality gates are explicitly documented
