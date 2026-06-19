# efctl Threat Model

## Scope

This document captures the initial security assumptions for `efctl`, its release workflow, and its CI/CD supply chain. It is intended to be reviewed whenever the CLI gains new commands, new deployment targets, or new credentials.

## Assets

- Source code and Git history.
- GitHub Actions workflows and release artifacts.
- Release tags, checksums, signatures, SBOMs, and provenance attestations.
- Maintainer credentials and GitHub tokens.
- User environments where the compiled CLI is executed.

## Trust boundaries

- Pull requests and issues are untrusted input.
- GitHub-hosted runners are ephemeral execution environments and should receive only the minimum token permissions needed for each job.
- Release jobs cross a higher-risk boundary because they can publish artifacts and attestations.
- Third-party GitHub Actions and Go modules are external supply-chain dependencies and should remain pinned/reviewed.

## Primary threats

- Malicious or compromised dependency introduced through `go.mod` or `go.sum`.
- Compromised GitHub Action or mutable action reference in CI/CD.
- Over-broad `GITHUB_TOKEN` permissions in workflows.
- Unauthorized release or tampered release artifact.
- Accidental credential exposure in workflow files, logs, configuration, or tests.
- AI-generated changes that bypass human review or dependency/security checks.

## Current controls

- GitHub Actions are pinned to full-length commit SHAs.
- Release workflow produces build provenance, SBOM, signatures, and checksums.
- CodeQL runs on pushes, pull requests, and a schedule.
- CI uses least-privilege default `contents: read` permissions.

## Recommended repository controls

- Require pull requests for the default branch.
- Require status checks for CI, CodeQL, dependency review, and Scorecard where appropriate.
- Require CODEOWNERS review for workflow and dependency changes.
- Restrict force pushes and branch deletion on protected branches.
- Enable secret scanning and push protection.
- Review release tags and publishing permissions periodically.

## Review cadence

Review this threat model at least quarterly and whenever release automation, credential usage, or dependency management materially changes.
