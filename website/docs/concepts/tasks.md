---
title: Tasks
---

# Tasks

Tasks are the raw signal of Lore. They are atomic, observable development actions — deterministic facts about what happened — collected automatically as you work.

**Tasks are internal plumbing. Users never interact with them directly.**

## What a Task Is

A Task records one thing that happened at one point in time:

- A file was written
- A commit was created
- A command was executed
- An AI agent finished its work and left a recap

Every Task has:

| Field | Description |
|-------|-------------|
| `id` | UUID |
| `kind` | What type of action this was (see below) |
| `occurred_at` | When it happened |
| `source` | Where the signal came from (`hook`, `agent:claude`, `human`, `ci`) |
| `trust_level` | How reliable the observation is (1=GroundTruth, 2=AgentTruth) |
| `path` | File path, if applicable |
| `detail` | Kind-specific payload (commit SHA, command string, JSON recap, etc.) |

## Task Kinds

| Kind | Collected By | Description |
|------|-------------|-------------|
| `FileWrite` | Agent hooks, glh | A file was created or modified |
| `FileDelete` | Agent hooks, glh | A file was deleted |
| `FileRename` | Agent hooks, glh | A file was moved or renamed |
| `FileRead` | — | Excluded from MVP (too noisy, low signal value) |
| `Command` | Agent hooks, glh | A shell command was executed |
| `CommitCreated` | `post-commit` Git hook | A commit was recorded by Git |
| `BranchSwitch` | `post-checkout` Git hook | The active branch changed |
| `MergeEvent` | `post-merge` Git hook | A branch was merged |
| `SearchQuery` | Agent hooks | A search was performed in the editor or codebase |
| `AgentAction` | Agent hooks | A discrete action taken by an AI agent |
| `Note` | `lore record` command | A developer annotation |
| `AgentRecap` | Agent Stop hooks | Structured summary provided by an AI agent |

## How Tasks Are Collected

Tasks enter the system through three paths:

### Git Hooks

`lore init` installs three Git hooks into `.git/hooks/`:

```
post-commit    → lore hook commit-created
post-checkout  → lore hook branch-switch
post-merge     → lore hook merge-event
```

These are invoked by Git automatically and emit the corresponding Task kinds. No developer action required.

### Agent Hooks

For Claude Code, `lore init` writes a Stop hook to `.claude/settings.json`:

```json
{
  "hooks": {
    "Stop": [
      {
        "command": "lore hook agent-recap agent:claude"
      }
    ]
  }
}
```

For Cursor and Windsurf, the MCP integration files (`.cursor/mcp.json`, `.windsurf/mcp.json`) expose `lore hook` as an MCP tool the agent calls when appropriate.

When an agent emits file writes or commands, those appear as `FileWrite`, `FileDelete`, and `Command` Tasks with `source=agent:claude` (or the relevant agent). When the agent finishes a unit of work, it emits an `AgentRecap` Task.

### Manual Notes

```bash
lore record "investigating JWT expiry regression — suspect clock skew"
```

This emits a `Note` Task that is included in the next Blob extraction window, giving the developer's own words to the AI when it generates the Blob's interpretation.

### glh (the Git wrapper)

When you use `glh` instead of `git`, the binary intercepts the invocation before passing it to Git and emits a `Command` Task for the full command string. This gives Lore richer command history than Git hooks alone can provide.

## Ephemeral by Design

Tasks are temporary. They exist to drive Blob extraction — once a Blob has absorbed a window of Tasks, those Tasks are no longer needed.

Default retention:
- **30 days** after a Task has been absorbed into a Blob
- **90 days** unconditionally

This means the `tasks` table stays small regardless of repository age. The knowledge lives in Blobs, not in the raw task stream.

## Why Tasks Are Not User-Facing

Tasks are the equivalent of Git's object store: the mechanism, not the product. Exposing them would surface a stream of low-level noise with no useful context.

`lore tasks` is not a valid command. Users interact with Blobs — the compressed, interpreted result of Task extraction.
