# PNPM ESBuild E2E Fix Hunt

## PROBLEM
The CI e2e workflow fails during `pnpm install` inside the container. The install stops with the following error, which then prevents world deployment:
```
[ERR_PNPM_IGNORED_BUILDS] Ignored build scripts: esbuild@0.27.2
Run "pnpm approve-builds" to pick which dependencies should be allowed to run scripts.
Deployment failed: failed to deploy world: exec error: exit status 1
```

## ROOT CAUSE
pnpm 10+ blocks build scripts unless they are explicitly allowed. `esbuild` needs its build step to produce the native binary required by `pnpm deploy-world`. The older `onlyBuiltDependencies=esbuild` configuration in `.npmrc` did not take effect reliably in this environment, and `pnpm approve-builds esbuild` reported that there were no pending approvals. As a result, the install continued to skip the `esbuild` build script.

## STEPS TRIED

### Step 5: pnpm-workspace.yaml with allowBuilds (implemented)
Create `pnpm-workspace.yaml` with `allowBuilds: { esbuild: true }` in each repo.
**Result:** Implemented and pending CI confirmation. The implementation handles both new file creation and merging into existing `allowBuilds:` blocks to avoid duplicate YAML keys.

### Step 1: .npmrc patch (commit c5963f2)
Add `onlyBuiltDependencies=esbuild` to `.npmrc` in `world-contracts` and `builder-scaffold`.
Also add `"onlyBuiltDependencies": ["esbuild"]` to `package.json` pnpm block.
**Result:** CI still failed. pnpm 10 still blocked the `esbuild` build script.

### Step 2: pnpm approve-builds (commit e4ea805)
Change `CmdDeployWorld` to:
```
pnpm approve-builds esbuild && pnpm install && pnpm deploy-world
```
**Result:** CI still failed. The logs showed "There are no packages awaiting approval," so pnpm did not treat the configuration in `.npmrc` or `package.json` as a pending approval in pnpm 10.

### Step 3: pnpm install || true
Try skip error with `|| true`.
**Result:** This was a bad idea. The `esbuild` native binding was still not built, so `pnpm deploy-world` failed later because the binary was missing.

### Step 4: Copilot review fixes (commit 8cf75b5)
- Fix CRLF handling in `patchNpmrc` — trim `\r` too
- Fix wrong `#nosec G304` (should be `G306` for write)
- Fix test error handling in `TestPatchPackageJSON`, `TestPatchNpmrc_Idempotent`
**Result:** Unit tests passed and `gosec` was clean, but the e2e flow still failed.

## WHAT WE KNOW
1. The container runs `pnpm install`, and pnpm 10 blocks the `esbuild` build script.
2. `.npmrc` with `onlyBuiltDependencies=esbuild` does not work as expected.
3. The `package.json` pnpm block does not work either.
4. `pnpm approve-builds esbuild` does not work; pnpm reports that there is nothing to approve.
5. The `esbuild` native binding must be built for `pnpm deploy-world` to work.
6. `pnpm install || true` is not a valid workaround; `esbuild` remains unbuilt and deployment still fails later.
7. The Docker image uses pnpm 10+ with Corepack.
8. The container uses Node.js v20.20.2, and the logs warn about version drift.

## INVESTIGATION PROMPT FOR NEXT AGENT

```
Hey agent. Help me fix pnpm esbuild e2e fail. Here is what we try already.

Context:
- efctl env up build Docker image from builder-scaffold/Dockerfile
- Container run pnpm install then pnpm deploy-world inside world-contracts
- pnpm 10+ block esbuild build scripts by default
- .npmrc onlyBuiltDependencies config not work for pnpm 10
- pnpm approve-builds not work (say nothing to approve)

Files to check:
- pkg/setup/constants.go (CmdDeployWorld line)
- pkg/setup/pnpm_patch.go (patchNpmrc, patchPackageJSON)
- pkg/setup/start.go (StartEnvironment calls patchPnpmDependencies)
- pkg/container/services.go (FrontendConfig has its own pnpm install)
- world-contracts/Dockerfile (from upstream clone, what pnpm version?)
- builder-scaffold/Dockerfile (from upstream clone)

Questions to answer:
1. What pnpm version in container? Check Dockerfile FROM image.
2. Does pnpm 10 support onlyBuiltDependencies in .npmrc? Or need .npmrcrc?
3. Can we add esbuild install step in Dockerfile instead of at runtime?
4. Should we use npm or yarn instead? (probably not — pnpm is standard)
5. Can we set pnpm config via ENV or CLI flag? --config.onlyBuiltDependencies?
6. Is there a .npmrcrc or pnpm config file we should patch instead?
7. Does the FrontendConfig pnpm install also fail? Check line 115 services.go

Suggested approaches to evaluate:
A. Patch Dockerfile to run `pnpm add -g esbuild` or similar at build time
B. Use pnpm CLI flag: `pnpm install --config.onlyBuiltDependencies='["esbuild"]'`
C. Create .npmrcrc file with esbuild config (pnpm 10 behavior?)
D. Patch entrypoint.sh to run approve-builds before deploy
E. Use --ignore-scripts flag but pre-build esbuild binary separately
F. Check if pnpm version in container supports the config differently

Pre-conditions (must have):
- All unit test pass: `go test -v ./pkg/setup/...`
- gosec clean: `gosec -quiet ./pkg/setup/...`
- govulncheck clean: `govulncheck ./...`
- e2e test pass locally (if Docker available): `go test -tags e2e -timeout 15m ./tests/e2e/...`
- e2e test pass in CI

Post-conditions (must achieve):
- CI e2e-tests check turn green
- pnpm install complete without ERR_PNPM_IGNORED_BUILDS error
- esbuild native binding compile successful
- pnpm deploy-world execute without fail
- No breaking change to existing behavior
- .npmrc and package.json patches remain idempotent

Output:
- Analysis of root cause
- Recommended fix approach with justification
- Implementation plan (files to change, what to change)
- Verification steps
```

## CURRENT STATE
- Branch: `chore/bump-codeql-and-crypto`
- Last commit: `8cf75b5` — Copilot fixes (CRLF, #nosec, test errors)
- `CmdDeployWorld` currently: `pnpm install --prefer-offline && pnpm deploy-world`
- CI status: e2e-tests FAILING, all other checks PASSING
- Unit tests: PASSING
- gosec: PASSING

## ROOT CAUSE DISCOVERED (Step 5)

### The pnpm 11 Breaking Change

The container runs **pnpm v11** (or at least v10.26+). In pnpm v10.26-v11:

1. **`.npmrc` ONLY reads auth and registry settings** — build-related settings like `onlyBuiltDependencies` are completely ignored
2. **`package.json`'s `pnpm` field is ignored** — `pnpm.onlyBuiltDependencies` does not work
3. **`onlyBuiltDependencies` was REMOVED** — replaced by `allowBuilds` in `pnpm-workspace.yaml`

### Why Previous Patches Failed

| Patch | File | Why It Failed |
|-------|------|---------------|
| Step 1 | `.npmrc` with `onlyBuiltDependencies=esbuild` | `.npmrc` ignored for non-auth settings |
| Step 1 | `package.json` with `pnpm.onlyBuiltDependencies` | `pnpm` field ignored in v11 |
| Step 2 | `pnpm approve-builds esbuild` | Says "no packages awaiting" because pnpm cached the decision to block |

### The Correct Fix

Create `pnpm-workspace.yaml` with `allowBuilds`:

```yaml
allowBuilds:
  esbuild: true
```

This is the ONLY supported location for build-related settings in pnpm v10.26+ / v11+.

The `allowBuilds` setting was added in pnpm v10.26.0 and replaces all deprecated settings:
- `onlyBuiltDependencies` → `allowBuilds: { esbuild: true }`
- `neverBuiltDependencies` → `allowBuilds: { pkg: false }`
- `ignoredBuiltDependencies` → `allowBuilds: { esbuild: false }`

### Implementation

- `patchPnpmDependencies()` now creates `pnpm-workspace.yaml` in both `builder-scaffold/` and `world-contracts/`
- Removed old `patchPackageJSON()` and `patchNpmrc()` functions (they don't work in pnpm 11)
- `CmdDeployWorld` unchanged (no `pnpm approve-builds` needed — `allowBuilds` pre-approves)
- All unit tests pass
- gosec clean (added G703 to #nosec for taint analysis)

## NOTES
- Copilot 4 review comments fixed
- CRLF fix good for cross-platform
- #nosec G304→G306 fix correct
- Test error handling fixes good
- Nothing about pnpm/esbuild yet
