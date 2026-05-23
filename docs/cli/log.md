---
layout: default
title: lore log
parent: CLI Reference
nav_order: 2
---
# lore log / glh log

List blobs and annotated git history.

## Synopsis

```
lore log [-n N] [--all] [--all-files] [--json]
glh log [git-flags]
```

## lore log

Lists blobs newest-first, like `git log`. Each line shows the blob's short ID, title, kind, date, trust level, and file count.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-n N` | 20 | Show at most N blobs |
| `--all` | false | Show a file-explorer tree grouped by node |
| `--all-files` | false | Verbose dump: all files and agent actions per blob |
| `--json` | false | Machine-readable JSON output |

### Default Output

```
$ lore log

abc1234  OAuth Provider Impl        Feature        2026-05-20  [AgentTruth]    3 files
def5678  Fix JWT expiry             BugFix         2026-05-18  [LoreInferred]  2 files
ghi9012  Auth middleware refactor   Refactor       2026-05-15  [AgentTruth]    5 files
jkl3456  Billing processor update   Feature        2026-05-12  [AgentTruth]    4 files
mno7890  Session expiry bug         BugFix         2026-05-10  [LoreInferred]  1 file
```

Trust level is color-coded on a TTY:
- `[AgentTruth]` — default or green (high confidence)
- `[LoreInferred]` — yellow or dim (inferred from tasks)

### --all Tree View

Groups all blobs under their assigned nodes and shows unassigned blobs at the bottom. Useful for understanding what engineering work exists across the repository.

```
$ lore log --all

Authentication
├── OAuth Provider Impl          Feature   2026-05-20  [AgentTruth]
├── Auth middleware refactor      Refactor  2026-05-15  [AgentTruth]
└── Fix JWT expiry               BugFix    2026-05-18  [LoreInferred]

Billing
├── Billing processor update     Feature   2026-05-12  [AgentTruth]
└── Stripe webhook handler       Feature   2026-05-05  [AgentTruth]

Unassigned
└── Session expiry bug           BugFix    2026-05-10  [LoreInferred]
    hint: use 'lore assign mno7890 <node>' to assign
```

### --all-files Verbose View

Shows every file and agent action per blob. Use this to understand the full scope of each unit of work.

```
$ lore log --all-files

abc1234  OAuth Provider Impl  Feature  2026-05-20  [AgentTruth]
  Files Modified:
    internal/auth/oauth.go
    internal/session/manager.go
  Files Deleted:
    internal/auth/token_legacy.go
  Commands:
    go test ./internal/auth/...
    go test ./...
  Commits: abc100..abc123

def5678  Fix JWT expiry  BugFix  2026-05-18  [LoreInferred]
  Files Modified:
    internal/auth/jwt.go
    internal/auth/jwt_test.go
  Commits: def789
```

### --json Output

All query commands support `--json` for machine-readable output. Each blob is a JSON object with all fields.

```
$ lore log --json

[
  {
    "id": "abc1234abcdef1234abcdef1234abcdef12345678",
    "title": "OAuth Provider Impl",
    "kind": "Feature",
    "trust_level": 2,
    "ai_source": "agent:claude",
    "started_at": "2026-05-20T09:14:00Z",
    "ended_at": "2026-05-20T16:42:00Z",
    "commit_start": "abc100",
    "commit_end": "abc123",
    "file_count": 3
  }
]
```

## glh log

`glh log` is a wrapper around `git log` that annotates commit lines with a `●` marker when a Lore blob was extracted from that commit.

All git log flags are passed through unchanged.

### Usage

```
glh log
glh log --oneline
glh log -n 20
glh log --since=2026-05-01
glh log --author="nishchay"
```

### Annotated Output

```
$ glh log --oneline

● abc123  feat: implement OAuth provider flow
  def456  chore: update dependencies
● ghi789  fix: correct JWT expiry calculation
  jkl012  docs: update README
● mno345  refactor: extract auth middleware
```

The `●` symbol appears next to commits that have an associated Lore blob. Commits without a blob (e.g., dependency updates, doc fixes) show no marker.

### Standard git log

With no special flags, `glh log` outputs standard git log format with blob annotations interspersed:

```
$ glh log

● commit abc123 (HEAD -> main, origin/main)
  Author: Nishchay Bhatt <nishchay@hirobin.ai>
  Date:   Tue May 20 16:42:01 2026

      feat: implement OAuth provider flow

  [lore: abc1234 — OAuth Provider Impl · Feature · AgentTruth]

  commit def456
  Author: Nishchay Bhatt <nishchay@hirobin.ai>
  Date:   Mon May 19 11:05:33 2026

      chore: update dependencies
```

## See Also

- [`lore show`](./show) — full detail for a single blob
- [`lore why`](./why) — why a specific file exists
- [`glh`](./glh) — full glh reference
