---
title: lore doctor
---

# lore doctor

Check your lore installation and diagnose configuration issues.

## Usage

```bash
lore doctor
```

## Output

```
✓  Git repository          /Users/nishchay/projects/myapp
✓  Lore initialized        .lore/  (since 2026-05-10)
✓  Git hooks wired         core.hooksPath = .lore/hooks
✓  Hook scripts            post-commit, post-checkout, post-merge
✓  Claude Code hooks       PostToolUse (Edit/Write/Read/Bash) + Stop
✓  Claude Code MCP         lore mcp agent:claude
✓  Cursor MCP              .cursor/mcp.json
✓  Windsurf MCP            .windsurf/mcp.json
⚠  LLM                     not reachable — ollama/llama3 (http://localhost:11434)

Blobs: 13  (6 Feature, 2 Refactor, 1 BugFix)
Nodes: 5
Pending tasks: 0

1 warning(s) — run 'lore init' to fix missing configs
```

## Checks

| # | Check | Pass condition | Failure |
|---|-------|----------------|---------|
| 1 | Git repository | Inside a git repo | ✗ critical |
| 2 | Lore initialized | `.lore/` directory exists | ✗ critical |
| 3 | Git hooks wired | `git config core.hooksPath` == `.lore/hooks` | ✗ critical |
| 4 | Hook scripts | `post-commit`, `post-checkout`, `post-merge` exist with lore header | ✗ critical |
| 5 | Claude Code hooks | `.claude/settings.json` has `hooks.Stop` with `lore hook agent-recap` | ⚠ warning |
| 6 | Claude Code MCP | `.claude/settings.json` has `mcpServers.lore` | ⚠ warning |
| 7 | Cursor MCP | `.cursor/mcp.json` has `mcpServers.lore` | ⚠ warning |
| 8 | Windsurf MCP | `.windsurf/mcp.json` has `mcpServers.lore` | ⚠ warning |
| 9 | LLM | Configured endpoint responds to a ping | ⚠ warning |

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | All checks passed (warnings OK) |
| 1 | One or more critical checks failed |

## Fixing failures

All critical failures and most warnings are fixed by re-running `lore init`:

```bash
lore init
```

`lore init` is idempotent — safe to run multiple times. It only adds missing configuration and never overwrites settings you have customized.

The LLM warning is resolved by [installing Ollama](https://ollama.com) and pulling a model:

```bash
ollama pull llama3
```

If you don't want local AI inference, the LLM warning can be ignored — lore will fall back to heuristic extraction for all blobs.

## See also

- [`lore init`](init.md) — initialize lore in a repository
- [`lore status`](status.md) — runtime repository state
