---
id: quickstart
title: Quickstart
sidebar_label: Quickstart
sidebar_position: 3
---

# Quickstart

This guide walks you from a fresh Git repository to your first Blob in about five minutes.

## Step 1 — Initialize Lore

Navigate to an existing Git repository (or create one with `git init`), then run:

```bash
cd ~/projects/myapp
lore init
```

```
initializing lore in /Users/you/projects/myapp...
  created  .lore/lore.db
  created  .lore/config.toml
  installed  .git/hooks/post-commit
  installed  .git/hooks/post-checkout
  installed  .git/hooks/post-merge
  wrote  .claude/settings.json   (Claude Code Stop hook)
  wrote  .cursor/mcp.json        (Cursor MCP integration)
  wrote  .windsurf/mcp.json      (Windsurf MCP integration)
done. run 'lore doctor' to verify prerequisites.
```

## Step 2 — What Was Created

```
.lore/
  lore.db        SQLite database — all Tasks, Blobs, Nodes, and graph data
  config.toml    Per-repo configuration (LLM model, retention, etc.)

.git/hooks/
  post-commit    Triggers 'lore hook commit-created' after each commit
  post-checkout  Triggers 'lore hook branch-switch' on checkout
  post-merge     Triggers 'lore hook merge-event' after merges

.claude/settings.json   Registers 'lore hook agent-recap agent:claude' as a Stop hook
.cursor/mcp.json        Registers Lore as a Cursor MCP server
.windsurf/mcp.json      Registers Lore as a Windsurf MCP server
```

The Git hooks and agent integrations are what make observation automatic. You do not need to run any command before or after working — Lore captures activity in the background.

## Step 3 — Make Changes and Commit

Write some code and commit it:

```bash
# make some changes
echo 'package auth' > internal/auth/oauth.go
go test ./internal/auth/...
git add .
git commit -m "feat: add OAuth provider flow"
```

```
[main abc1234] feat: add OAuth provider flow
 3 files changed, 87 insertions(+), 12 deletions(-)

lore: extracting blob from 4 tasks...
lore: blob a3f1c82 created [LoreInferred, heuristic]
```

Lore ran its `post-commit` hook, gathered the Tasks accumulated since the last extraction, and created your first Blob.

## Step 4 — View Your Blobs

```bash
lore log
```

```
a3f1c82  Add OAuth provider flow   Feature   2026-05-20  [LoreInferred]  3 files
```

The `[LoreInferred]` label means Lore used heuristic inference (no Ollama, or no agent recap available). If you are using Claude Code with `lore init` having written the Stop hook, subsequent commits will show `[AgentTruth]` instead.

## Step 5 — Inspect a Blob

Copy the short ID from `lore log` and run:

```bash
lore show a3f1c82
```

```
ID:           a3f1c82
Title:        Add OAuth provider flow
Kind:         Feature
Trust:        LoreInferred (source: lore:heuristic)

── Observed ────────────────────────────────────────
Started:      2026-05-20 14:03
Ended:        2026-05-20 14:31
Commits:      abc1234

Files Modified:
  internal/auth/oauth.go
  internal/auth/oauth_test.go
  internal/session/manager.go

Commands:
  go test ./internal/auth/...

── Interpreted ─────────────────────────────────────
User Intent:  Add OAuth provider flow
Summary:      Modified 3 files. Ran 1 command. Produced 1 commit.
Tags:         auth, session

── Part of ─────────────────────────────────────────
Node: (unassigned)
  hint: use 'lore node create <name>' then 'lore assign a3f1c82 <node>'
```

The `── Observed ──` section contains only deterministic facts Lore read directly from hooks and Git. The `── Interpreted ──` section contains AI-generated or heuristic-derived meaning. They are always kept separate.

## Step 6 — Ask Why a File Exists

```bash
lore why internal/auth/oauth.go
```

```
Add OAuth provider flow (Feature)  2026-05-20  [LoreInferred]
  Modified 3 files. Ran 1 command. Produced 1 commit.
  Commits: abc1234
  Node: (unassigned)
```

As you accumulate more work, `lore why` becomes a timeline of every Blob that touched a file — the closest thing to `git blame` for intent rather than authorship.

## Next Steps

- Install Ollama for richer AI summaries: `ollama pull llama3`
- Create a Node to group related Blobs: `lore node create "Authentication"`
- Assign the Blob: `lore assign a3f1c82 Authentication`
- Explore the knowledge graph: `lore graph`
- Use `glh commit` instead of `git commit` to bypass the hook delay and capture commands more precisely
