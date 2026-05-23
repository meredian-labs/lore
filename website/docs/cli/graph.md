# lore graph

Show the ASCII knowledge graph.

## Synopsis

```
lore graph
lore graph [--json]
```

## Description

`lore graph` renders the repository's knowledge graph as an ASCII tree. It shows subsystem nodes, the blobs they contain, and the files each blob modified or produced.

The graph is a navigational view of the repository's engineering history — not a visualization product. It is derived entirely from blobs and nodes. No graph data is manually authored.

## Output Format

```
$ lore graph

Subsystem: Authentication
├── Blob: OAuth Provider Implementation   (Feature,        2026-05-20)  [AgentTruth]
│   ├── Modified  internal/auth/oauth.go
│   ├── Modified  internal/session/manager.go
│   └── Produced  abc123
├── Blob: Fix JWT expiry                  (BugFix,         2026-05-18)  [LoreInferred]
│   ├── Modified  internal/auth/jwt.go
│   └── Produced  def789
└── Blob: Auth middleware refactor        (Refactor,       2026-05-15)  [AgentTruth]
    ├── Modified  internal/auth/middleware.go
    ├── Modified  internal/auth/middleware_test.go
    └── Produced  ghi005

Subsystem: Billing
└── Blob: Billing processor update        (Feature,        2026-05-12)  [AgentTruth]
    ├── Modified  internal/billing/processor.go
    ├── Modified  internal/billing/processor_test.go
    └── Produced  jkl100

Unassigned Blobs:
└── Session expiry bug                    (BugFix,         2026-05-10)  [LoreInferred]
    ├── Modified  internal/session/session.go
    └── Produced  mno345
    hint: use 'lore assign mno345 <node>' to assign this blob
```

## What Is Shown

**Default scope:**
- All subsystem nodes (Tier 3)
- The 3 most recent blobs per node
- Up to 5 unassigned blobs
- Files listed per blob (modified and deleted)
- Commit SHA produced per blob

**Ordering:**
- Nodes are shown in order of most recent blob activity
- Blobs within a node are shown newest first
- Files within a blob are shown in the order they were recorded

## Tree Symbols

| Symbol | Meaning |
|--------|---------|
| `├──` | Item with more siblings following |
| `└──` | Last item in the list |
| `│` | Vertical connector for nested items |
| `[AgentTruth]` | Interpreted by the agent that did the work |
| `[LoreInferred]` | Interpreted by Lore's local AI or heuristics |

## Edge Types Shown

| Label | Meaning |
|-------|---------|
| `Modified` | Blob wrote to this file |
| `Deleted` | Blob deleted this file |
| `Produced` | Blob was extracted from this commit |

## JSON Output

```
$ lore graph --json

{
  "nodes": [
    {
      "id": "node-uuid",
      "name": "Authentication",
      "blob_count": 3,
      "blobs": [
        {
          "id": "abc1234...",
          "title": "OAuth Provider Implementation",
          "kind": "Feature",
          "trust_level": 2,
          "date": "2026-05-20",
          "files_modified": ["internal/auth/oauth.go", "internal/session/manager.go"],
          "commit_end": "abc123"
        }
      ]
    }
  ],
  "unassigned_blobs": [
    {
      "id": "mno7890...",
      "title": "Session expiry bug",
      "kind": "BugFix",
      "trust_level": 4
    }
  ]
}
```

## Notes

- The graph is fully derived from blobs and nodes. It is rebuilt automatically as new blobs are extracted.
- The graph is stored in `graph_nodes` and `graph_edges` SQLite tables (adjacency list). No graph database is used.
- Interactive TUI navigation (`bubbletea`) is deferred to post-MVP.

## See Also

- [`lore log --all`](./log.md) — flat tree view grouped by node
- [`lore why`](./why.md) — all blobs for a specific file
- [`lore node list`](../cli/node.md) — list all subsystem nodes
- [Architecture: Knowledge Graph](../architecture/storage.md) — graph tier explanation
