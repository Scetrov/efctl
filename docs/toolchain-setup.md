# Toolchain Setup Guide

## Node.js Environment Configuration

This document explains the Node.js toolchain setup in efctl, particularly focusing on the build configurations for pnpm and esbuild in containerized environments.

### Problem Identified

E2E tests were failing due to pnpm v10.26+ / v11+ blocking build scripts by default. Specifically, `esbuild` was being prevented from running its build scripts during `pnpm install`, leading to:
```
[ERR_PNPM_IGNORED_BUILDS] Ignored build scripts: esbuild@0.27.2
Run "pnpm approve-builds" to pick which dependencies should be allowed to run scripts.
Deployment failed: failed to deploy world: exec error: exit status 1
```

### Solution Implemented

#### 1. pnpm-workspace.yaml Configuration

Created `pnpm-workspace.yaml` in both `builder-scaffold/` and `world-contracts/` with the following content:

```yaml
allowBuilds:
  esbuild: true
```

This configuration is the **ONLY** supported location for build-related settings in pnpm v10.26+ and v11+, replacing the older `onlyBuiltDependencies` setting in `.npmrc`.

#### 2. Node.js Engine Specifications

Updated `package.json` files to include proper engine specifications:

```json
{
  "engines": {
    "node": ">=24.0.0",
    "pnpm": ">=9.0.0"
  }
}
```

#### 3. Version Management

Added `.nvmrc` file to specify Node.js version requirement:

```
24
```

### Key Changes Made

1. **Container Image**: Uses `docker.io/library/node:24-slim` ensuring consistent Node.js version
2. **Configuration**: Dynamically adds engine specifications and allows builds in cloned repositories on demand
3. **Backwards Compatibility**: Safe to run multiple times (idempotent)
4. **Security**: All file operations use validated and sanitized paths

### Technical Details

- **Root Cause**: In pnpm v10.26+, `.npmrc` only handles auth and registry settings; build-related configs must be in `pnpm-workspace.yaml`
- **Old Approach**: `.npmrc` with `onlyBuiltDependencies` (deprecated)
- **New Approach**: `pnpm-workspace.yaml` with `allowBuilds: { esbuild: true }` 
- **Node Version**: Locked to Node.js 24 (matches container image version)

### Integration Point

The configuration is automatically applied in `pkg/setup/pnpm_patch.go` via the `patchPnpmDependencies()` function, which:
1. Creates/updates `pnpm-workspace.yaml` in both repositories
2. Updates `package.json` in both repositories to specify required engine versions  
3. Handles idempotency and safely manages missing files

### Version Alignment

- **Container Runtime**: Node.js 24-slim Docker image
- **Developers Local**: Managed by .nvmrc to use Node.js 24
- **package.json**: Specifies `>=24.0.0` requirement
- **pnpm Version**: `>=9.0.0` to align with container ecosystem

## Testing

All changes maintain:
- Unit test compatibility `go test ./pkg/setup/...`
- Integration test compatibility `go test -tags integration ./tests/integration/...`
- Security compliance with gosec and govulncheck
- Backwards compatibility with previous configurations