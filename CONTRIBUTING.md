# Contributing to `efctl`

First off, thank you for considering contributing to `efctl`! It's people like you that make this tool such a great utility.

## Development Setup

1. **Fork the repo** and clone it locally.
2. Ensure you have **Go 1.20+** installed.
3. Install development tools:
   - We use `gofmt` and `goimports` for code formatting.
   - Run `go test ./...` to ensure everything works before making changes.

## Branching & Pull Requests

1. Create a branch for your feature or bug fix:

   ```bash
   git checkout -b feature/my-new-feature
   ```

2. Commit your changes. Write concise, descriptive commit messages.
3. Run formatting and tests:

   ```bash
   gofmt -w .
   go test ./...
   ```

4. Push your branch and open a Pull Request.
5. Please use the provided Pull Request template and fill it out completely.

## Required Checks and Tests

New functionality and bug fixes **must include automated tests** at the appropriate level: unit tests for isolated logic, integration tests for component interactions, and E2E tests for user workflows. Run the applicable checks before requesting review:

```bash
make fmt-check       # formatting
make vet             # compiler and static-analysis warnings
go test ./...        # unit tests, including repository invariants
make test-integration # integration tests
make test-e2e        # Docker-backed end-to-end tests
make security        # gosec and govulncheck security checks
pre-commit run --all-files # complete local quality gate
```

All applicable checks must pass. Regenerate CLI documentation with `make docs` when a command or its output changes.

## Release Notes

For every user-visible change or security fix, add a concise human-readable entry under the `Unreleased` section in [CHANGELOG.md](CHANGELOG.md). Do not use raw commit logs as release notes. If the change fixes a publicly known efctl runtime vulnerability (for example, one with a CVE), identify it in that entry.

## Best Practices Badge Owner Workflow

[`.bestpractices.json`](.bestpractices.json) is a version-controlled proposal, not the live assessment. After an attestation change merges, an authorized bestpractices.dev editor must:

1. Open [project 13754](https://www.bestpractices.dev/projects/13754) and compare every Passing control with the proposal and the current Passing criteria.
2. Record the audit in [`docs/best-practices-reconciliation.md`](docs/best-practices-reconciliation.md), including the audit date, merged commit, project URL, score, badge state, affected control, proposed and live values, classification (`pending owner acceptance`, `unsupported repository evidence`, or `service-side discrepancy`), and required follow-up. Do not record credentials, private report details, or maintainer identities.
3. Review and save only accurate public-evidence proposals. Do not save a `Met` answer that needs an unconfirmed private fact; retain `N/A` where allowed or record the control as unresolved.
4. Wait **24 hours**, then check the project JSON endpoint and the live README badge. Record the resulting score and every remaining control. If the state did not propagate, classify it as a service-side discrepancy and inspect or correct the project in the service.

For unsupported repository evidence, open a bounded follow-up that names the control, missing evidence or enforcement, validation required, and owner. Do not change the manifest speculatively. If a relevant change modifies `.bestpractices.json`, `SECURITY.md`, the release process, a cited quality gate, or the service's Passing criteria, repeat the audit after it merges.

To roll back an incorrect repository claim, revert the repository change and update the reconciliation record. Repository rollback does not undo an answer already saved in bestpractices.dev: an authorized editor must correct that answer in the service, wait the same 24-hour interval, and re-verify the endpoint and badge.

## Bug Reports & Feature Requests

If you encounter an issue or have a feature idea, please open an Issue on GitHub using the provided templates. Provide as much detail as possible (logs, reproduction steps, environment details) when reporting bugs.

## Code of Conduct

Please note that this project is released with a Contributor Code of Conduct. By participating in this project you agree to abide by its terms.
