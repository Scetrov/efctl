## ADDED Requirements

### Requirement: Repository agent guide artifact
The project SHALL provide the requested repository-root `LLMS.txt` whose content follows the llms.txt Markdown structure: one project H1, a concise summary blockquote, optional non-heading guidance, and H2-delimited file lists containing descriptive Markdown links. The artifact SHALL identify itself as repository guidance and MUST NOT claim canonical lowercase website `/llms.txt` discovery.

#### Scenario: Agent discovers the guide
- **WHEN** an agent inspects the repository root for project guidance
- **THEN** it finds `LLMS.txt` and can identify `efctl`, its purpose, and linked detail without parsing provider-specific instruction files

#### Scenario: Guide structure is machine-readable
- **WHEN** a format validator reads `LLMS.txt`
- **THEN** it finds the sole H1 before all H2 sections and finds descriptive Markdown link entries within each H2 file list

#### Scenario: Agent interprets the uppercase repository filename
- **WHEN** an agent loads `LLMS.txt` from a repository checkout
- **THEN** the guide presents its content as llms.txt-structured repository guidance without promising case-sensitive website discovery at `/llms.txt`

### Requirement: Source-backed capability map
The guide MUST describe every supported `efctl` command family as agent-usable capabilities and MUST direct agents to generated command references for exact syntax and flags. Coverage SHALL include initialization and diagnosis; environment up, down, status, and dashboard; extension init, list, build, test, and publish; assembly deploy and authorize workflows; container run and shell access; faucet, GraphQL, package/object, and world inspection; Sui installation; and completion, update, and version maintenance.

#### Scenario: Agent selects a capability
- **WHEN** an agent needs to perform an `efctl` operation
- **THEN** the guide maps the intent to the appropriate command family and links to an authoritative command reference

#### Scenario: Narrative documentation conflicts with executable behavior
- **WHEN** command source or generated command documentation disagrees with a narrative usage guide
- **THEN** `LLMS.txt` states or links to the executable behavior and does not repeat the stale narrative claim

### Requirement: Operational workflow instructions
For each mutating or externally scoped capability family, the guide SHALL provide or link to side effects, preconditions, interaction mode, safe sequencing, a post-action verification step, recovery guidance, and approval boundaries. This coverage MUST include `init`/`init --ai`, `env up`, `env down`, extension publish, assembly deploy/authorize, `env run`, `env shell`, `sui install`, and self-update. The environment lifecycle instructions MUST direct agents to diagnose prerequisites before startup, verify status after startup, and use environment teardown as recovery from a partial startup when appropriate.

#### Scenario: Agent starts a local environment
- **WHEN** an agent plans to run `efctl env up`
- **THEN** the guide directs it to validate prerequisites and workspace/config selection, use non-interactive output when capturing logs, verify the resulting environment, and retain actionable failure output

#### Scenario: Startup fails after partial mutation
- **WHEN** environment startup fails after repositories or containers may have been created
- **THEN** the guide directs the agent to inspect the reported error and recent state before using `efctl env down` as the documented cleanup path

#### Scenario: Agent performs an extension workflow
- **WHEN** an agent initializes, builds, tests, or publishes an extension
- **THEN** the guide presents the lifecycle in dependency order and requires explicit workspace, extension path, network selection, and post-operation verification where applicable

#### Scenario: Agent evaluates Sui installation for unattended execution
- **WHEN** an agent considers `efctl sui install` and required tools may be absent
- **THEN** the guide states that the command can issue two confirmation prompts, that `--no-progress` and `CI=true` do not answer them, and that unattended execution requires prerequisites to be present or a human-controlled interactive run

#### Scenario: Agent runs a builder-container command
- **WHEN** an agent uses `efctl env run`
- **THEN** the guide states that the command and each argument are restricted to safe-name characters and does not repeat generated help claims that imply arbitrary shell syntax

### Requirement: Configuration and execution model
The guide MUST provide explicit command-path-specific configuration precedence, workspace resolution, global execution flags, all supported `efctl.yaml` keys, deprecated configuration aliases, and operational environment variables. It SHALL distinguish configuration defaults from command flag defaults, distinguish prerequisite engine preference from actual container-client selection, and distinguish startup preflight port reservations from Sui JSON-RPC/faucet, GraphQL, frontend, and PostgreSQL service addresses.

#### Scenario: Agent resolves effective configuration
- **WHEN** an agent operates in a nested workspace without an explicit `--config-file`
- **THEN** the guide explains upward discovery preferring `efctl.yaml` over `efctl.yml`, configuration validation, workspace handling, explicitly set feature-flag overrides, YAML-before-environment preference during prerequisite checks, and the actual client's separate `EFCTL_ENGINE`/`DOCKER_HOST`/daemon-reachability selection that does not consult YAML

#### Scenario: Agent prepares automation output
- **WHEN** an agent runs `efctl` in CI or captures command output
- **THEN** the guide instructs it to use `--no-progress` or `CI=true` and to enable `--debug` only when diagnostic detail is needed

#### Scenario: Agent configures an operational override
- **WHEN** an agent needs to select a container engine, daemon, startup timeout, or PostgreSQL password source
- **THEN** the guide identifies `EFCTL_ENGINE`, `DOCKER_HOST`, `EFCTL_STARTUP_TIMEOUT_SECONDS`, and `EFCTL_PG_PASSWORD`, explains their precedence and purpose, and does not embed a secret value

#### Scenario: Agent selects a local service endpoint
- **WHEN** an agent needs to connect to an efctl-managed service
- **THEN** the guide distinguishes host/container Sui JSON-RPC `9000`, faucet `9123`, GraphQL host/container `9125` at `/graphql`, frontend host `5173`, and PostgreSQL host `5432` when explicitly exposed

#### Scenario: Agent diagnoses a GraphQL startup port failure
- **WHEN** GraphQL-enabled startup reports host port `8000` or `5432` unavailable
- **THEN** the guide identifies these as current startup preflight reservations, distinguishes them from the GraphQL endpoint on `9125`, and does not claim that GraphQL is mapped from container port `8000`

### Requirement: Security and approval boundaries
The guide SHALL identify destructive, externally visible, and host-access-expanding actions and MUST instruct agents to obtain human approval before actions outside the already authorized local-development scope. It MUST prohibit printing or recording mnemonics, recovery phrases, private keys, and passwords, and MUST treat additional bind mounts and non-loopback service exposure as explicit security decisions.

#### Scenario: Agent approaches a destructive action
- **WHEN** an agent would overwrite configuration with `--force`, let `init` alter git/workspace files, remove containers/images/networks/volumes with `env down`, execute a builder-container command or shell, replace the efctl executable through self-update, or run broader cleanup
- **THEN** the guide identifies the side effect and requires confirmation of workspace and impact plus human approval unless that exact action was already authorized

#### Scenario: Agent approaches a remote or exposed operation
- **WHEN** an agent would publish or authorize against a non-local network, bind services beyond loopback, expose PostgreSQL, or add a host bind mount
- **THEN** the guide explains the increased scope and requires explicit approval before proceeding

#### Scenario: Agent handles diagnostic or secret-bearing data
- **WHEN** an agent reads configuration, environment files, doctor output, or command failures
- **THEN** the guide instructs it to redact secret material and never echo mnemonic, private-key, recovery-phrase, or password values

### Requirement: Local documentation navigation
Every local file destination linked from `LLMS.txt` MUST resolve within the repository, and each link MUST include a concise description of the information an agent should retrieve from it. Detailed syntax SHALL remain in linked generated or maintained documentation instead of being exhaustively duplicated in the guide.

#### Scenario: Agent follows a local reference
- **WHEN** an agent resolves any repository-local link in `LLMS.txt`
- **THEN** the destination exists and the link description states why the destination is relevant

#### Scenario: Agent needs exact flag syntax
- **WHEN** an agent requires exact arguments, flags, or defaults for a command
- **THEN** the guide directs it to the corresponding generated command page rather than relying on an incomplete copied help block

### Requirement: Guide validation and maintenance
The project MUST provide a dependency-free automated check that validates the stable guide contract, including structure, local-link integrity, representative command-family coverage, all config/environment identifiers, exact host endpoint mappings, interaction-mode warnings, mutating-family approval/recovery markers, and absence of credential-like example material. Changes to commands, flags, configuration keys, operational environment variables, service endpoints, side effects, or generated command references SHALL review and update `LLMS.txt` in the same change when agent-visible behavior changes.

#### Scenario: Guide contract regresses
- **WHEN** `LLMS.txt` loses its required structure, contains a broken local link, omits a representative capability/config/environment/endpoint/interaction/safety concern from the acceptance matrix, or adds prohibited credential-like example material
- **THEN** the standard Go test suite fails with an actionable diagnostic naming the missing or prohibited concern

#### Scenario: Agent-visible CLI behavior changes
- **WHEN** a change modifies an `efctl` command, flag, configuration rule, environment override, endpoint, side effect, or recovery path
- **THEN** maintainers reconcile `LLMS.txt` against command source and regenerated CLI documentation before the change is accepted

#### Scenario: Documentation-only implementation is verified
- **WHEN** this capability is implemented
- **THEN** generated CLI documentation is checked for drift, a manual acceptance matrix verifies source/help reconciliation and every mutating family's side effects/preconditions/interaction/verification/recovery/approval guidance, `go test ./...` passes, all pre-commit hooks pass, and the final diff contains no unintended generated or temporary artifacts
