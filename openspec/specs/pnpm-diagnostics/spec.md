## ADDED Requirements

### Requirement: Pnpm version diagnostic before deploy-world install

The `CmdDeployWorld` command SHALL output the installed pnpm version and the contents of the `pnpm-workspace.yaml` file in the working directory before executing `pnpm install`. This diagnostic output MUST appear in the container log stream.

#### Scenario: Successful diagnostic output on world deploy
- **WHEN** `CmdDeployWorld` is executed inside the sui-playground container
- **THEN** the output SHALL include `pnpm --version` and the contents of `pnpm-workspace.yaml` (or a "not found" message) before `pnpm install` begins

#### Scenario: Missing workspace yaml
- **WHEN** `pnpm-workspace.yaml` does not exist in `/workspace/world-contracts/` at deploy time
- **THEN** the diagnostic output SHALL print a clear "not found" message so the absence is visible in logs

### Requirement: Fallback esbuild approval before install

The `CmdDeployWorld` command SHALL run `pnpm approve-builds esbuild` (or equivalent fallback) before `pnpm install` to ensure esbuild build scripts are approved even if the `pnpm-workspace.yaml` config is not recognized.

#### Scenario: Normal flow with existing allowBuilds config
- **WHEN** `CmdDeployWorld` is executed and `pnpm-workspace.yaml` with `allowBuilds: esbuild: true` is present
- **THEN** the `approve-builds` fallback step SHALL execute without error (idempotent no-op or acknowledgment) and `pnpm install` SHALL succeed

#### Scenario: Fallback when config is not recognized
- **WHEN** `CmdDeployWorld` is executed and the pnpm version does not recognize `allowBuilds` in the workspace yaml
- **THEN** the `approve-builds` fallback step SHALL attempt to approve esbuild and SHALL NOT cause the overall command to fail (must not block on interactive prompts)

### Requirement: Diagnostic command must not block pipeline

The diagnostic and fallback steps appended to `CmdDeployWorld` SHALL NOT cause the deploy pipeline to hang or fail. All diagnostic echo commands SHALL be wrapped or structured to continue on failure.

#### Scenario: approve-builds returns error
- **WHEN** `pnpm approve-builds esbuild` exits with a non-zero status
- **THEN** the deploy pipeline SHALL continue to `pnpm install` without stopping
