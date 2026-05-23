---
id: graph
title: Knowledge Graph
sidebar_label: Knowledge Graph
sidebar_position: 4
---

# Knowledge Graph

The knowledge graph is a navigational index derived entirely from Blobs and Nodes. It represents the relationships between subsystems, work units, files, commits, and concepts — and it is always rebuilt from that source data, never manually authored.

**The graph is a consequence of good Blobs and Nodes. It is not the product itself.**

If you deleted the graph tables and ran `lore rebuild-graph`, you would get an identical result. The Blobs and Nodes are the durable state. The graph is a derived view over them.

## What the Graph Contains

The graph has five node types and eight edge types.

### Node Types

| Kind | Source | Example |
|------|--------|---------|
| `Subsystem` | Nodes (Tier 3) | "Authentication", "Billing" |
| `Blob` | Blobs (Tier 2) | "OAuth provider implementation" |
| `File` | `blob_files` rows | `internal/auth/oauth.go` |
| `Commit` | Blob commit fields | `abc1234` |
| `Concept` | Blob and Node tags | `auth`, `oauth`, `middleware` |

Tasks are **not** graph nodes. Tasks are ephemeral. The graph only contains permanent data.

### Edge Types

| Relation | Direction | Source |
|----------|-----------|--------|
| `Contains` | Subsystem → Blob | `node_blobs` join |
| `Modified` | Blob → File | `blob_files` where `role=written` |
| `Deleted` | Blob → File | `blob_files` where `role=deleted` |
| `Produced` | Blob → Commit | `blobs.commit_start` / `commit_end` |
| `RelatedTo` | Subsystem ↔ Subsystem | Subsystems sharing Blobs or Concepts |
| `CausedBy` | Blob → Blob | Agent-provided or inferred causal link |
| `Involves` | Subsystem → Concept | Node tags |
| `PartOf` | Subsystem → Subsystem | Hierarchical nesting (post-MVP) |

Duplicate edges are not created — adding the same relationship a second time increments the edge's `weight` instead.

## How to Read It

```bash
lore graph
```

```
Subsystem: Authentication
├── Blob: OAuth provider implementation    (Feature,       2026-05-20)  [AgentTruth]
│   ├── Modified  internal/auth/oauth.go
│   ├── Modified  internal/session/manager.go
│   └── Produced  abc1234
├── Blob: Fix JWT token expiry edge case   (BugFix,        2026-05-18)  [LoreInferred]
│   ├── Modified  internal/auth/jwt.go
│   └── Produced  def5678
└── Blob: Auth middleware refactor         (Refactor,      2026-05-15)  [AgentTruth]
    ├── Modified  internal/auth/middleware.go
    ├── Modified  internal/session/manager.go
    └── Produced  c91a447

Subsystem: Billing
└── Blob: Payment processor update        (Feature,       2026-05-01)  [AgentTruth]
    ├── Modified  internal/billing/processor.go
    ├── Modified  internal/billing/processor_test.go
    └── Produced  ghi9012

Unassigned Blobs:
└── Auth service investigation            (Investigation, 2026-04-10)  [LoreInferred]
    ├── Modified  internal/auth/session.go
    └── Produced  abc000
    hint: use 'lore assign abc000 Authentication' to assign this blob
```

**Reading the tree:**
- Top-level entries are Subsystems (Nodes)
- Second level shows the 3 most recent Blobs in each Subsystem
- Each Blob shows its Modified files and Produced commits
- Trust level labels (`[AgentTruth]`, `[LoreInferred]`) appear on every Blob
- Unassigned Blobs are listed separately at the bottom

## How the Graph Is Used

Most graph traversal happens implicitly through Lore's query commands. You rarely interact with the graph directly.

| Command | Graph traversal |
|---------|----------------|
| `lore why <file>` | File node → incoming `Modified` edges → Blob nodes |
| `lore trace <file>` | Same, ordered chronologically |
| `lore show <id>` | Blob node → outgoing edges → Files, Commits, Subsystem |
| `lore node show <name>` | Subsystem node → `Contains` edges → Blob nodes → File frequency |
| `lore graph` | Full Subsystem → Blob → File tree, default depth |

## Graph Storage

The graph is stored in two SQLite tables in `.lore/lore.db`:

```sql
graph_nodes (id, kind, ref_id, label, properties)
graph_edges (id, from_id, to_id, relation, weight, properties)
```

This is an adjacency list representation. It is sufficient for repositories with thousands of Blobs and tens of thousands of edges. No graph database is used — SQLite is correct and sufficient.

The graph is rebuilt incrementally as Blobs are created. A full rebuild from scratch is deterministic.

## What the Graph Is Not

- **Not the product.** The value of Lore is in the Blobs and Nodes. The graph is navigation.
- **Not a visualization platform.** The ASCII tree in `lore graph` is a navigation aid, not a showpiece.
- **Not user-authored.** You never create graph nodes or edges directly.
- **Not a separate database.** Everything lives in the same `.lore/lore.db` SQLite file.

Advanced graph features — BFS/DFS traversal, centrality metrics, community detection — are post-MVP. The foundation is in place; the complexity is deferred until it is justified by real repository scale.
