## ADDED Requirements

### Requirement: Sui config existence check before client commands

The system SHALL verify that `~/.sui/sui_config/client.yaml` exists before invoking any `sui client` subcommand on the host. If the file does not exist, the system SHALL report `not configured` and SHALL NOT execute any `sui client` subcommand.

#### Scenario: Doctor run with existing sui config
- **WHEN** `efctl doctor` is executed and `~/.sui/sui_config/client.yaml` exists
- **THEN** `gatherSuiClient()` SHALL execute `sui client active-env`, `sui client active-address`, and `sui client envs` subcommands normally and populate the report fields

#### Scenario: Doctor run without sui config
- **WHEN** `efctl doctor` is executed and `~/.sui/sui_config/client.yaml` does not exist
- **THEN** `gatherSuiClient()` SHALL set `Found: true`, `ActiveEnv` to `not configured`, and SHALL NOT invoke any `sui client` subcommand

#### Scenario: Doctor run without sui binary
- **WHEN** `efctl doctor` is executed and the `sui` binary is not found in PATH
- **THEN** `gatherSuiClient()` SHALL set `Found: false` and SHALL NOT execute any sui-related logic (existing behavior preserved)

### Requirement: No credentials in doctor output

The system SHALL NOT output BIP-39 mnemonics, secret recovery phrases, or private keys to stdout during `efctl doctor` execution under any circumstances.

#### Scenario: Partial environment after failed env up
- **WHEN** `efctl env up` fails partway through (e.g., pnpm/esbuild error) and user runs `efctl doctor`
- **THEN** the doctor output SHALL NOT contain any mnemonic or secret recovery phrase text

### Requirement: Config guard on deployment summary address resolution

The `resolveAddress()` function SHALL check for the existence of `~/.sui/sui_config/client.yaml` before invoking `sui client addresses --json`. If the config does not exist, the function SHALL return an empty string without executing the sui command.

#### Scenario: Summary with valid sui config
- **WHEN** `resolveAddress("alias")` is called and `~/.sui/sui_config/client.yaml` exists
- **THEN** the function SHALL execute `sui client addresses --json` and parse the result normally (existing behavior preserved)

#### Scenario: Summary without sui config
- **WHEN** `resolveAddress("alias")` is called and `~/.sui/sui_config/client.yaml` does not exist
- **THEN** the function SHALL return an empty string without invoking the sui CLI
