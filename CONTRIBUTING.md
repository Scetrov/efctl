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

## Bug Reports & Feature Requests

If you encounter an issue or have a feature idea, please open an Issue on GitHub using the provided templates. Provide as much detail as possible (logs, reproduction steps, environment details) when reporting bugs.

## Code of Conduct

Please note that this project is released with a Contributor Code of Conduct. By participating in this project you agree to abide by its terms.
