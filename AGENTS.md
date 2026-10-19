# Agent Constitution

This document defines the core principles and operational constraints for any AI Agent working on the `efctl` repository.

## 1. Test-First Development

All features and bug fixes must follow a test-driven approach. Create or update tests before or alongside implementation to ensure the product remains fully verified.

## 2. Testing Pyramid Adherence

Maintain a healthy testing balance according to the pyramid:

- **Unit Tests:** High volume, testing individual functions and logic in isolation.
- **Integration Tests:** Medium volume, testing interactions between modules (e.g., container operations, CLI commands).
- **E2E Tests:** Low volume, verifying full user flows across the entire environment.

When making changes to the codebase, ensure that tests are also updated in-line with the changes.

## 3. Clean Code & Quality Gates

- Write clean, maintainable, and idiomatic Go code.
- Always run `pre-commit` hooks locally to ensure all linting, formatting, and security gates pass before finalizing a task.

## 4. Security-First

Prioritize security in every change. Avoid hardcoding credentials, ensure proper file permissions (e.g., `0600` for sensitive logs), and utilize security scanning tools like `gosec`.

## 5. Independent Operation

The Agent is authorized to operate independently. Continue working through sub-tasks and error resolution unattended until the primary objective is fully complete or a critical blocker is reached.

## 5.1 GitHub Repository Policies

- Use GitHub **Rulesets** for repository-policy changes.
- Do not create or modify legacy branch-protection rules unless the user explicitly requests them.
- Before changing a policy, verify its target branches and enforcement scope.

## 6. Environmental Isolation

To maintain a non-breaking flow and avoid permission requests:

- All temporary files, intermediate build artifacts, or scratchpads must be stored in the `./tmp` directory.
- Do not write to system-level directories or paths outside the repository root.
- Do not write changes to the builder-scaffold and world-contracts directory for anything other than testing, prefer patching the files in code.

## 7. Context-Mode Routing

- Prefer context-mode MCP tools for analysis, research, file processing, and commands likely to produce large output.
- Do not use `curl`, `wget`, inline HTTP shell snippets, or direct web fetch tools for external content. Use `ctx_fetch_and_index(...)` or `ctx_execute(...)` instead.
- Use `ctx_batch_execute(...)` for multi-step repo research and `ctx_search(...)` for follow-up queries against indexed content.
- Use `ctx_execute(...)` or `ctx_execute_file(...)` for logs, search output, and large file inspection. Use normal file reads when preparing to edit a file.
- Keep terminal usage limited to short operational commands such as `git`, `mkdir`, `rm`, `mv`, `cd`, `ls`, and install commands.
- Keep responses concise and write substantial artifacts to files instead of long inline output.
- Utility commands: `ctx stats`, `ctx doctor`, `ctx upgrade`.

## 7.1 Graphify Code Insights

- Before investigating code, architecture, dependencies, or file relationships, check for the ignored `graphify-out/` directory at the repository root.
- When `graphify-out/graph.json` exists, use the existing graph first: `graphify query "<question>" --budget 1500`. Use `graphify path "<source>" "<target>"` to trace relationships and `graphify explain "<node>"` for focused insights.
- Consult `graphify-out/GRAPH_REPORT.md` when available for an architectural overview, central nodes, communities, and suggested questions. Route large graph data and query output through context-mode rather than dumping the full JSON.
- Treat graph insights as navigation aids, not authoritative proof: verify relevant source files before editing or making definitive claims. Cite source locations and distinguish extracted relationships from inferred or ambiguous ones; the graph may be stale or incomplete.
- If the directory, graph, or CLI is unavailable, fall back to targeted source inspection (and available graph artifacts). Do not automatically install graphify or rebuild the graph unless requested.
- Keep `graphify-out/` ignored and untracked; do not commit generated graph artifacts.

## 8. Development Cheat Sheet

Quick reference for common `efctl` operations:

- **Initialize configuration**: `efctl init` (or `efctl init --ai [agent]`)
- **Environment Lifecycle**:
  - Up: `efctl env up`
  - Down: `efctl env down`
- **Status Check**: `efctl env status`
- **Dash**: `efctl env dash`
- **Deploy Extension**: `efctl env extension publish [contract-path]` (path defaults to `./my-extension`)
- **Query World**: `efctl world query [object_id]` (queries the Sui GraphQL RPC)
<!-- EFCTL_INSTRUCTIONS_START -->
# Agent Constitution

This document defines the core principles and operational constraints for any AI Agent working on this repository.

## 1. Test-First Development
All features and bug fixes must follow a test-driven approach.

## 2. Testing Pyramid Adherence
- **Unit Tests:** High volume, isolation.
- **Integration Tests:** Interaction between modules.
- **E2E Tests:** Full user flows.

## 3. Clean Code & Quality Gates
- Write clean, maintainable, and idiomatic Go code.
- Always run pre-commit hooks.

## 4. Security-First
Prioritize security in every change. Avoid hardcoding credentials.

## 5. Independent Operation
The Agent is authorized to operate independently.

## 5.1 GitHub Repository Policies
- Use GitHub **Rulesets** for repository-policy changes.
- Do not create or modify legacy branch-protection rules unless the user explicitly requests them.
- Before changing a policy, verify its target branches and enforcement scope.

## 6. Environmental Isolation
- Use ./tmp for temporary files.
- Do not write to system-level directories.

## 7. Context-Mode Routing
- Prefer context-mode MCP tools for large analysis.

## 8. Development Cheat Sheet

Quick reference for common efctl operations:

- **Initialize configuration**: efctl init (or efctl init --ai [agent])
- **Environment Lifecycle**:
  - Up: efctl env up
  - Down: efctl env down
- **Status Check**: efctl env status
- **Deploy Extension**: efctl env extension publish [contract-path] (path defaults to ./my-extension)
- **Query World**: efctl world query [object_id] (queries the Sui GraphQL RPC)
<!-- EFCTL_INSTRUCTIONS_END -->
Manual Agent Note
