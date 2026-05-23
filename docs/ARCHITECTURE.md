# Lore — Architecture Reference

> This is the authoritative architectural document for the Lore project.
> All other documents derive from or reference this one.
> If any document contradicts this file, this file wins.

---

## Related Documents

| Document | Purpose |
|----------|---------|
| [../Readme.md](../Readme.md) | Product vision and user-facing overview |
| [ROADMAP.md](./ROADMAP.md) | Phase-by-phase build plan |
| [SCHEMA.md](./SCHEMA.md) | SQLite schema specification |
| [rules/STORAGE.md](./rules/STORAGE.md) | Storage tier rules and retention policy |
| [rules/AI.md](./rules/AI.md) | AI model responsibilities and trust model |
| [rules/CLI.md](./rules/CLI.md) | CLI design rules and permitted commands |
| [rules/GRAPH.md](./rules/GRAPH.md) | Knowledge graph philosophy and constraints |

---

## Product Thesis

Software systems accumulate knowledge faster than they accumulate code.

Every day, developers, AI agents, reviewers, and platform teams make thousands of decisions that shape a codebase. That knowledge dissolves immediately — scattered across commits, PRs, Slack threads, Jira tickets, and human memory.

**Git stores what changed.**

**Lore stores why it changed.**

Lore is a repository-local engineering memory system. It observes development activity as discrete Tasks, compresses those Tasks into durable Blobs of engineering knowledge, groups related Blobs into long-lived Nodes, and builds a queryable Knowledge Graph from those relationships.

---

## What Lore Is

- A repository-local engineering memory system
- A knowledge compression and provenance tool for development activity
- A layer that answers "why does this code exist?"
- A Git companion that enriches code history with engineering context
- A local-first CLI tool modeled after Git's UX conventions

---

## What Lore Is Not

| Not This | Because |
|----------|---------|
| A Git replacement | Git remains the source of truth for code, commits, and branches |
| An AI memory tool | Lore stores engineering knowledge, not agent conversations or prompts |
| A session recorder | Lore does not require manual start/stop — it observes automatically |
| A telemetry platform | Tasks are temporary; Blobs are the artifact |
| A graph visualization product | The graph is a navigation layer, not the value proposition |
| A chat history store | Lore is not aware of conversation content |
| A chain-of-thought store | Lore observes actions, not reasoning |
| A database server | Lore is a local CLI tool that reads and writes `.lore/` |

---

## Core Architecture

Lore operates in four stages. This pipeline is the most important concept in the entire system.

```
Development Activity
        │
        ▼
┌─────────────────────┐
│  Tier 1: Tasks      │  ← temporary, deterministic facts
│  (ephemeral)        │
└────────┬────────────┘
         │  blob extraction
         ▼
┌─────────────────────┐
│  Tier 2: Blobs      │  ← durable units of engineering work
│  (permanent)        │     (observed + inferred, clearly separated)
└────────┬────────────┘
         │  node resolution
         ▼
┌─────────────────────┐
│  Tier 3: Nodes      │  ← long-lived engineering topics
│  (permanent)        │     (collections of related Blobs)
└────────┬────────────┘
         │  relationship derivation
         ▼
┌─────────────────────┐
│  Tier 4: Knowledge  │  ← navigational index
│  Graph (permanent)  │
└─────────────────────┘
```

Tasks are facts. Blobs are memory. Nodes are topics. The graph is navigation.

See [rules/STORAGE.md](./rules/STORAGE.md) for tier retention rules.

---

## Git-Like Behavior

Lore must feel like Git.

### Initialization

```bash
git init    →    creates .git/
lore init   →    creates .lore/
```

After `lore init`, Lore automatically begins observing the repository. No daemon command, no explicit start, no manual session lifecycle.

### Repository Layout

```
repo/
├── .git/        ← managed by Git
├── .lore/       ← managed by Lore
│   ├── lore.db
│   ├── config.toml
│   └── cache/
└── src/
```

### Observation Model

Git does not require `git watch` to start tracking. It simply works.
Lore must behave the same way.

`lore init` installs Git hooks. Those hooks are the observation mechanism. No persistent daemon is required for MVP.

### The Rule

> Lore behaves like `git init`, not like screen recording software.

There is no `lore start`. There is no `lore stop`. There is no `lore session end`.

---

## Storage Model

Lore's storage is local-only and repository-scoped.

```
.lore/
├── lore.db         ← SQLite database (all four tiers)
├── config.toml     ← repository-local configuration
└── cache/          ← transient extraction cache
```

The `lore` binary reads and writes `.lore/` exactly the way `git` reads and writes `.git/`.

Lore is not a database server. Lore is not a service. Lore is a CLI tool.

See [SCHEMA.md](./SCHEMA.md) for the complete SQLite schema.

---

## Task Model (Tier 1)

Tasks are the atomic, observable development actions captured from the repository. They are **facts** — deterministic records of what happened with no AI interpretation.

### Task Types (MVP)

| Kind | Description |
|------|-------------|
| `FileWrite` | A file was created or modified |
| `FileDelete` | A file was deleted |
| `FileRename` | A file was renamed or moved |
| `Command` | A terminal command was executed |
| `CommitCreated` | A Git commit was made |
| `BranchSwitch` | A checkout to a different branch occurred |
| `MergeEvent` | A merge was performed |
| `SearchQuery` | A search was performed within the repository |
| `AgentAction` | An AI agent performed a tool action |
| `Note` | An explicit developer annotation |
| `AgentRecap` | A structured recap emitted by an AI agent at the end of work |

`FileRead` is intentionally excluded from MVP. File reads generate enormous noise and rarely contribute meaningful signal until a solid extraction pipeline exists.

### Task Properties

```
id            string     — unique identifier
kind          TaskKind   — type from the list above
path          string     — affected file path (if applicable)
detail        string     — command text, commit hash, recap JSON, note text, etc.
source        string     — "human" | "agent:claude" | "agent:cursor" | "agent:openhands" | "ci" | "hook"
trust_level   int        — 1=GroundTruth, 2=AgentTruth, 3=HumanAssertion, 4=LoreInference
ts            int64      — unix nanoseconds
extracted     bool       — whether this task has been absorbed into a Blob
extracted_into string    — blob.id (null until extracted)
```

### Retention

Tasks are **temporary telemetry**. Default retention: 30 days regardless of extraction status.

`lore tasks` is not a user-facing command. Tasks are internal plumbing.

---

## Blob Model (Tier 2)

Blobs are the **primary artifact of Lore**. A Blob represents one meaningful unit of engineering work, generated from a group of Tasks. Blobs are what users see in `lore log`, `lore why`, and `lore show`.

Blobs explicitly separate **observed actions** from **inferred reasoning**. This distinction is non-negotiable.

### Blob Structure

```
Blob: OAuth Provider Implementation

Observed Actions (deterministic — set by Lore from Tasks):
  - FileWrite  internal/auth/oauth.go       source=agent:claude
  - FileWrite  internal/session/manager.go  source=agent:claude
  - FileDelete internal/auth/token_legacy.go
  - Command    "go test ./..."
  - CommitCreated  abc123  "feat: add oauth provider"

Inferred Reasoning (AI-generated or agent-provided):
  User Intent:    Add Google OAuth support to replace legacy token auth
  Summary:        Implemented OAuth2 provider flow and callback handling.
                  Replaced custom token system with provider-based login.
  Recap:          Authentication subsystem migrated toward provider-based login.
                  This change eliminates maintenance burden on the legacy token system.

Kind:       Feature
Tags:       auth, oauth, session, provider
Commits:    abc123..def456
Trust:      AgentTruth (recap provided by agent:claude)
```

### Blob Boundary

**A commit is the canonical Blob boundary.**

All Tasks collected between two commits are compressed into a single Blob when the second commit fires. Lore does not perform semantic splitting of commits in MVP. The repository author is responsible for commit hygiene — small, focused commits produce better Blobs.

One commit → one Blob extraction run. One `post-commit` hook → one Blob created.

### Blob Properties

```
id                string    — unique identifier
kind              BlobKind  — Feature | BugFix | Migration | Investigation | Refactor | Architecture | Review | Incident
title             string    — human-readable title (agent-provided or AI-generated)
summary           string    — what was done (agent-provided or AI-generated, 2-5 sentences)
recap             string    — bigger picture significance (agent-provided or AI-generated)
user_intent       string    — what was the developer trying to accomplish
inferred_reasoning string   — Lore's interpretation of the work (when no recap exists)
started_at        int64     — unix nanoseconds of first task in window
ended_at          int64     — unix nanoseconds of last task in window
commit_start      string    — earliest commit SHA (canonical boundary)
commit_end        string    — commit SHA for this Blob's commit
primary_node_id   string    — the Node this Blob belongs to (null if unassigned)
tags              []string  — domain concepts (agent-assisted or AI-generated)
trust_level       int       — 1=GroundTruth | 2=AgentTruth | 3=HumanAssertion | 4=LoreInference
ai_source         string    — "agent:claude" | "agent:cursor" | "lore:ollama" | "lore:heuristic"
created_at        int64     — when this row was inserted
```

**Critical rule:** Observed fields (files, commands, commits, timestamps) are **always** set deterministically by Lore from Task records. Interpreted fields (`summary`, `recap`, `user_intent`, `inferred_reasoning`, `title`, `kind`, `tags`) are set by an AI source or human assertion. These two categories must never be mixed. See [rules/AI.md](./rules/AI.md).

### Retention

Blobs are **permanent**. Never automatically deleted.

---

## Node Model (Tier 3)

Nodes represent **long-lived repository subsystems**. They are stable, named areas of the codebase that persist as the repository evolves.

**Node = subsystem. Blob = work performed on that subsystem.**

### What Nodes Are

Nodes represent enduring parts of the system:

```
Authentication
Billing
Payments
Session Management
API Gateway
User Management
Notification Service
```

### What Nodes Are Not

Nodes are not individual tasks, investigations, or incidents. Those are Blobs.

| This is a Blob | Not a Node |
|----------------|-----------|
| OAuth provider implementation | ✗ |
| JWT expiry bug fix | ✗ |
| Database migration for billing | ✗ |
| Performance investigation | ✗ |
| Billing subsystem | Node ✓ |
| Authentication subsystem | Node ✓ |

### Node Creation (MVP)

In MVP, Nodes are **user-created**. Automatic Node generation is explicitly deferred to post-MVP.

```bash
lore node create Authentication
lore node create Billing
lore node create "Session Management"
```

Users decide which subsystems to track. Lore assigns Blobs to existing Nodes automatically or via explicit assignment.

Future versions may use agent context or specialized local models to suggest Nodes.

### Node Stability

Nodes are **stable and long-lived**. Once created, a Node rarely changes.

- Subsystem evolution is represented by new Blobs assigned to the existing Node, not by creating new Nodes.
- If "Authentication" adds OAuth, that is new Blobs in the Authentication Node, not a new "OAuth Authentication" Node.
- Node identity persists across months and years of development.

### Blob Assignment

A Blob may belong to **one primary Node** in MVP. Secondary relationships are deferred.

Primary Node assignment happens:
1. **Automatically**: if the Blob's tags or file paths match an existing Node's known files (heuristic, `trust_level=4`)
2. **Via explicit user command**: `lore assign <blob-id> <node>` (`trust_level=3`, HumanAssertion — preferred)
3. **Via agent recap**: if the agent recap includes node context (`trust_level=2`)

Unassigned Blobs are valid. They appear in `lore graph` as "Unassigned."

### Node Properties

```
id            string    — unique identifier
title         string    — subsystem name (user-provided)
description   string    — what this subsystem is
status        string    — active | archived
created_by    string    — "user" | "agent_recap" (post-MVP) | "lore_inference" (post-MVP)
created_at    int64
updated_at    int64
```

### Retention

Nodes are **permanent** once created.

---

## Trust Model

Lore has four trust levels. Every piece of interpreted data must be tagged with its trust level.

### Level 1 — Ground Truth

Source: Git commits, file writes, commands, branches, merges. Observed directly.

- Cannot be inferred.
- Cannot be wrong unless the observation mechanism failed.
- Stored with `trust_level = 1`.

Examples: commit SHAs, file paths, timestamps, command strings, branch names.

### Level 2 — Agent Truth

Source: Structured recap emitted by the AI agent that performed the work.

- Claude's session summary via the `Stop` hook.
- Cursor's task description.
- OpenHands' plan and outcome.

Trusted because they come from the actor that did the work. The agent has the most context.

- Stored with `trust_level = 2`.
- Preferred over all non-observed sources.
- Stored in the Blob's `summary`, `recap`, `user_intent`, `kind`, `tags` fields.

### Level 3 — Human Assertion

Source: Explicit user commands that assert facts about Blobs and Nodes.

- `lore node create Authentication` — user defines a subsystem
- `lore assign <blob-id> Authentication` — user assigns a Blob to a Node

Human assertions are more trusted than Lore inference because the human has direct knowledge of the codebase context.

- Stored with `trust_level = 3`.
- Preferred over Lore inference.
- Takes precedence over any previous Lore inference for the same field.

### Level 4 — Lore Inference

Source: Lore's local AI model or heuristics, applied when no higher-trust source exists.

- Generated only when Tasks are present but Agent Truth and Human Assertion are absent.
- Always stored separately from deterministic facts.
- Stored with `trust_level = 4`.

**The rule:** Never treat Level 4 inference as Level 1 ground truth. If Lore inferred something, it must remain distinguishable from what was actually observed or asserted.

---

## Observed vs Inferred — The Critical Distinction

Every Blob must clearly separate what was **observed** from what was **inferred**.

**Observed** = what Lore saw happen (Tasks → deterministic Blob fields):
- Which files were written
- Which commands were run
- Which commits were made
- Time range of work

**Inferred** = interpretation of why it happened (AI-generated Blob fields):
- User's intent
- Significance of the work
- Topic classification
- Node assignment

These two categories must never be mixed in storage, output, or documentation.

---

## AI Model Responsibilities

The AI has two distinct roles. See [rules/AI.md](./rules/AI.md) for the complete specification.

### Role A — Agent-Provided Recaps (Trust Level 2)

When an AI agent (Claude, Cursor, OpenHands) emits a structured recap at the end of its work, Lore ingests it as trusted metadata.

This is the **preferred** path. Agent recaps have the most context about the work because they come from the actor that performed it.

### Role B — Lore Fallback Inference (Trust Level 3)

When no agent recap exists (human coding session, legacy hooks, incomplete data), Lore's local AI model analyzes the observed Tasks and generates:
- Blob title and summary
- User intent inference
- Kind classification
- Tag suggestion
- Node assignment candidates

This is the **fallback** path. Lore inference is lower trust than agent recaps.

**In both roles, the AI never touches:** file lists, commit SHAs, timestamps, command strings, source attribution. Those are always deterministic.

---

## Knowledge Graph Model (Tier 4)

The knowledge graph is a navigational index derived from Blobs and Nodes. It answers relationship questions by following edges.

### Node Types

| Kind | Description | Example |
|------|-------------|---------|
| `Topic` | A long-lived Node | "OAuth Migration" |
| `Blob` | A unit of work | "OAuth Provider Implementation" |
| `File` | A source file path | `internal/auth/oauth.go` |
| `Commit` | A Git commit | `abc123` |
| `Concept` | A domain concept (from tags) | `authentication` |

### Edge Types

| Relation | From → To | Meaning |
|----------|-----------|---------|
| `Contains` | Topic → Blob | This Node groups this Blob |
| `Modified` | Blob → File | File was written during this work |
| `Deleted` | Blob → File | File was deleted during this work |
| `Produced` | Blob → Commit | This work produced this commit |
| `RelatedTo` | Topic ↔ Topic | Topics that share Blobs or concepts |
| `CausedBy` | Blob → Blob | This work was caused by prior work (e.g. incident → fix) |
| `Involves` | Topic → Concept | Topic is associated with this domain concept |
| `PartOf` | Topic → Topic | Hierarchical nesting of topics |

### The Rule

The graph is a consequence of having good Blobs and Nodes. It is not the product.

Do not over-invest in graph visualization at the expense of knowledge quality.

See [rules/GRAPH.md](./rules/GRAPH.md).

---

## Relationship to Git

```
Git Repository
      │
      ▼ (hooks produce Tasks)
     Lore
      │  Tasks → Blobs → Nodes → Graph
      ▼
Knowledge Graph
```

Lore:
- **Consumes** Git metadata (commits, branches, diffs via hooks)
- **Does not modify** Git internals
- **Does not modify** commit messages
- **Does not replace** any Git functionality

Git remains the source of truth for source code, commits, branches, and merges.

---

## Relationship to Atlas

**Ownership boundary:**
- Lore owns: Tasks, Blobs, Nodes
- Atlas consumes: Lore's Blobs and Nodes to enrich architectural understanding
- Atlas must not be required for Lore to function

Lore is a **standalone product**. Atlas integration is an enhancement, not a dependency.

| | Atlas | Lore |
|-|-------|------|
| Understands | Architecture, services, APIs, dependencies | Evolution, provenance, decisions, investigations |
| Answers | "What is this?" | "Why does this exist?" |
| Time dimension | Present state | Historical journey |
| Primary input | Source code, runtime topology | Development activity, Git history |
| Primary artifact | Service graph | Blobs and Nodes |
| Data flow | Consumes Lore output | Produces for Atlas |

Atlas queries Lore's Nodes and Blobs to understand *why* a system looks the way it does. Lore remains fully useful without Atlas installed or configured.

---

## Local-First Requirement

Lore must function completely offline. No request to any external service is required to:
- Initialize a repository (`lore init`)
- Record tasks (Git hooks)
- Extract blobs (heuristic fallback)
- Resolve nodes (clustering heuristics)
- Query the knowledge graph (`lore why`, `lore trace`, `lore log`)

Optional: Ollama for AI-quality summaries. This is additive, not required.

---

## MVP Command Set

```bash
lore init               # initialize .lore/, install git hooks
lore status             # show repository lore state
lore log                # list blobs newest-first (like git log)
lore show <id>          # full detail for one blob
lore why <file>         # why does this file exist?
lore trace <file>       # chronological history of a file
lore graph              # ASCII knowledge graph
lore record <note>      # emit a Note task
lore doctor             # check prerequisites
```

See [rules/CLI.md](./rules/CLI.md) for what is and is not in scope.

---

## What the MVP Explicitly Excludes

- `lore watch` — observation is automatic via hooks
- `lore session` command family — no manual session lifecycle
- `lore tasks` command — tasks are internal, not user-facing
- `lore serve` HTTP server — Lore is a CLI tool, not a service
- `lore explain` / `lore history` — post-MVP query enhancements
- Multi-repo federation
- Cloud synchronization
- Modifying commit messages
- Custom graph database
- Replacing Git
