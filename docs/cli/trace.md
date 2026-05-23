---
sidebar_position: 5
---

# lore trace

Show the chronological history of a file across blobs.

## Synopsis

```
lore trace <file>
lore trace <file> [--json]
```

## Description

`lore trace` shows the complete history of a file, ordered oldest to newest. It is the chronological counterpart to [`lore why`](./why).

Use `lore trace` when you want to follow the evolution of a file from its first appearance to today — understanding how and why it was shaped over time.

## Difference from lore why

| Command | Order | Use case |
|---------|-------|----------|
| `lore why <file>` | Newest first | "What most recently changed this file and why?" |
| `lore trace <file>` | Oldest first (chronological) | "How did this file evolve from the beginning?" |

Both commands query the same data — `blob_files` joined to `blobs`. The only difference is sort order:

- `lore why` uses `ORDER BY b.started_at DESC`
- `lore trace` uses `ORDER BY b.started_at ASC`

## Usage

```
$ lore trace internal/auth/oauth.go
$ lore trace cmd/lore/main.go
$ lore trace go.mod
```

## Output Format

```
$ lore trace internal/auth/oauth.go

Auth middleware refactor  (Refactor)  2026-05-15  [AgentTruth]
  Extracted authentication middleware from handler chain.
  Commits: ghi001..ghi005
  Node: Authentication

Fix JWT expiry  (BugFix)  2026-05-18  [LoreInferred]
  Corrected token expiry calculation in oauth.go token validation.
  Commits: def789
  Node: Authentication

OAuth Provider Implementation  (Feature)  2026-05-20  [AgentTruth]
  Replaced legacy token auth with OAuth2 provider flow.
  Commits: abc100..abc123
  Node: Authentication
```

Reading top-to-bottom shows the file's full story: it started as part of a refactor, had a bug fixed, then was overhauled for the OAuth migration.

## No Results

```
$ lore trace internal/auth/new_file.go

error: no blobs found for 'internal/auth/new_file.go'
hint: run 'lore status' to see what lore has captured
hint: this file may have been created before lore init, or may not yet be part of a committed blob
```

## JSON Output

```
$ lore trace internal/auth/oauth.go --json

[
  {
    "id": "ghi9012abcdef...",
    "title": "Auth middleware refactor",
    "kind": "Refactor",
    "trust_level": 2,
    "ai_source": "agent:claude",
    "summary": "Extracted authentication middleware from handler chain.",
    "started_at": "2026-05-15T10:00:00Z",
    "commit_start": "ghi001",
    "commit_end": "ghi005",
    "primary_node": "Authentication"
  },
  {
    "id": "def5678abcdef...",
    "title": "Fix JWT expiry",
    "kind": "BugFix",
    "trust_level": 4,
    "ai_source": "lore:ollama",
    "summary": "Corrected token expiry calculation in oauth.go token validation.",
    "started_at": "2026-05-18T14:22:00Z",
    "commit_start": "def789",
    "commit_end": "def789",
    "primary_node": "Authentication"
  }
]
```

## Notes

- `lore trace` includes blobs where the file appears as `role=written` or `role=deleted`. A deletion entry marks the end of the file's active life in the repository.
- If the same file appears in multiple blobs in close succession, this typically indicates iterative development within a short period. Each blob is a separate extraction window.
- The file path is matched as a repository-relative path. Pass a path relative to the repository root or relative to your current directory — Lore resolves both.

## See Also

- [`lore why`](./why) — same data, newest first
- [`lore show`](./show) — full blob detail
- [`lore graph`](./graph) — knowledge graph including file relationships
