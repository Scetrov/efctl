## 1. Establish the Documentation Contract

- [x] 1.1 Add a root-package Go test that locates `LLMS.txt` from the test source and validates the sole H1, following summary blockquote, H2 link-list structure, and descriptive link entries using only the standard library
- [x] 1.2 Extend the test with a named acceptance matrix that resolves every repository-local Markdown link; requires representative commands, every supported config/environment identifier, exact host endpoint mappings, interaction-mode warnings, and approval/recovery markers for each mutating family; and rejects credential-like example material
- [x] 1.3 Run the focused documentation test before creating `LLMS.txt` and confirm it fails for the missing guide with an actionable diagnostic

## 2. Reconcile Authoritative efctl Behavior

- [x] 2.1 Inventory the current Cobra command tree and generated `docs/efctl*.md` pages, recording every capability family and exact command-reference destination; reconcile help text against executable restrictions such as `env run` safe-name arguments
- [x] 2.2 Build an explicit command-path config/endpoint matrix from `cmd/` and `pkg/` covering upward `efctl.yaml`/`efctl.yml` discovery, every supported key and deprecated alias, explicit feature-flag overrides, prerequisite YAML-before-`EFCTL_ENGINE` engine preference versus the actual client's `EFCTL_ENGINE`/`DOCKER_HOST`/daemon selection, workspace/mount behavior, all operational variables, service mappings including GraphQL `9125` to `9125`, and the separate GraphQL-startup free-port checks for `8000` and `5432`
- [x] 2.3 Build a mutating-capability matrix for `init`/`init --ai`, `env up`, `env down`, extension publish, assembly deploy/authorize, `env run`, `env shell`, conditional-interactive `sui install`, and self-update, recording side effects, prerequisites, network scope, TTY/automation mode, verification, recovery, approval, and secret/exposure boundaries from source and security requirements

## 3. Author the Agent Operations Guide

- [x] 3.1 Create root `LLMS.txt` with `# efctl`, a concise blockquote summary, non-heading operating context, and H2 sections containing only descriptive Markdown link lists
- [x] 3.2 Document repository-only uppercase filename scope, the authority hierarchy, command-path-specific config/workspace precedence, non-interactive output controls versus real prompts, service endpoints versus startup-reserved ports, verification discipline, recovery approach, and human-approval boundaries in the preamble
- [x] 3.3 Add skill-oriented instructions from the mutating-capability matrix for diagnosis and initialization; environment lifecycle; extension init/list/build/test/publish; assembly deployment/authorization; restricted container run/shell; faucet and GraphQL/world inspection; conditionally interactive Sui installation; and mutating CLI maintenance
- [x] 3.4 Add source-backed configuration and environment-variable guidance without secret values, including host/PostgreSQL exposure and additional-bind-mount cautions
- [x] 3.5 Add descriptive links to core, generated command, operations/security, and contributor references plus an explicit maintenance rule for agent-visible command, config, endpoint, side-effect, and recovery changes

## 4. Validate and Review

- [x] 4.1 Run the focused documentation test and refine `LLMS.txt` or named acceptance-matrix assertions until structure, links, capability/config/environment/endpoint/interaction/safety coverage, and secret-pattern checks pass without exact-prose coupling
- [x] 4.2 Run `make docs`, inspect the generated documentation diff for command drift, and remove any unintended generated changes
- [x] 4.3 Run `go test ./...` and resolve all failures
- [x] 4.4 Run `pre-commit run --all-files` and resolve all formatting, lint, security, generated-doc, and secret-scan failures
- [x] 4.5 Manually walk the config/endpoint and mutating-capability matrices against `LLMS.txt`, verifying source/help reconciliation, side effects, preconditions, interaction mode, network/workspace scope, verification/recovery/approval guidance, valid link descriptions, concise format, and no temporary or unrelated files in the final diff
