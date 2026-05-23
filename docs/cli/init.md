---
sidebar_position: 1
---

# lore init

Initialize a repository for Lore and install all hooks.

## Synopsis

```
lore init
```

## Description

`lore init` sets up Lore in the current Git repository. It creates the `.lore/` directory, initializes the SQLite database, installs Git hooks, and writes agent integration configs.

Run it once after cloning or creating a repository. The command is idempotent — it is safe to re-run at any time. Re-running will not overwrite existing blob data, but will repair missing hooks, scripts, or config files.

## What It Does

`lore init` performs the following steps in order:

1. **Verifies Git repository** — confirms the current directory is inside a Git repository. Exits with code 128 if not.

2. **Creates `.lore/`** — creates the hidden directory that holds all Lore state:
   ```
   .lore/
   ├── lore.db          # SQLite database (tasks, blobs, nodes, graph)
   ├── hooks/           # Git hook scripts
   │   ├── post-commit
   │   └── post-checkout
   └── config.toml      # Lore configuration (LLM, retention, etc.)
   ```

3. **Configures Git hooks path** — runs `git config core.hooksPath .lore/hooks` so Git uses Lore's hook directory instead of `.git/hooks/`. This is how Lore intercepts commits and branch operations without modifying `.git/` directly.

4. **Writes hook scripts** — creates executable shell scripts in `.lore/hooks/`:
   - `post-commit` — calls `lore hook commit-created` after each commit
   - `post-checkout` — calls `lore hook branch-switch` after each checkout or switch

5. **Writes agent configs** — creates integration configs for installed AI tools:
   - `.claude/settings.json` — adds a `Stop` hook (`lore hook agent-recap agent:claude`) and MCP server registration for Claude Code
   - `cursor_mcp.json` / `.cursorrules` — Cursor MCP registration (if Cursor detected)
   - Windsurf config (if Windsurf detected)

6. **Initializes the database** — runs all schema migrations on `lore.db`, creating the full table set if they do not exist.

7. **Records init task** — inserts a `RepoInit` task into the database with the repository path and current timestamp.

## Files Created or Modified

| Path | Action | Description |
|------|--------|-------------|
| `.lore/` | Created | Lore state directory |
| `.lore/lore.db` | Created | SQLite database |
| `.lore/hooks/post-commit` | Created | Git post-commit hook script |
| `.lore/hooks/post-checkout` | Created | Git post-checkout hook script |
| `.lore/config.toml` | Created | Lore configuration file |
| `.claude/settings.json` | Modified | Claude Code hook + MCP registration |
| `git config core.hooksPath` | Set | Points Git at `.lore/hooks/` |

## Idempotency

Re-running `lore init` on an already-initialized repository is safe:

- Existing blob and node data is never touched.
- Missing hook scripts are recreated.
- The database schema is migrated forward if needed.
- Agent config entries are added only if not already present.
- `core.hooksPath` is re-confirmed.

## Example Output

```
$ lore init

  Initializing Lore in /Users/nishchay/projects/myapp

  Created   .lore/
  Created   .lore/lore.db
  Created   .lore/hooks/post-commit
  Created   .lore/hooks/post-checkout
  Created   .lore/config.toml
  Set       git config core.hooksPath = .lore/hooks
  Updated   .claude/settings.json (Stop hook, MCP server)

  Lore initialized. Git activity will be captured automatically.

  hint: run 'lore doctor' to verify all hooks and integrations are working
  hint: run 'lore status' to see repository state
```

## Error Cases

```
error: not a git repository (or any parent up to mount point /)
hint: run 'git init' first, then run 'lore init'
```

```
error: .lore/ already exists but lore.db is missing or corrupted
hint: run 'lore init' again to repair, or remove .lore/ and re-initialize
```

## See Also

- [`lore doctor`](../cli/doctor) — verify hooks and integrations after init
- [`lore status`](../cli/status) — see current repository state
