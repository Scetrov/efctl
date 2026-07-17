## Why

Agents currently have to reconstruct `efctl`'s operational model from command source, generated CLI pages, usage guides, and provider-specific instruction files, which are incomplete in different ways and can lead to unsafe or ineffective automation. A repository-root, llms.txt-formatted guide will give agents one concise, source-backed entry point for choosing commands, configuring environments, sequencing workflows, and handling destructive or security-sensitive operations.

## What Changes

- Add the requested repository-root `LLMS.txt` whose Markdown content follows the llms.txt grammar and links agents to authoritative project documentation; explicitly scope it as a repository guide rather than claiming canonical lowercase website discovery.
- Describe `efctl` capabilities as reusable agent skills: environment diagnosis and lifecycle, extension development and publishing, assembly deployment and authorization, container execution and shell access, GraphQL/world inspection, Sui installation, and CLI maintenance commands.
- Provide agent-oriented operating instructions for prerequisites, workspace/config resolution, automation-safe versus interactive execution, service endpoint mappings, workflow sequencing, validation, recovery, and escalation before destructive or externally visible actions.
- Document configuration keys, command-line override behavior, operational environment variables, defaults, network/host exposure risks, mount boundaries, and secret-handling rules without embedding credentials.
- Reconcile the guide against command source and generated CLI documentation, explicitly avoiding stale behavior already present in narrative usage docs.
- Add lightweight automated validation for required llms.txt structure, local-link integrity, a stable content-acceptance matrix covering workflows/configuration/endpoints/safety, representative command coverage, and the absence of credential-like example material.
- Define an update policy so changes to commands, flags, configuration, or workflows keep `LLMS.txt` synchronized.

## Capabilities

### New Capabilities
- `agent-operations-guide`: A source-backed, llms.txt-structured `LLMS.txt` repository entry point that teaches agents `efctl` capabilities, safe workflows, usage, configuration, recovery, and documentation navigation.

### Modified Capabilities

None.

## Impact

- Adds `LLMS.txt` at the repository root plus a focused documentation validation test or script using existing project tooling.
- Draws authoritative content from `cmd/`, `pkg/config`, generated `docs/`, `README.md`, `USAGE.md`, security guidance, and existing operational specifications.
- May extend existing quality-gate wiring so guide validation runs locally and in CI; no `efctl` runtime command, configuration behavior, public API, or production dependency changes are intended.
- Establishes a maintenance obligation for future command, configuration, and workflow changes to update the agent guide in the same change.
