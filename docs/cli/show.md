---
layout: default
title: lore show
parent: CLI Reference
nav_order: 3
---
# lore show

Show full detail for a single blob.

## Synopsis

```
lore show <id>
lore show <id> [--json]
```

## Description

`lore show` displays the complete record for a single blob, with observed facts and AI-interpreted fields in visually distinct sections. It is the primary way to read the full context for a unit of engineering work.

## ID Matching

Blob IDs are full SHA-like strings (e.g., `abc1234abcdef1234abcdef1234abcdef12345678`). You do not need to type the full ID.

**Minimum prefix: 7 characters.** If the prefix matches exactly one blob, that blob is shown. If it matches more than one, Lore lists the matches and asks you to be more specific.

```
$ lore show abc1234
```

```
$ lore show abc1234abcdef12
```

Both work, as long as the prefix is unambiguous.

## Output Format

The output is divided into two clearly separated sections: **Observed** and **Interpreted**.

- **Observed** fields are facts Lore recorded deterministically from Git and task data. They cannot be wrong.
- **Interpreted** fields are AI-generated text (from an agent recap or Lore's fallback inference). They carry a trust level.

```
$ lore show abc1234

ID:           abc1234abcdef1234abcdef1234abcdef12345678
Title:        OAuth Provider Implementation
Kind:         Feature
Trust:        AgentTruth (source: agent:claude)

── Observed ────────────────────────────────────────────────────
Started:      2026-05-20 09:14
Ended:        2026-05-20 16:42
Commits:      abc100..abc123

Files Modified:
  internal/auth/oauth.go
  internal/session/manager.go

Files Deleted:
  internal/auth/token_legacy.go

Commands:
  go test ./internal/auth/...
  go test ./...

── Interpreted ─────────────────────────────────────────────────
User Intent:  Add Google OAuth support to replace legacy token auth
Summary:      Implemented OAuth2 provider flow and callback handling.
              Session integration updated to use provider tokens.
Recap:        Authentication subsystem migrated to provider-based login.
              This eliminates maintenance of the legacy token system.
Tags:         auth, oauth, session, provider

── Part of ─────────────────────────────────────────────────────
Node:         Authentication (assigned by: human)
```

## Fields Reference

### Header

| Field | Description |
|-------|-------------|
| `ID` | Full blob ID |
| `Title` | Short name for the work (AI-generated) |
| `Kind` | Classification: Feature, BugFix, Refactor, Migration, Investigation, Architecture, Review, Incident |
| `Trust` | Trust level and AI source |

### Observed Section

| Field | Description |
|-------|-------------|
| `Started` | Timestamp of the first task in the extraction window |
| `Ended` | Timestamp of the last task in the extraction window |
| `Commits` | Commit SHA range. Single commit shown as one SHA. |
| `Files Modified` | Files written during the window (from `blob_files` where role=written) |
| `Files Deleted` | Files deleted during the window (from `blob_files` where role=deleted) |
| `Commands` | Terminal commands run during the window (from `blob_commands`) |

### Interpreted Section

| Field | Description |
|-------|-------------|
| `User Intent` | What the developer or agent was trying to accomplish |
| `Summary` | 2–5 sentences describing what was done |
| `Recap` | 1–3 sentences on why it matters in the bigger picture |
| `Tags` | Domain concepts extracted from the work |

When trust level is `LoreInferred`, an additional field may appear:

| Field | Description |
|-------|-------------|
| `Inferred Reasoning` | Lore's reasoning process when no agent recap was available |

### Part Of Section

Shows the Node (subsystem) this blob is assigned to, if any. If unassigned:

```
── Part of ─────────────────────────────────────────────────────
Node:         (unassigned)
              hint: use 'lore assign abc1234 <node>' to assign
```

## Trust Level Display

| Trust | Label | Meaning |
|-------|-------|---------|
| 2 | `AgentTruth (source: agent:claude)` | AI agent that did the work provided the recap |
| 4 | `LoreInferred (source: lore:ollama)` | Lore's local AI inferred from observed tasks |
| 4 | `LoreInferred (source: lore:heuristic)` | Lore used commit message heuristics (no AI available) |

## JSON Output

```
$ lore show abc1234 --json

{
  "id": "abc1234abcdef1234abcdef1234abcdef12345678",
  "title": "OAuth Provider Implementation",
  "kind": "Feature",
  "trust_level": 2,
  "ai_source": "agent:claude",
  "started_at": "2026-05-20T09:14:00Z",
  "ended_at": "2026-05-20T16:42:00Z",
  "commit_start": "abc100",
  "commit_end": "abc123",
  "user_intent": "Add Google OAuth support to replace legacy token auth",
  "summary": "Implemented OAuth2 provider flow and callback handling...",
  "recap": "Authentication subsystem migrated to provider-based login...",
  "tags": ["auth", "oauth", "session", "provider"],
  "files_modified": ["internal/auth/oauth.go", "internal/session/manager.go"],
  "files_deleted": ["internal/auth/token_legacy.go"],
  "commands": ["go test ./internal/auth/...", "go test ./..."],
  "primary_node_id": "node-uuid-here"
}
```

## Error Cases

```
error: no blob found with id prefix 'xyz9999'
hint: run 'lore log' to see available blob IDs
```

```
error: ambiguous blob id prefix 'abc' — matches 3 blobs
hint: use a longer prefix (minimum 7 characters)
```

## See Also

- [`lore log`](./log) — list all blobs
- [`lore why`](./why) — find blobs by file
- [`lore assign`](../cli/assign) — assign a blob to a node
