# Rule: Knowledge Graph Philosophy

> Parent: [../ARCHITECTURE.md](../ARCHITECTURE.md)
> Schema: [../SCHEMA.md](../SCHEMA.md)

---

## The Core Rule

**Lore is not a graph visualization product.**

The knowledge graph is a navigational index derived from Blobs and Nodes. It is not the value proposition.

If the graph disappeared tomorrow, Lore would still be valuable because of the Blobs and Nodes.

If the Blobs and Nodes disappeared tomorrow, the graph would be meaningless.

Build Blobs first. Resolve Nodes second. The graph follows naturally from both.

---

## Graph Structure

The graph has five node types and eight edge types, derived from the four-tier pipeline.

### Node Types

| Kind | Source | Example |
|------|--------|---------|
| `Subsystem` | Nodes (Tier 3) | "Authentication", "Billing", "Session Management" |
| `Blob` | Blobs (Tier 2) | "OAuth Provider Implementation" |
| `File` | blob_files | `internal/auth/oauth.go` |
| `Commit` | Blob commit fields | `abc123` |
| `Concept` | Blob/Node tags | `auth`, `oauth` |

`Subsystem` nodes map 1:1 with Tier 3 Nodes. They represent stable repository areas, not individual tasks or migrations.

Note: Tasks (Tier 1) are **not** graph nodes. Tasks are ephemeral. The graph only contains permanent data.

### Edge Types

| Relation | From → To | Source |
|----------|-----------|--------|
| `Contains` | Topic → Blob | node_blobs join |
| `Modified` | Blob → File | blob_files where role=written |
| `Deleted` | Blob → File | blob_files where role=deleted |
| `Produced` | Blob → Commit | blobs.commit_start / commit_end |
| `RelatedTo` | Topic ↔ Topic | Topics sharing Blobs or Concepts |
| `CausedBy` | Blob → Blob | Explicit (agent-provided) or inferred |
| `Involves` | Topic → Concept | node tags |
| `PartOf` | Topic → Topic | Hierarchical nesting |

---

## What the Graph Is Not

| Not This | Because |
|----------|---------|
| The primary product | Blobs and Nodes are the product |
| A visualization showpiece | Visualization is a navigation aid, not the value |
| A separate database | SQLite adjacency tables are correct and sufficient |
| A graph database | No Neo4j, no DGraph, no custom engine |
| User-authored data | The graph is always derived; users never create nodes or edges directly |
| A reason to over-engineer | Start simple, stay simple |

---

## Graph Storage

Two tables: `graph_nodes` and `graph_edges`. Adjacency list representation.

This is sufficient for repositories with thousands of Blobs and tens of thousands of edges. If performance becomes a concern, add indexes before considering a different storage engine.

Do not introduce a graph database. Operational complexity is not justified for a local-first CLI tool.

See [../SCHEMA.md](../SCHEMA.md) for the exact table definitions.

---

## Graph Construction Rules

1. **Graphs are derived, never authored.** No user or developer manually creates graph nodes or edges.

2. **One Blob → one Blob graph node.** Blob nodes are 1:1 with Blob rows.

3. **One Node → one Topic graph node.** Topic nodes are 1:1 with Node rows.

4. **Files become graph nodes when referenced.** A File node is created for every unique path in `blob_files`.

5. **Commits become graph nodes when referenced.** A Commit node is created for every SHA in `blobs.commit_start` or `blobs.commit_end`.

6. **Concepts are derived from tags.** A Concept node is created for each unique tag across all Blobs and Nodes.

7. **Edges are deduplicated with weight.** A duplicate `Modified` edge increments `weight` instead of creating a new row.

8. **The graph is rebuildable.** Given Blobs and Nodes, the entire graph can be reconstructed. `lore rebuild-graph` (post-MVP) must demonstrate this.

9. **Task nodes do not exist.** Tasks are not in the graph. They are ephemeral.

---

## Graph Traversal in MVP

Graph traversal in MVP is SQL queries over adjacency tables. No graph algorithm library needed.

### `lore why auth.go`

```sql
SELECT b.*
FROM blobs b
JOIN blob_files bf ON b.id = bf.blob_id
WHERE bf.path = 'internal/auth.go'
ORDER BY b.started_at DESC;
```

### `lore trace auth.go`

Same query, `ORDER BY b.started_at ASC`.

### `lore show <blob-id>` — related Nodes

```sql
SELECT n.*
FROM nodes n
JOIN node_blobs nb ON n.id = nb.node_id
WHERE nb.blob_id = ?;
```

### Files co-modified with auth.go

```sql
SELECT bf2.path, COUNT(*) as co_modifications
FROM blob_files bf1
JOIN blob_files bf2 ON bf1.blob_id = bf2.blob_id
WHERE bf1.path = 'internal/auth.go'
  AND bf2.path != 'internal/auth.go'
GROUP BY bf2.path
ORDER BY co_modifications DESC
LIMIT 10;
```

### Blobs in a Node

```sql
SELECT b.*
FROM blobs b
JOIN node_blobs nb ON b.id = nb.blob_id
WHERE nb.node_id = ?
ORDER BY b.started_at ASC;
```

BFS/DFS traversal (for `Related(nodeID, depth)`) is a post-MVP feature.

---

## `lore graph` Output

MVP produces ASCII output. No interactive TUI.

```
Subsystem: Authentication
├── Blob: OAuth provider impl        (Feature,       2026-04-22)  [AgentTruth]
│   ├── Modified  internal/auth/oauth.go
│   ├── Modified  internal/session/manager.go
│   └── Produced  abc123
├── Blob: Auth system investigation  (Investigation, 2026-04-10)  [LoreInferred]
│   ├── Modified  internal/auth/jwt.go
│   └── Produced  abc000
└── Blob: OAuth test coverage        (BugFix,        2026-04-25)  [AgentTruth]
    ├── Modified  internal/auth/oauth_test.go
    └── Produced  abc456

Subsystem: Billing
└── Blob: Payment processor update   (Feature,       2026-05-01)  [AgentTruth]
    ├── Modified  internal/billing/processor.go
    └── Produced  def100

Unassigned Blobs:
└── JWT Expiry Fix  (BugFix, 2026-05-10)  [LoreInferred]
    ├── Modified  internal/auth/oauth.go
    └── Produced  def789
    hint: use 'lore assign def789 Authentication' to assign this blob
```

Default scope: all Subsystem Nodes and their 3 most recent Blobs each, plus the 5 most recent unassigned Blobs.

Interactive TUI (bubbletea) is deferred to post-MVP.

---

## Centrality and Clustering (Post-MVP)

File centrality — how often a file appears across Blobs:

```sql
SELECT path, COUNT(DISTINCT blob_id) as blob_count
FROM blob_files
GROUP BY path
ORDER BY blob_count DESC
LIMIT 20;
```

Do not implement PageRank, betweenness centrality, or community detection in MVP. The signal-to-noise ratio at typical repository sizes does not justify the complexity.
