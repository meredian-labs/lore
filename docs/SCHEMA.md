# Lore — SQLite Schema

> MVP schema for `.lore/lore.db`
>
> See [ARCHITECTURE.md](./ARCHITECTURE.md) for the four-tier pipeline this schema implements.
> See [rules/STORAGE.md](./rules/STORAGE.md) for tier retention and rules.

---

## Design Principles

1. **Tier 1 (tasks) is append-only.** No updates to task rows after insert.
2. **Tier 2 (blobs) is permanent.** Blob rows are never deleted automatically.
3. **Tier 3 (nodes) is permanent but updatable.** Node→Blob assignments may be revised as more evidence arrives.
4. **Tier 4 (graph) is derived and rebuildable.** Drop and rebuild without data loss.
5. **Observed and inferred fields are always separate columns.** Never mix Ground Truth with AI inference in the same field.
6. **Every interpreted field carries its trust level and AI source.** Provenance is mandatory.
7. **No foreign key enforcement on task purge.** Tasks reference blobs, but tasks may be purged; blob rows must not depend on task rows surviving.

---

## Tier 1 — Tasks (Ephemeral)

Tasks are atomic, observable development actions. They are deterministic facts about what happened.

```sql
CREATE TABLE IF NOT EXISTS tasks (
    id            TEXT    NOT NULL PRIMARY KEY,
    kind          TEXT    NOT NULL,
    -- FileWrite | FileDelete | FileRename | Command | CommitCreated
    -- BranchSwitch | MergeEvent | SearchQuery | AgentAction | Note | AgentRecap
    path          TEXT,
    detail        TEXT,
    source        TEXT    NOT NULL DEFAULT 'human',
    -- "human" | "agent:claude" | "agent:cursor" | "agent:openhands" | "ci" | "hook"
    trust_level   INTEGER NOT NULL DEFAULT 1,
    -- 1=GroundTruth | 2=AgentTruth | 3=HumanAssertion | 4=LoreInference
    ts            INTEGER NOT NULL,
    -- unix nanoseconds
    extracted     INTEGER NOT NULL DEFAULT 0,
    -- 0 = pending extraction, 1 = absorbed into a blob
    extracted_into TEXT
    -- blobs.id this task was absorbed into (NULL until extracted)
);

CREATE INDEX IF NOT EXISTS idx_tasks_ts           ON tasks(ts);
CREATE INDEX IF NOT EXISTS idx_tasks_extracted    ON tasks(extracted);
CREATE INDEX IF NOT EXISTS idx_tasks_path         ON tasks(path);
CREATE INDEX IF NOT EXISTS idx_tasks_kind         ON tasks(kind);
CREATE INDEX IF NOT EXISTS idx_tasks_source       ON tasks(source);
```

### AgentRecap Task

When an AI agent emits a structured end-of-session recap, it is stored as a `kind=AgentRecap` task with `trust_level=2`. The `detail` column stores a JSON payload:

```json
{
  "user_intent": "Add Google OAuth support",
  "summary": "Implemented OAuth2 provider flow and callback handling.",
  "recap": "Authentication subsystem migrated toward provider-based login.",
  "kind": "Feature",
  "tags": ["auth", "oauth", "session"]
}
```

This task has `trust_level = 2` (AgentTruth) and is ingested preferentially over Lore's own inference during blob extraction.

### Retention

Tasks are purged when:
- `extracted = 1` AND `ts < now() - retention_nanoseconds` (default: 30 days)
- OR `ts < now() - max_retention_nanoseconds` regardless of extraction (default: 90 days)

Purge runs after each extraction and on `lore init` startup.

---

## Tier 2 — Blobs (Permanent)

Blobs are the primary artifact of Lore. Each represents one meaningful unit of engineering work.

Blobs have two explicitly separated sections: deterministic observed fields and AI-sourced interpreted fields.

```sql
CREATE TABLE IF NOT EXISTS blobs (
    id              TEXT    NOT NULL PRIMARY KEY,

    -- ── Interpreted fields (AI-generated or agent-provided) ────────────────
    kind            TEXT    NOT NULL DEFAULT 'Feature',
    -- Feature | BugFix | Migration | Investigation | Refactor | Architecture | Review | Incident
    title           TEXT    NOT NULL,
    summary         TEXT,
    -- what was done (2-5 sentences)
    recap           TEXT,
    -- bigger picture significance
    user_intent     TEXT,
    -- what the developer was trying to accomplish
    inferred_reasoning TEXT,
    -- Lore's interpretation when no agent recap was available
    tags            TEXT,
    -- JSON array of domain concept strings

    -- ── Provenance fields (deterministic) ──────────────────────────────────
    trust_level     INTEGER NOT NULL DEFAULT 4,
    -- 1=GroundTruth | 2=AgentTruth | 3=HumanAssertion | 4=LoreInference
    -- reflects the trust of the interpreted fields above
    ai_source       TEXT    NOT NULL DEFAULT 'lore:heuristic',
    -- "agent:claude" | "agent:cursor" | "agent:openhands"
    -- | "lore:ollama" | "lore:heuristic"

    -- ── Observed fields (deterministic — set by Lore from tasks) ───────────
    started_at      INTEGER NOT NULL,
    -- unix nanoseconds of first task in extraction window
    ended_at        INTEGER NOT NULL,
    -- unix nanoseconds of last task in extraction window
    commit_start    TEXT,
    -- earliest commit SHA — canonical Blob boundary start
    commit_end      TEXT,
    -- commit SHA for this Blob's commit — canonical Blob boundary end

    -- ── Node assignment ─────────────────────────────────────────────────────
    primary_node_id TEXT,
    -- nodes.id of the primary subsystem this Blob belongs to (NULL = unassigned)
    -- set by: user assertion (trust=3) > agent recap (trust=2) > Lore inference (trust=4)

    created_at      INTEGER NOT NULL
    -- unix nanoseconds when this row was inserted
);

CREATE INDEX IF NOT EXISTS idx_blobs_kind        ON blobs(kind);
CREATE INDEX IF NOT EXISTS idx_blobs_started_at  ON blobs(started_at);
CREATE INDEX IF NOT EXISTS idx_blobs_ended_at    ON blobs(ended_at);
CREATE INDEX IF NOT EXISTS idx_blobs_trust       ON blobs(trust_level);
CREATE INDEX IF NOT EXISTS idx_blobs_ai_source   ON blobs(ai_source);
CREATE INDEX IF NOT EXISTS idx_blobs_node        ON blobs(primary_node_id);

-- FTS for lore explain / lore history (post-MVP)
-- CREATE VIRTUAL TABLE IF NOT EXISTS blobs_fts
-- USING fts5(title, summary, recap, user_intent, tags, content=blobs, content_rowid=rowid);
```

### Tasks Absorbed by a Blob

```sql
CREATE TABLE IF NOT EXISTS blob_tasks (
    blob_id  TEXT    NOT NULL,
    task_id  TEXT    NOT NULL,
    PRIMARY KEY (blob_id, task_id)
    -- task rows may be purged; this join table is also purged when tasks are
);

CREATE INDEX IF NOT EXISTS idx_blob_tasks_blob ON blob_tasks(blob_id);
CREATE INDEX IF NOT EXISTS idx_blob_tasks_task ON blob_tasks(task_id);
```

Note: `blob_tasks` rows are purged when the referenced task is purged. This is intentional — the task list is temporary scaffolding. The blob's permanent record is the interpreted fields and the `blob_files` / `blob_commands` tables.

### Files Associated with a Blob

```sql
CREATE TABLE IF NOT EXISTS blob_files (
    blob_id TEXT    NOT NULL,
    path    TEXT    NOT NULL,
    role    TEXT    NOT NULL,
    -- "written" | "deleted" | "renamed_from" | "renamed_to"
    PRIMARY KEY (blob_id, path, role)
);

CREATE INDEX IF NOT EXISTS idx_bfiles_path   ON blob_files(path);
CREATE INDEX IF NOT EXISTS idx_bfiles_blob   ON blob_files(blob_id);
```

### Commands Associated with a Blob

```sql
CREATE TABLE IF NOT EXISTS blob_commands (
    blob_id TEXT    NOT NULL,
    command TEXT    NOT NULL,
    ts      INTEGER NOT NULL,
    PRIMARY KEY (blob_id, ts, command)
);

CREATE INDEX IF NOT EXISTS idx_bcmds_blob ON blob_commands(blob_id);
```

---

## Tier 3 — Nodes (Permanent)

Nodes are long-lived repository subsystems (Authentication, Billing, Payments, Session Management, etc.). In MVP, Nodes are user-created. Primary Blob → Node assignment is stored directly on the `blobs.primary_node_id` column.

```sql
CREATE TABLE IF NOT EXISTS nodes (
    id          TEXT    NOT NULL PRIMARY KEY,
    title       TEXT    NOT NULL UNIQUE,
    -- subsystem name (user-provided in MVP)
    description TEXT,
    -- what this subsystem is (optional)
    status      TEXT    NOT NULL DEFAULT 'active',
    -- active | archived
    created_by  TEXT    NOT NULL DEFAULT 'user',
    -- "user" | "agent_recap" (post-MVP) | "lore_inference" (post-MVP)
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status);
CREATE INDEX IF NOT EXISTS idx_nodes_title  ON nodes(title);
```

Note: No `confidence` column on Nodes. User-created Nodes are authoritative. Confidence is a post-MVP concept for auto-generated Nodes.

### Secondary Blob → Node Relationships (Post-MVP)

Primary Node assignment lives on `blobs.primary_node_id`. Secondary relationships (a Blob touches multiple subsystems) are deferred post-MVP and will use `node_blobs` when implemented.

```sql
-- Post-MVP: secondary blob-to-node relationships
-- CREATE TABLE IF NOT EXISTS node_blobs (
--     node_id    TEXT NOT NULL,
--     blob_id    TEXT NOT NULL,
--     is_primary INTEGER NOT NULL DEFAULT 0,
--     PRIMARY KEY (node_id, blob_id)
-- );
```

---

## Tier 4 — Knowledge Graph (Derived, Permanent)

The graph is derived from Blobs and Nodes. It can be fully rebuilt from those tables.

```sql
CREATE TABLE IF NOT EXISTS graph_nodes (
    id    TEXT NOT NULL PRIMARY KEY,
    kind  TEXT NOT NULL,
    -- Topic | Blob | File | Commit | Concept
    label TEXT NOT NULL,
    ref   TEXT
    -- nodes.id | blobs.id | file path | commit SHA | concept string
);

CREATE INDEX IF NOT EXISTS idx_gnodes_kind ON graph_nodes(kind);
CREATE INDEX IF NOT EXISTS idx_gnodes_ref  ON graph_nodes(ref);

CREATE TABLE IF NOT EXISTS graph_edges (
    id       TEXT    NOT NULL PRIMARY KEY,
    from_id  TEXT    NOT NULL,
    to_id    TEXT    NOT NULL,
    relation TEXT    NOT NULL,
    -- Contains | Modified | Deleted | Produced | RelatedTo
    -- CausedBy | Involves | PartOf
    weight   INTEGER NOT NULL DEFAULT 1
    -- incremented on duplicate edge (same from_id, to_id, relation)
);

CREATE INDEX IF NOT EXISTS idx_gedges_from     ON graph_edges(from_id);
CREATE INDEX IF NOT EXISTS idx_gedges_to       ON graph_edges(to_id);
CREATE INDEX IF NOT EXISTS idx_gedges_relation ON graph_edges(relation);
CREATE UNIQUE INDEX IF NOT EXISTS idx_gedges_unique
    ON graph_edges(from_id, to_id, relation);
```

### Graph Edge Reference

| Relation | From → To | Meaning |
|----------|-----------|---------|
| `Contains` | Topic → Blob | Node groups this Blob |
| `Modified` | Blob → File | File was written during this work |
| `Deleted` | Blob → File | File was deleted during this work |
| `Produced` | Blob → Commit | Work produced this commit |
| `RelatedTo` | Topic ↔ Topic | Topics that share Blobs or concepts |
| `CausedBy` | Blob → Blob | Incident/fix chain between Blobs |
| `Involves` | Topic → Concept | Topic associated with this domain concept |
| `PartOf` | Topic → Topic | Hierarchical nesting of topics |

---

## Repository Metadata

```sql
CREATE TABLE IF NOT EXISTS meta (
    key   TEXT NOT NULL PRIMARY KEY,
    value TEXT NOT NULL
);

-- Rows inserted on lore init:
-- ('schema_version', '2')
-- ('initialized_at', '<unix_ns>')
-- ('git_root', '/absolute/path/to/repo')
```

---

## Schema Migrations

Numbered SQL files embedded in the binary via `//go:embed`.

```
internal/store/migrations/
├── 001_initial.sql         ← tasks, blobs, blob_files, blob_commands, meta
├── 002_blob_tasks.sql      ← blob_tasks join table
├── 003_nodes.sql           ← nodes, node_blobs
├── 004_graph.sql           ← graph_nodes, graph_edges
└── ...
```

On startup: read `meta.schema_version`, run pending migrations in a single transaction.

---

## Column Ownership Summary

| Table | Columns | Set By | Trust |
|-------|---------|--------|-------|
| `blobs` | `started_at`, `ended_at`, `commit_start`, `commit_end`, `created_at` | Lore (deterministic) | 1 |
| `blobs` | `title`, `summary`, `recap`, `user_intent`, `kind`, `tags` | Agent recap (preferred) or Lore AI fallback | 2 or 4 |
| `blobs` | `inferred_reasoning` | Lore AI only (never agent-provided) | 4 |
| `blobs` | `primary_node_id` | User assertion (preferred) > agent recap > Lore inference | 3, 2, or 4 |
| `blobs` | `trust_level`, `ai_source` | Lore (records provenance) | — |
| `blob_files` | all columns | Lore (from task records) | 1 |
| `blob_commands` | all columns | Lore (from task records) | 1 |
| `blob_tasks` | all columns | Lore (transient — purged with tasks) | 1 |
| `nodes` | `title`, `description` | User (MVP) | 3 |
| `nodes` | `created_by`, `status` | Lore | — |
| `graph_nodes` | all columns | Lore (derived) | — |
| `graph_edges` | all columns | Lore (derived) | — |
| `tasks` | all columns | Lore (from hooks and direct emission) | 1 |
