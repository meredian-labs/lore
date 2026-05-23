# Storage Architecture

Lore stores engineering knowledge in four tiers. Understanding the tiers is essential for understanding how Lore works.

## The Four-Tier Pipeline

```
┌─────────────────────────────────────────────────────────────────┐
│  Tier 1: Tasks                                   (ephemeral)    │
│  Atomic observed actions. Raw signal from Git hooks and agents. │
└───────────────────────┬─────────────────────────────────────────┘
                        │  blob extraction (on commit)
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│  Tier 2: Blobs                                   (permanent)    │
│  Units of engineering work. Observed facts + AI interpretation. │
└───────────────┬───────────────────────────────────┬────────────┘
                │  node resolution                  │
                ▼                                   │
┌─────────────────────────────────┐                 │
│  Tier 3: Nodes       (permanent)│                 │
│  Long-lived subsystems.         │                 │
│  Group related Blobs.           │                 │
└───────────────┬─────────────────┘                 │
                │  graph derivation                 │
                └──────────────────┬────────────────┘
                                   ▼
┌─────────────────────────────────────────────────────────────────┐
│  Tier 4: Knowledge Graph              (derived, permanent)      │
│  Navigational index of relationships.                           │
│  Fully rebuildable from Tiers 2 and 3.                          │
└─────────────────────────────────────────────────────────────────┘
```

---

## Tier 1 — Tasks (Ephemeral)

**What it is:** Atomic, observable development actions. Deterministic facts about what happened.

Tasks are emitted by:
- Git hooks (`post-commit`, `post-checkout`) installed by `lore init`
- Agent integrations (Claude Code Stop hook, Cursor MCP, Windsurf MCP)
- `glh` commands (BranchSwitch, MergeEvent)
- `lore record` (Note tasks)

**Examples:**

```
FileWrite      path=internal/auth/oauth.go    source=agent:claude  trust=1
CommitCreated  detail=abc123                  source=hook          trust=1
Command        detail="go test ./..."         source=human         trust=1
AgentRecap     detail={...json...}            source=agent:claude  trust=2
BranchSwitch   detail=main→feature/oauth      source=glh           trust=1
```

**Retention:** Default 30 days after extraction into a Blob, 90 days unconditionally. Tasks are not the product — they are consumed to create Blobs and then purged.

**User visibility:** None in MVP. Tasks are internal plumbing. There is no `lore tasks` command.

**Rules:**
- Tasks are append-only. A task row is never updated after insert.
- `FileRead` tasks are excluded from MVP (too noisy).
- `AgentRecap` tasks carry `trust_level=2` (AgentTruth) and are the preferred input for blob interpretation.

---

## Tier 2 — Blobs (Permanent)

**What it is:** The primary artifact of Lore. Each Blob is a compressed, human-readable unit of engineering work. It contains both observed facts (deterministic) and interpreted meaning (AI-generated), always clearly separated.

**Examples:**
- OAuth Provider Implementation (kind=Feature, trust=AgentTruth)
- JWT Expiry Fix (kind=BugFix, trust=LoreInferred)
- Billing Service Refactor (kind=Refactor, trust=AgentTruth)

**Retention:** Permanent. Blobs are never automatically deleted.

**User visibility:** Primary. `lore log`, `lore show`, `lore why`, and `lore trace` all operate on Blobs.

**Rules:**
- A Blob must have at least one associated commit or file write to be valid.
- The observed fields (`started_at`, `ended_at`, `commit_start`, `commit_end`, file lists, command lists) are populated deterministically from task data.
- The interpreted fields (`title`, `summary`, `recap`, `user_intent`, `kind`, `tags`) are set by an AI agent recap or by Lore's fallback inference.
- If an `AgentRecap` task exists in the extraction window, Lore uses it directly and skips local AI inference.
- Blobs survive task purges. They have no runtime dependency on tasks.

---

## Tier 3 — Nodes (Permanent)

**What it is:** Long-lived repository subsystems. Stable, named areas of the codebase. Nodes group related Blobs.

**Examples (correct — these are Nodes):**
- Authentication
- Billing
- Session Management
- API Gateway

**Examples (wrong — these are Blobs, not Nodes):**
- ~~OAuth Migration~~ → this is a Blob assigned to the Authentication Node
- ~~JWT Expiry Fix~~ → this is a Blob assigned to the Authentication Node

**Retention:** Permanent once created.

**User visibility:** Primary for navigation. `lore node list`, `lore node show`, and `lore graph` operate on Nodes.

**Rules:**
- In MVP, Nodes are user-created via `lore node create <name>`. Automatic Node generation is post-MVP.
- A Blob has one `primary_node_id`. Secondary relationships are post-MVP.
- Unassigned Blobs (null `primary_node_id`) are valid.
- Node subsystem evolution = new Blobs, not new Nodes.

---

## Tier 4 — Knowledge Graph (Derived, Permanent)

**What it is:** A navigational index of relationships between Blobs, Nodes, files, commits, and concepts. Derived entirely from Tier 2 and Tier 3 data.

**Storage:** Two SQLite tables: `graph_nodes` and `graph_edges`. Adjacency list representation. No graph database is used.

**Retention:** Permanent, but fully rebuildable from Blobs and Nodes.

**User visibility:** Indirect. `lore why`, `lore trace`, and `lore graph` navigate the graph.

**Rules:**
- Graph nodes and edges are derived automatically — never manually authored.
- The graph is a consequence of good Blobs and Nodes. Graph design never drives data model decisions.
- Tasks (Tier 1) are not graph nodes. The graph contains only permanent data.

---

## The Observed vs. Inferred Rule

The most important storage rule in Lore:

**Observed fields** (what Lore saw happen) are always stored separately from **inferred fields** (what AI interpreted).

| Category | Trust Level | Examples | How it's set |
|----------|-------------|----------|--------------|
| Ground Truth | 1 | file paths, commit SHAs, timestamps, commands | Lore reads task records deterministically |
| Agent Truth | 2 | agent recap: title, summary, intent, kind, tags | Agent emits `AgentRecap` task |
| Human Assertion | 3 | node assignments, node names | User runs `lore assign` or `lore node create` |
| Lore Inference | 4 | title, summary, intent, kind, tags (no agent recap) | Lore's local AI or heuristics |

These categories are never mixed in storage. AI-generated content is never written into observed-fact columns. Observed facts are never passed through AI transformation before storage.

**Consequence:** Given the same sequence of observed tasks, two Lore installations on the same repository produce identical Blob observed fields — regardless of AI model availability.

---

## Extraction Pipeline

When a commit occurs, the extraction pipeline runs:

```
PendingTasks (from store)
      │
      ▼
WindowBuilder
      │  fills: started_at, ended_at, files_written,
      │          commands, commit_start, commit_end
      ▼
RecapLookup
      │
      ├── AgentRecap found → RecapIngester
      │    sets: title, summary, recap, user_intent, kind, tags
      │    trust_level=2, ai_source=agent:<name>
      │    (skips LLMClient)
      │
      └── No recap → PromptBuilder → LLMClient (or HeuristicExtractor)
           sets: title, summary, recap, user_intent, kind, tags
           trust_level=4, ai_source=lore:ollama|lore:heuristic
      │
      ▼
Store.InsertBlob()
      │
      ▼
GraphBuilder.UpdateFromBlob()
```

If the AI model (Ollama) is unavailable, the heuristic extractor runs instead. Blobs are always created — the system never blocks on AI availability.

---

## See Also

- [Schema](./schema.md) — all SQLite table definitions
- [Trust Model](./trust-model.md) — trust levels in detail
- [Storage architecture](../architecture/storage.md) — internal storage rules reference
