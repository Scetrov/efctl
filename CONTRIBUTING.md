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

After this change merges to the default branch, an authorized bestpractices.dev editor must open [project 13754](https://www.bestpractices.dev/projects/13754), review the proposals imported from `.bestpractices.json`, save the reviewed answers, and verify both the project JSON endpoint and the live README badge. The repository file proposes answers only; it does not change the service assessment by itself. If an answer is inaccurate, correct it in the service and submit a repository follow-up; reverting this repository change does not roll back an already saved service answer.

## Bug Reports & Feature Requests

If you encounter an issue or have a feature idea, please open an Issue on GitHub using the provided templates. Provide as much detail as possible (logs, reproduction steps, environment details) when reporting bugs.

## Code of Conduct

Please note that this project is released with a Contributor Code of Conduct. By participating in this project you agree to abide by its terms.
