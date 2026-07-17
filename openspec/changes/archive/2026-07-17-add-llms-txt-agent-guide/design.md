## Context

`efctl` exposes a broad Cobra command tree across local environment setup, container operations, extension and assembly workflows, GraphQL/world inspection, and tool installation. The facts needed to operate it are spread across `cmd/`, `pkg/config`, generated command pages in `docs/`, narrative guides, security documentation, and provider-specific agent files. Reconnaissance found drift in `USAGE.md` (including missing commands, obsolete extension-publish discovery behavior, and outdated component defaults), so a guide assembled from one narrative source alone would reproduce errors.

The llms.txt proposal defines a Markdown index with one H1, a summary blockquote, optional non-heading context, and H2-delimited link lists. The requested repository artifact is `LLMS.txt`; canonical website discovery normally uses lowercase `/llms.txt`, so this design treats content conformance and web discovery naming as separate concerns.

The guide serves agents operating a developer workstation. It must account for commands that modify repositories and configuration, start or destroy containers and volumes, execute arbitrary workspace commands, publish Move packages, and authorize or deploy assemblies. It must also avoid disclosing mnemonic phrases, private keys, PostgreSQL passwords, or other credential material.

## Goals / Non-Goals

**Goals:**

- Give agents a concise, source-backed map of every supported command family and the operational skill each enables.
- Teach safe workflows with prerequisites, sequencing, verification, recovery, approval boundaries, and an explicit classification of automation-safe versus interactive/TTY-dependent behavior.
- Explain configuration discovery and precedence, supported keys, operational environment variables, workspace/mount semantics, endpoints, and exposure risks.
- Conform the content of `LLMS.txt` to the llms.txt Markdown shape while keeping linked detail in existing authoritative files.
- Detect structural drift, broken repository links, missing representative command coverage, and credential-like examples using existing Go tooling.
- Make guide maintenance an explicit part of future CLI, configuration, and workflow changes.

**Non-Goals:**

- Changing command behavior, configuration semantics, container topology, public APIs, or supported networks.
- Replacing generated CLI documentation, `USAGE.md`, security policy, or provider-specific instruction files.
- Embedding exhaustive generated help output or environment-specific object IDs and credentials.
- Generating `LLMS.txt` from Go source, adding a third-party parser/link checker, or requiring Docker-backed E2E tests for a documentation-only change.
- Publishing a website-root lowercase `/llms.txt`; that can be added separately if web discovery becomes a requirement.

## Decisions

### Use an llms.txt index plus a compact operational preamble

`LLMS.txt` will begin with `# efctl`, a single summary blockquote, and concise non-heading Markdown that defines agent operating rules, configuration precedence, endpoint distinctions, and skill-oriented runbooks. H2 sections will then contain only Markdown link lists with informative descriptions. Planned link groups are core documentation, agent skills/workflows, command reference, operations and security, development/maintenance, and optional deep references.

This preserves the format's machine-readable shape while giving an agent enough high-value guidance to act without copying forty generated command pages into one file. The alternative—an exhaustive standalone manual—was rejected because it would be large, duplicate generated content, and drift quickly.

### Preserve the requested `LLMS.txt` path and define repository-only discovery

The implementation will create exactly the requested root `LLMS.txt`. Its content will follow the llms.txt grammar, but this repository artifact will not claim canonical website `/llms.txt` discovery or interoperability through case-sensitive URL lookup. A lowercase duplicate was rejected because repositories on case-insensitive filesystems cannot reliably carry both names and duplicated content can drift; a symlink was rejected because consumer and platform support varies. If the project later serves the guide on a website, publication tooling should expose this single source at lowercase `/llms.txt` and rewrite links as needed.

### Apply an explicit authority hierarchy

Content will be reconciled in this order:

1. Executable behavior and defaults in `cmd/` and relevant `pkg/` packages.
2. Generated `docs/efctl*.md` command pages.
3. Current OpenSpec requirements and tests for safety-sensitive behavior.
4. `README.md`, `USAGE.md`, and provider-specific instruction summaries.

When sources disagree, `LLMS.txt` will state current behavior from the higher-ranked source and link to the most reliable detail rather than perpetuating the lower-ranked claim. This reconciliation also applies when Cobra help and generated pages overstate executable behavior—for example, `env run` permits only safe-name arguments rather than arbitrary shell syntax. The alternative of treating all documentation as equally authoritative was rejected after reconnaissance identified concrete drift.

### Organize capabilities as agent skills with operational contracts

The guide will group commands into reusable skills rather than only list syntax:

- diagnose prerequisites and environment state;
- initialize and configure a workspace, including the files and git metadata written by `init`/`init --ai`;
- start, inspect, and stop the local environment, including the persistent resources removed by teardown;
- initialize, list, build, test, and publish extensions;
- deploy and authorize assembly, storage, gate, and turret objects;
- run safe-name commands or enter a shell in the builder container;
- use faucet, GraphQL, package/object, and world-query inspection;
- install Sui with its conditional confirmation prompts; and
- use version, self-update, and completion maintenance commands, including executable replacement during update.

Each mutating or externally scoped skill will identify side effects, preconditions, network/workspace selection, interaction mode, a verification command, recovery guidance, and actions that require human approval. In particular, the guide will state that `sui install` is not unattended-safe when prerequisites are absent because it can prompt twice; `--no-progress` and `CI=true` suppress spinners but do not answer prompts. A flat command inventory was rejected because it does not help an agent sequence operations safely.

### Separate configuration, environment, and service-address concerns

The guide will distinguish root `--config-file`, `--debug`, and `--no-progress` controls from command-local flags. It will provide command-path-specific precedence rather than a misleading global rule: config discovery walks upward for `efctl.yaml` before `efctl.yml` unless `--config-file` is supplied; `--with-graphql` and `--with-frontend` override loaded YAML only when explicitly set; prerequisite engine checks consult valid YAML `container-engine` before `EFCTL_ENGINE` and auto-detection, while the actual container client currently consults `EFCTL_ENGINE`, `DOCKER_HOST`, daemon reachability, and Podman-before-Docker fallback without consulting YAML. It will also cover workspace resolution, every supported configuration key, deprecated branch aliases, and operational variables including `CI`, `EFCTL_ENGINE`, `DOCKER_HOST`, `EFCTL_STARTUP_TIMEOUT_SECONDS`, and `EFCTL_PG_PASSWORD`.

It will separate reachable service endpoints from startup preflight reservations: Sui JSON-RPC uses host/container `9000`, faucet `9123`, GraphQL host/container `9125` at `/graphql`, frontend host `5173`, and PostgreSQL host `5432` only when exposed. With GraphQL enabled, current startup preflight separately requires host ports `8000` and `5432` to be free even though the GraphQL service is published on `9125`; the guide will record this current behavior as a precondition, not mislabel `8000` as the GraphQL endpoint. Secret-valued variables will be named but never populated with realistic values.

### Encode security and approval boundaries in the guide

Agents will be instructed to run `efctl doctor`, confirm the absolute workspace and target network, prefer `--no-progress`/`CI=true` for captured output, verify state after mutation, and stop for approval before destructive cleanup, `--force` overwrite, remote-network publication/authorization, or broad host/PostgreSQL exposure. Additional bind mounts will be treated as explicit host-data access grants. Diagnostic output and examples must never expose mnemonics, private keys, recovery phrases, or passwords.

This is guidance, not a new authorization mechanism. Runtime enforcement was rejected as out of scope.

### Validate with a dependency-free Go documentation test

A focused root-package test will read `LLMS.txt` relative to its source location and verify:

- the first meaningful line is the sole H1 and a summary blockquote follows;
- subsequent headings are H2 link-list sections;
- every local Markdown link resolves to an existing repository file;
- representative commands from every skill family are present;
- stable acceptance tokens cover every supported config key and operational variable, automation/TTY classification, exact host endpoint mappings, and approval/recovery rules for each mutating family;
- prohibited credential-like example patterns are absent.

The test will express these tokens as a small table of named acceptance concerns so failures identify the missing contract rather than pinning full prose. Manual review will use the same matrix to verify semantics that token checks cannot prove, including source/help conflicts and side-effect accuracy. The test will use only the Go standard library and run under `go test ./...`, so existing local and pre-commit gates cover it. Adding an external llms.txt parser or network link checker was rejected to avoid dependency and flaky-network costs.

### Define maintenance triggers rather than generate the guide

The guide will state that changes to Cobra commands/flags, `pkg/config`, operational environment variables, service endpoints, workflow side effects, or generated command references must review and update `LLMS.txt` in the same change. Implementation validation will regenerate CLI docs and inspect their diff before running tests and pre-commit. Automatic generation was rejected because the most valuable content—safe sequencing, recovery, and approval boundaries—is semantic rather than derivable from Cobra metadata.

## Risks / Trade-offs

- [Uppercase filename is not the canonical web discovery path] → Treat `LLMS.txt` as the explicitly requested repository-only artifact, do not promise lowercase URL discovery, and add lowercase publication only through future website tooling.
- [Manual operational prose can drift from implementation] → Use the authority hierarchy, command-family coverage tests, maintenance triggers, generated-doc drift checks, and review against source.
- [A command-presence test can pass while semantics are stale] → Keep tests focused on baseline coverage and require human/source review for defaults, side effects, and recovery guidance.
- [Security guidance may give agents false confidence] → Describe approval and trust boundaries clearly and link to `SECURITY.md` and the threat model; do not claim runtime policy enforcement.
- [Relative links are portable in the repository but not every raw-text consumer resolves them] → Use repository-root-relative Markdown destinations consistently and informative descriptions; website publication can rewrite them to absolute URLs later.
- [The guide could become too long for its indexing purpose] → Keep detailed command syntax in generated pages and include only decision-critical instructions and examples in `LLMS.txt`.

## Migration Plan

1. Add the failing documentation-contract test for the requested root file and required coverage.
2. Author `LLMS.txt` from the reconciled command/configuration inventory, interaction/side-effect matrix, endpoint mapping, and existing security guidance.
3. Run the focused test, regenerate command docs and inspect for drift, then run `go test ./...` and all pre-commit hooks.
4. Review links and examples manually from an agent/operator perspective.

Rollback is removal of `LLMS.txt` and its documentation-contract test; no runtime data or configuration migration is involved.

## Open Questions

- If efctl documentation is later published as a website, should release tooling expose this source as canonical lowercase `/llms.txt` and rewrite repository-relative links to absolute published URLs?
