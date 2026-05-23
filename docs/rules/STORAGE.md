# Rule: Storage Tiers

> Parent: [../ARCHITECTURE.md](../ARCHITECTURE.md)
> Schema: [../SCHEMA.md](../SCHEMA.md)

---

## The Four Tiers

Lore has exactly four storage tiers. Every piece of data belongs to exactly one tier. Confusing which tier something belongs to is a design error.

The four tiers form a pipeline, not a flat list:

```
Tier 1: Tasks       →  (blob extraction)    →  Tier 2: Blobs
Tier 2: Blobs       →  (node resolution)    →  Tier 3: Nodes
Tier 2+3            →  (graph derivation)   →  Tier 4: Knowledge Graph
```

---

## Tier 1 — Tasks (Ephemeral)

**What it is:** Atomic, observable development actions. Deterministic facts about what happened. The raw signal.

**What it is not:** The primary artifact. Not a log product. Not something users query routinely.

**Retention:** Temporary. Default: 30 days after extraction, 90 days unconditionally.

**User visibility:** None in MVP. Tasks are internal plumbing.

**Examples:**
- `FileWrite   path=internal/auth/oauth.go  source=agent:claude  trust=1`
- `CommitCreated  detail=abc123  source=hook  trust=1`
- `Command  detail="go test ./..."  source=human  trust=1`
- `AgentRecap  detail={...}  source=agent:claude  trust=2`

**Rules:**
- Tasks are append-only. Never update a task row after insert.
- Tasks drive blob extraction but are not the output.
- Do not expose a `lore tasks` CLI command in MVP.
- Do not retain tasks indefinitely. They are not the product.
- `FileRead` tasks are excluded from MVP (too noisy, insufficient signal value).
- `AgentRecap` tasks carry `trust_level=2` (AgentTruth). They are the preferred input for blob interpretation.

---

## Tier 2 — Blobs (Permanent)

**What it is:** The primary artifact of Lore. Compressed, human-readable units of engineering work. Each Blob contains both observed facts (deterministic) and interpreted meaning (AI-sourced), clearly separated.

**What it is not:** A session. Not a chat log. Not a raw task replay. Not a stream of thought.

**Retention:** Permanent. Never automatically deleted.

**User visibility:** Primary. `lore log`, `lore show`, `lore why`, `lore trace` all operate on Blobs.

**Examples:**
- OAuth Provider Implementation (kind=Feature, trust=AgentTruth)
- JWT Expiry Fix (kind=BugFix, trust=LoreInference)
- Billing Service Refactor (kind=Refactor, trust=AgentTruth)
- Auth Flow Investigation (kind=Investigation, trust=LoreInference)

**Rules:**
- Blobs are permanent once created.
- The observed fields (`started_at`, `ended_at`, `commit_start`, `commit_end`, file lists, command lists) are set by Lore deterministically from task data.
- The interpreted fields (`title`, `summary`, `recap`, `user_intent`, `kind`, `tags`) are set by AI (agent-provided or Lore fallback).
- `inferred_reasoning` is set by Lore's local AI only — never by agent recaps. It represents Lore's own interpretation when no agent context exists.
- A Blob must have at least one associated commit or file write to be valid.
- Blobs have no dependency on tasks surviving. Tasks may be purged; Blobs remain.
- `trust_level` on a Blob reflects the source of its interpreted fields:
  - `2` = agent-provided recap (AgentTruth) — preferred
  - `3` = Lore fallback inference (LoreInference) — acceptable

---

## Tier 3 — Nodes (Permanent)

**What it is:** Long-lived repository subsystems. Stable, named areas of the codebase.

**What it is not:** A task, investigation, migration, or bug fix. Those are Blobs.

**Retention:** Permanent once created.

**User visibility:** Primary for navigation. `lore node list` shows all subsystems. `lore graph` shows Node → Blob hierarchy. `lore why` shows which Node a Blob belongs to.

**Examples (correct — these are Nodes):**
- Authentication (active, 12 blobs)
- Billing (active, 8 blobs)
- Session Management (active, 5 blobs)
- API Gateway (archived, 3 blobs)

**Examples (wrong — these are Blobs, not Nodes):**
- ~~OAuth Migration~~ → Blob, assigned to Authentication Node
- ~~JWT Expiry Fix~~ → Blob, assigned to Authentication Node
- ~~Database Migration for Billing~~ → Blob, assigned to Billing Node

**Rules:**
- In MVP, Nodes are **user-created** via `lore node create <name>`.
- Automatic Node generation is deferred to post-MVP.
- Nodes are stable. Subsystem evolution = new Blobs, not new Nodes.
- A Blob has one `primary_node_id` (one primary Node). Secondary relationships are post-MVP.
- Unassigned Blobs (null `primary_node_id`) are valid.
- Node assignment trust levels: user assertion (trust=3) > agent recap (trust=2) > Lore inference (trust=4).

---

## Tier 4 — Knowledge Graph (Derived, Permanent)

**What it is:** A navigational index of relationships between Blobs, Nodes, files, commits, and concepts. Derived entirely from Tier 2 and Tier 3 data.

**What it is not:** The product. Not a visualization platform. Not a separate database.

**Retention:** Permanent, but fully rebuildable from Blobs and Nodes.

**User visibility:** Indirect. Users navigate the graph via `lore why`, `lore trace`, and `lore graph`. They do not query nodes and edges directly.

**Rules:**
- The graph is a consequence of good Blobs and Nodes. Never let graph design drive data model decisions.
- Graph nodes and edges are derived — never manually authored.
- Do not build a custom graph database. SQLite adjacency tables are correct and sufficient.
- See [GRAPH.md](./GRAPH.md) for graph philosophy rules.

---

## Observed vs Inferred — Storage Rule

The most important storage rule in the entire system:

**Observed fields** (what Lore saw happen) must always be stored separately from **inferred fields** (what AI interpreted or users asserted).

| Category | Trust Level | Examples | Storage location |
|----------|-------------|----------|-----------------|
| Observed (Ground Truth) | 1 | file paths, commit SHAs, timestamps, commands | `started_at`, `ended_at`, `commit_start`, `commit_end`, `blob_files`, `blob_commands` |
| Agent-provided (Agent Truth) | 2 | agent recap fields | `title`, `summary`, `recap`, `user_intent`, `kind`, `tags` (when ai_source=agent:*) |
| Human-asserted | 3 | user node assignments | `primary_node_id` (when set via `lore assign`), node `title` |
| Lore-inferred | 4 | Lore AI interpretation | `title`, `summary`, `recap`, `kind`, `tags` (when ai_source=lore:*), `inferred_reasoning` |
| Provenance | — | where interpretation came from | `trust_level`, `ai_source` columns |

Never put AI-generated content into observed-fact columns. Never put observed facts through AI transformation before storage.

---

## What Does Not Belong in Any Tier

| Data | Reason Not Stored |
|------|------------------|
| Agent conversation history | Not engineering knowledge; not provenance |
| LLM prompt text | Intermediate computation; not durable state |
| LLM raw response text | Parsed fields are stored; raw response is not |
| Agent chain-of-thought | Not observable action; not repository-relevant |
| User session tokens / auth | Lore has no auth system |
| Build artifacts, lock files | Not source evolution |

---

## Storage Is Deterministic for Observed Facts

Given the same sequence of observed tasks, two Lore installations on the same repository must produce identical Blob observed fields (modulo AI-generated text).

This means:
- Task capture is rule-based, not probabilistic
- Extraction windows are time-based and signal-based, not AI-driven
- File associations come from task records, not inference
- Commit associations come from Git hooks, not guessing

**The AI compresses and interprets. It does not define what happened.**
