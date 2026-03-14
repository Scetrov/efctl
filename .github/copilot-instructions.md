# context-mode routing rules for efctl

This workspace is configured with context-mode. Prefer the MCP tools over raw terminal, read, grep, and fetch operations when the output could be large.

## Hard routing rules

- Do not use terminal commands with `curl` or `wget`.
- Do not use direct web fetching tools for external pages.
- Do not use terminal commands that are likely to dump large output when a context-mode tool can do the same work.

Preferred replacements:

- `ctx_fetch_and_index(url, source)` for web pages and remote docs
- `ctx_batch_execute(commands, queries)` for multi-step repo research
- `ctx_execute(language, code)` for controlled shell or script execution
- `ctx_execute_file(path, language, code)` for analyzing files without dumping raw content into context
- `ctx_search(queries)` for follow-up retrieval from indexed data

## When normal tools are still correct

- Use file reads when the file content is needed for a direct edit.
- Use terminal for short operational commands such as `git`, `mkdir`, `rm`, `mv`, `cd`, `ls`, and install commands.

## Working style

- Keep final responses short.
- Write artifacts to files instead of returning long inline blocks.
- Use clear source labels when indexing content so future searches can target them.

## Utility commands

- `ctx stats`
- `ctx doctor`
- `ctx upgrade`
