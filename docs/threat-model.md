# Threat Model

This document describes the high-level threats and trust boundaries for `efctl`. It is a living artifact and should be reviewed as the project evolves.

## Assets

- `efctl` command-line tool and its release binaries.
- User credentials used to interact with EVE Frontier and other upstream services.
- Repository source code, build configuration, and release artifacts.
- GitHub Actions workflow credentials and tokens.

## Trust Boundaries

- Developer workstations and the GitHub repository.
- GitHub Actions runner execution environment.
- Downstream consumers of published `efctl` binaries and releases.
- External APIs and package registries consumed at build time.

## Identified Threats

| Ref | Threat | Mitigation |
| --- | ------ | ---------- |
| T1 | Compromise of a maintainer account leading to malicious commits or releases. | Require pull request review, branch protection, signed commits, and CODEOWNERS review for sensitive paths. |
| T2 | Supply-chain compromise of a third-party action or dependency. | Pin GitHub Actions to immutable commit SHAs, use lockfiles, and enable dependency review. |
| T3 | Leakage of repository or cloud credentials in source or CI logs. | Enable secret scanning and push protection, avoid long-lived secrets, and prefer OIDC. |
| T4 | Injection of untrusted input into CI scripts. | Avoid interpolating attacker-controlled context values directly into shell commands. |
| T5 | Tampering with release artifacts after build. | Generate artifact attestations, SBOMs, and checksums for every release. |

## Dependencies

- Go standard library and third-party Go modules listed in `go.mod`/`go.sum`.
- GitHub Actions used in `.github/workflows`.
- Container tooling invoked by e2e tests.

## Out of Scope

- Security of the upstream EVE Frontier services themselves.
- Compromise of end-user systems after a legitimate binary has been installed.

## Review Cadence

This threat model should be reviewed at least quarterly or when the architecture, trust boundaries, or dependency surface changes significantly.
