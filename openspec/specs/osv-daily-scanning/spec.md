# osv-daily-scanning Specification

## Purpose
Detect dependency vulnerabilities daily and publish OSV-Scanner results securely to GitHub Code Scanning.

## Requirements

### Requirement: Run a daily OSV dependency scan
The repository SHALL run a scheduled GitHub Actions workflow that executes `osv-scanner` at least once per day against the Go module graph of the default branch, producing SARIF output.

#### Scenario: Daily scheduled scan
- **WHEN** the daily schedule fires on the default branch
- **THEN** the workflow runs `osv-scanner` and writes a SARIF report of all reported vulnerabilities in the resolved module graph

#### Scenario: Manual rescan
- **WHEN** a maintainer triggers the workflow via `workflow_dispatch`
- **THEN** the same scan-and-upload behavior executes immediately without waiting for the schedule

### Requirement: Publish SARIF results to Code Scanning
The workflow SHALL upload the SARIF report to GitHub Code Scanning via `upload-sarif`, using a category distinct from the CodeQL alerts, so findings appear under the OSV-Scanner tool.

#### Scenario: SARIF upload after a clean scan
- **WHEN** the scan completes with no vulnerabilities
- **THEN** the SARIF is uploaded and Code Scanning records no open OSV-Scanner alerts

#### Scenario: SARIF upload when vulnerabilities are found
- **WHEN** the scan reports one or more vulnerabilities
- **THEN** the SARIF is still uploaded and GitHub Code Scanning creates or updates alerts for each reported vulnerability

### Requirement: Harden the scanning workflow
The workflow SHALL pin every action by commit SHA with a version comment, restrict the GITHUB_TOKEN to `contents: read` and `security-events: write`, and NOT retain the SARIF as a public workflow artifact.

#### Scenario: Workflow review
- **WHEN** the workflow file is reviewed
- **THEN** all `uses:` references are full 40-character SHA pins and the job declares only the `contents: read` and `security-events: write` permissions

#### Scenario: SARIF hygiene
- **WHEN** the workflow completes
- **THEN** the SARIF file exists only within the job workspace and is not published as a download artifact
