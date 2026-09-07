## MODIFIED Requirements

### Requirement: Pnpm version diagnostic before deploy-world install
The `CmdDeployWorld` command SHALL identify the `sui-playground` execution boundary and output the container operating system and architecture when available, the pnpm launcher resolved by `command -v`, the project's declared `packageManager`, the result of `pnpm --version`, and the contents of `pnpm-workspace.yaml` before executing `pnpm install`. Diagnostic failures other than inability to start the required pnpm command MUST NOT block deployment, and pnpm stderr MUST remain visible so a selected standalone executable and loader failure can be identified.

#### Scenario: Successful diagnostic output on world deploy
- **WHEN** `CmdDeployWorld` is executed inside the `sui-playground` container
- **THEN** the output SHALL identify `sui-playground` and include available OS and architecture information
- **AND** the output SHALL include the pnpm launcher path, declared `packageManager`, `pnpm --version`, and contents of `pnpm-workspace.yaml` or a clear "not found" message before `pnpm install` begins

#### Scenario: Missing workspace yaml
- **WHEN** `pnpm-workspace.yaml` does not exist in `/workspace/world-contracts/` at deploy time
- **THEN** the diagnostic output SHALL print a clear "not found" message so the absence is visible in logs

#### Scenario: Project-selected pnpm cannot load
- **WHEN** the global pnpm launcher dispatches to a project-selected standalone executable that cannot load a shared library
- **THEN** the deployment output SHALL preserve the selected executable path and dynamic-loader error from pnpm stderr
- **AND** the failure SHALL be attributable to the `sui-playground` execution boundary rather than the host or frontend container
