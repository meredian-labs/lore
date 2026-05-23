---
layout: default
title: Schema
parent: Architecture
nav_order: 2
---
# Database Schema

Lore stores all data in a single SQLite file at `.lore/lore.db`. The schema is divided across the four storage tiers.

---

## Tier 1 — Tasks

### `tasks`

Atomic observed development actions. The raw signal before extraction.

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT PRIMARY KEY | UUID |
| `kind` | TEXT NOT NULL | Task kind: FileWrite, CommitCreated, Command, AgentRecap, BranchSwitch, MergeEvent, Note, RepoInit |
| `source` | TEXT NOT NULL | Who emitted this task: `human`, `agent:claude`, `agent:cursor`, `hook`, `glh` |
| `trust_level` | INTEGER NOT NULL | 1=GroundTruth, 2=AgentTruth |
| `detail` | TEXT | Kind-specific payload. JSON for structured kinds (AgentRecap). Plain string for others. |
| `path` | TEXT | File path, for FileWrite tasks |
| `created_at` | DATETIME NOT NULL | When the task was recorded |
| `extracted_at` | DATETIME | When this task was absorbed into a Blob. NULL = pending. |

**Retention:** Default 30 days after `extracted_at`, 90 days unconditionally.

**Rules:**
- Append-only. Never updated after insert.
- `FileRead` tasks are not recorded in MVP.
- `AgentRecap` tasks carry `trust_level=2`.

---

## Tier 2 — Blobs

### `blobs`

Units of engineering work. The primary artifact. Permanent.

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT PRIMARY KEY | UUID |
| `kind` | TEXT NOT NULL | Feature, BugFix, Refactor, Migration, Investigation, Architecture, Review, Incident |
| `trust_level` | INTEGER NOT NULL | 2=AgentTruth, 4=LoreInference |
| `ai_source` | TEXT NOT NULL | `agent:claude`, `agent:cursor`, `lore:ollama`, `lore:heuristic` |
| `started_at` | DATETIME NOT NULL | Timestamp of first task in extraction window |
| `ended_at` | DATETIME NOT NULL | Timestamp of last task in extraction window |
| `commit_start` | TEXT | First commit SHA in the window |
| `commit_end` | TEXT | Last commit SHA in the window |
| `title` | TEXT | Short name for the work (AI-generated) |
| `summary` | TEXT | 2–5 sentence description (AI-generated) |
| `recap` | TEXT | 1–3 sentence bigger-picture significance (AI-generated) |
| `user_intent` | TEXT | Best guess at what the developer was trying to accomplish (AI-generated) |
| `inferred_reasoning` | TEXT | Lore's internal reasoning when using fallback inference. NULL for AgentTruth blobs. |
| `tags` | TEXT | JSON array of domain concept strings |
| `primary_node_id` | TEXT | FK → `nodes.id`. NULL if unassigned. |
| `created_at` | DATETIME NOT NULL | When this blob was created |

**Retention:** Permanent. Never automatically deleted.

**Key constraints:**
- A blob must have at least one row in `blob_files` to be valid.
- `title`, `summary`, `recap`, `user_intent`, `kind`, `tags` are always AI-generated (agent or Lore). Never set from observed task data directly.
- `started_at`, `ended_at`, `commit_start`, `commit_end` are always observed facts. Never set from AI output.

### `blob_files`

Files associated with a blob.

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT PRIMARY KEY | UUID |
| `blob_id` | TEXT NOT NULL | FK → `blobs.id` |
| `path` | TEXT NOT NULL | Repository-relative file path |
| `role` | TEXT NOT NULL | `written` or `deleted` |

**Retention:** Permanent (follows its blob).

### `blob_commands`

Terminal commands run during a blob's extraction window.

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT PRIMARY KEY | UUID |
| `blob_id` | TEXT NOT NULL | FK → `blobs.id` |
| `command` | TEXT NOT NULL | Command string as executed |
| `ran_at` | DATETIME NOT NULL | When the command was run |

**Retention:** Permanent (follows its blob).

### `blob_tasks`

Join table recording which tasks were absorbed by a blob. Used for provenance before tasks are purged.

| Column | Type | Description |
|--------|------|-------------|
| `blob_id` | TEXT NOT NULL | FK → `blobs.id` |
| `task_id` | TEXT NOT NULL | FK → `tasks.id` |

**Retention:** Rows are removed when the corresponding task is purged. The blob itself is unaffected.

---

## Tier 3 — Nodes

### `nodes`

Long-lived repository subsystems. Permanent.

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT PRIMARY KEY | UUID |
| `name` | TEXT NOT NULL UNIQUE | Subsystem name (e.g., "Authentication", "Billing") |
| `description` | TEXT | Optional description |
| `status` | TEXT NOT NULL | `active` or `archived` |
| `created_at` | DATETIME NOT NULL | When the node was created |
| `created_by` | TEXT NOT NULL | `human` (all nodes are user-created in MVP) |

**Retention:** Permanent once created.

### `node_blobs`

Join table assigning blobs to nodes.

| Column | Type | Description |
|--------|------|-------------|
| `node_id` | TEXT NOT NULL | FK → `nodes.id` |
| `blob_id` | TEXT NOT NULL | FK → `blobs.id` |
| `trust_level` | INTEGER NOT NULL | 3=HumanAssertion (set via `lore assign`) |
| `assigned_at` | DATETIME NOT NULL | When the assignment was made |

**Retention:** Permanent (follows its node and blob).

**Rules:**
- In MVP, all `node_blobs` rows have `trust_level=3` (HumanAssertion) — set by `lore assign`.
- Automatic node assignment (trust_level=4) is post-MVP.

---

## Tier 4 — Knowledge Graph

### `graph_nodes`

Adjacency list nodes for the knowledge graph.

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT PRIMARY KEY | UUID |
| `kind` | TEXT NOT NULL | `Subsystem`, `Blob`, `File`, `Commit`, `Concept` |
| `ref_id` | TEXT | FK to the corresponding Tier 2/3 row (blob id, node id, etc.) |
| `label` | TEXT NOT NULL | Display name |
| `created_at` | DATETIME NOT NULL | When this graph node was created |

### `graph_edges`

Directed relationships between graph nodes.

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT PRIMARY KEY | UUID |
| `from_id` | TEXT NOT NULL | FK → `graph_nodes.id` |
| `to_id` | TEXT NOT NULL | FK → `graph_nodes.id` |
| `relation` | TEXT NOT NULL | `Contains`, `Modified`, `Deleted`, `Produced`, `RelatedTo`, `CausedBy`, `Involves`, `PartOf` |
| `weight` | INTEGER NOT NULL DEFAULT 1 | Incremented on duplicate edges instead of inserting new rows |
| `created_at` | DATETIME NOT NULL | When this edge was first created |

**Retention:** Permanent, but fully rebuildable from Tiers 2 and 3.

---

## Meta

### `meta`

Repository-level metadata.

| Column | Type | Description |
|--------|------|-------------|
| `key` | TEXT PRIMARY KEY | Metadata key |
| `value` | TEXT NOT NULL | Metadata value |

**Common keys:**

| Key | Description |
|-----|-------------|
| `repo_path` | Absolute path to the repository root |
| `initialized_at` | Timestamp of `lore init` |
| `lore_version` | Version of Lore that initialized this database |
| `schema_version` | Current schema migration version |

---

## Table Summary

| Table | Tier | Retention | User-Visible |
|-------|------|-----------|--------------|
| `tasks` | 1 | 30–90 days | No |
| `blobs` | 2 | Permanent | Yes (primary) |
| `blob_files` | 2 | Permanent | Yes (in `lore show`) |
| `blob_commands` | 2 | Permanent | Yes (in `lore show`) |
| `blob_tasks` | 2 | Until task purged | No |
| `nodes` | 3 | Permanent | Yes |
| `node_blobs` | 3 | Permanent | Yes |
| `graph_nodes` | 4 | Permanent (rebuildable) | Indirect |
| `graph_edges` | 4 | Permanent (rebuildable) | Indirect |
| `meta` | — | Permanent | Via `lore status` |

---

## See Also

- [Storage Architecture](./storage) — four-tier pipeline explanation
- [Trust Model](./trust-model) — trust levels and their effects
