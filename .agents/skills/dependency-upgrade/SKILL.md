---
name: dependency-upgrade
description: Manage major dependency version upgrades with compatibility analysis, staged rollout, and comprehensive testing. Use when upgrading Go modules, GitHub Actions, or managing breaking changes in libraries.
---

# Dependency Upgrade

Master major dependency version upgrades, compatibility analysis, staged upgrade strategies, and comprehensive testing approaches for the `efctl` Go ecosystem and CI/CD pipelines.

## When to Use This Skill

- Upgrading major Go module versions (e.g., v1 to v2)
- Updating security-vulnerable dependencies using `govulncheck`
- Bumping GitHub Actions versions in workflows
- Resolving Go module conflicts or retracted versions
- Planning incremental upgrade paths
- Automating dependency updates via Dependabot

## Semantic Versioning Review

```text
MAJOR.MINOR.PATCH (e.g., v2.3.1)

MAJOR: Breaking changes (in Go, requires changing the import path to /v2)
MINOR: New features, backward compatible
PATCH: Bug fixes, backward compatible

Go Modules resolve to the highest applicable minor/patch version within the required major version.
```

## Dependency Analysis

### Audit Dependencies

```bash
# Check for available minor/patch module updates
go list -m -u all

# Check for known vulnerabilities in dependencies
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

### Analyze Dependency Tree

```bash
# See why a package is installed
go mod why <module-name>

# Visualize dependency graph
go mod graph

# Clean up unused dependencies and verify the tree
go mod tidy
go mod verify
```

## Staged Upgrade Strategy

### Phase 1: Planning

```bash
# 1. Identify current versions
go list -m all

# 2. Check for breaking changes
# Review GitHub releases and CHANGELOG.md of the target modules

# 3. Create an upgrade plan
echo "Upgrade order:
1. Core utilities
2. SUI / SDK dependencies
3. GraphQL clients
4. Test dependencies
5. GitHub Actions" > UPGRADE_PLAN.md
```

### Phase 2: Incremental Updates

```bash
# Don't upgrade everything at once!

# Step 1: Update a specific module
go get [github.com/example/module@v1.2.3](https://github.com/example/module@v1.2.3)
go mod tidy

# Test
go test ./...
go build ./cmd/...

# Step 2: Update GitHub Actions (Manual or via dependabot PRs)
# Edit .github/workflows/*.yml (e.g., actions/checkout@v3 -> actions/checkout@v4)

# Step 3: Continue with other packages
```

## Breaking Change Handling

### Identifying Breaking Changes

```bash
# In Go, major version bumps (v2+) require import path changes
# Example: [github.com/example/module](https://github.com/example/module) -> [github.com/example/module/v2](https://github.com/example/module/v2)
```

### Custom Migration / Refactoring

```bash
# For standard library or widespread API changes, use `gofmt` or `go fix`
go fix ./...

# Use string replacement tools (like sed) to update major version imports globally
find . -name '*.go' -exec sed -i '' 's|"[github.com/example/module](https://github.com/example/module)"|"[github.com/example/module/v2](https://github.com/example/module/v2)"|g' {} +
```

## Testing Strategy

The `efctl` repository separates tests into standard package tests, integration tests, and E2E tests.

### Unit Tests

```bash
# Run all standard unit tests to ensure backward compatibility
go test -short ./...

# Run tests with race detector enabled
go test -race ./...
```

### Integration Tests

```bash
# Run specific integration tests after an upgrade
go test ./tests/integration/... -v
```

### E2E Tests

```bash
# Run end-to-end tests to verify overall CLI functionality
go test ./tests/e2e/... -v
```

## Automated Dependency Updates

### Dependabot Configuration

The repository uses Dependabot to manage automated upgrades for both Go modules and GitHub Actions using grouping to reduce PR noise:

```yaml
# .github/dependabot.yml
version: 2
updates:
  - package-ecosystem: gomod
    directory: /
    schedule:
      interval: daily
    groups:
      all-go-deps:
        patterns:
          - "*"

  - package-ecosystem: github-actions
    directory: /
    schedule:
      interval: weekly
    groups:
      all-actions:
        patterns:
          - "*"
```

## Rollback Plan

```bash
#!/bin/bash
# rollback.sh

# Save current state
git stash
git checkout -b upgrade-branch

# Attempt upgrade
go get [github.com/example/pkg@latest](https://github.com/example/pkg@latest)
go mod tidy

# Run tests
if go test ./...; then
  echo "Upgrade successful"
  git add go.mod go.sum
  git commit -m "chore: upgrade dependencies"
else
  echo "Upgrade failed, rolling back"
  git checkout main
  git branch -D upgrade-branch
  # Restore clean go.mod and go.sum state
  git checkout go.mod go.sum
  go mod tidy
fi
```

## Common Upgrade Patterns

### Update All Minor/Patch Versions

```bash
# Safely upgrade all direct and indirect dependencies to their latest minor/patch versions
go get -u ./...
go mod tidy
```

### Workspace Upgrades (if using Go Workspaces)

```bash
# If using go.work, ensure you sync the workspace
go work sync
```

### Fixing Checksum Mismatches

```bash
# If go.sum gets out of sync or complains about checksums
go clean -modcache
go mod tidy
```
