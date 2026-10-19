# workflow-linting Specification

## Purpose
Validate GitHub Actions workflows, enforce SHA-pinned actions, and publish workflow lint findings to Code Scanning.

## Requirements

### Requirement: Lint workflow files with actionlint
The repository SHALL lint all GitHub Actions workflow files with actionlint v1.7.12 in both the pre-commit hook set and CI, and a failing lint SHALL block the commit (pre-commit) or the CI job (on push/PR to the default branch).

#### Scenario: Pre-commit lint
- **WHEN** a maintainer commits a change under `.github/workflows/` with pre-commit installed
- **THEN** actionlint v1.7.12 runs against the workflow files and the commit is blocked if actionlint reports errors

#### Scenario: CI lint on pull request
- **WHEN** a pull request against the default branch is opened or updated
- **THEN** the CI `workflow-lint` job runs actionlint v1.7.12 and fails if any workflow file is reported in error

### Requirement: Enforce SHA-pinned actions with version comments
The repository SHALL enforce, via `scripts/check-action-pinning.sh` in pre-commit and CI, that every `uses:` reference in `.github/workflows/` is a full 40-character commit SHA pin with an adjacent `# v<version>` comment.

#### Scenario: Compliant workflow file
- **WHEN** a workflow file contains `uses: actions/checkout@<40-char-sha> # v7.0.1`
- **THEN** the pinning check passes for that reference

#### Scenario: Tag-pinned action rejected
- **WHEN** a workflow file contains `uses: actions/checkout@v4` or any reference without the `# v<version>` comment
- **THEN** the pinning check fails and lists the offending file and line

### Requirement: Publish actionlint results as SARIF
CI SHALL generate SARIF 2.1.0 output from actionlint using the vendored template `scripts/actionlint-sarif-template.txt` and upload it to GitHub Code Scanning under a category distinct from the CodeQL and OSV-Scanner categories.

#### Scenario: SARIF upload after a clean lint
- **WHEN** the `workflow-lint` job completes
- **THEN** the SARIF file is uploaded to Code Scanning under the `workflow-lint` category whether or not actionlint reported errors
