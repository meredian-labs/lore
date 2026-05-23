---
sidebar_position: 4
---

# lore why

Show all blobs that have modified a file, newest first.

## Synopsis

```
lore why <file>
lore why <file> [--json]
```

## Description

`lore why` answers the question: _why does this file exist?_ It shows every blob that has modified the given file, newest first. Each result includes the blob's title, kind, date, trust level, a one-line summary, and the commit range.

Use `lore why` when reading unfamiliar code and you want to understand the engineering context behind it.

## How It Works

`lore why` queries the `blob_files` table, joining against `blobs`, and filters by the given path:

```sql
SELECT b.*
FROM blobs b
JOIN blob_files bf ON b.id = bf.blob_id
WHERE bf.path = ?
ORDER BY b.started_at DESC;
```

The file path is matched exactly. You can pass a relative path from anywhere inside the repository and Lore resolves it to the canonical repository-relative path.

## Usage

```
$ lore why internal/auth/oauth.go
$ lore why cmd/lore/main.go
$ lore why go.mod
```

## Output Format

```
$ lore why internal/auth/oauth.go

OAuth Provider Implementation  (Feature)   2026-05-20  [AgentTruth]
  Replaced legacy token auth with OAuth2 provider flow.
  Commits: abc100..abc123
  Node: Authentication

Fix JWT expiry  (BugFix)  2026-05-18  [LoreInferred]
  Corrected token expiry calculation in oauth.go token validation.
  Commits: def789
  Node: Authentication

Auth middleware refactor  (Refactor)  2026-05-15  [AgentTruth]
  Extracted authentication middleware from handler chain.
  Commits: ghi001..ghi005
  Node: Authentication
```

Each entry shows:
- **Title** and **Kind** — what category of work it was
- **Date** — when the work happened
- **Trust** — whether interpretation came from an agent recap or Lore inference
- **Summary** — one-sentence description of what happened
- **Commits** — the commit or commit range that produced the blob
- **Node** — the subsystem this blob belongs to (if assigned)

## No Results

```
$ lore why internal/auth/new_file.go

error: no blobs found for 'internal/auth/new_file.go'
hint: run 'lore status' to see what lore has captured
hint: this file may have been created before lore init, or may not yet be part of a committed blob
```

## JSON Output

```
$ lore why internal/auth/oauth.go --json

[
  {
    "id": "abc1234abcdef...",
    "title": "OAuth Provider Implementation",
    "kind": "Feature",
    "trust_level": 2,
    "ai_source": "agent:claude",
    "summary": "Replaced legacy token auth with OAuth2 provider flow.",
    "started_at": "2026-05-20T09:14:00Z",
    "commit_start": "abc100",
    "commit_end": "abc123",
    "primary_node": "Authentication"
  }
]
```

## Notes

- `lore why` shows blobs where the file appears as `role=written` or `role=deleted` in `blob_files`. A file deletion appears as its own entry.
- If a file was renamed, both the old path and the new path may appear in separate blob entries depending on how the rename was recorded.
- For a chronological view (oldest first), use [`lore trace`](./trace).

## See Also

- [`lore trace`](./trace) — chronological file history (oldest first)
- [`lore show`](./show) — full blob detail
- [`lore log`](./log) — list all blobs
