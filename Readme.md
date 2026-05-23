# Lore

> Git stores what changed.
>
> Lore stores why it changed.

---

## Documentation

| Document | Purpose |
|----------|---------|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Authoritative architecture reference — read this first |
| [docs/ROADMAP.md](docs/ROADMAP.md) | Phase-by-phase build plan |
| [docs/SCHEMA.md](docs/SCHEMA.md) | SQLite schema specification |
| [docs/rules/STORAGE.md](docs/rules/STORAGE.md) | Storage tier rules and retention policy |
| [docs/rules/AI.md](docs/rules/AI.md) | AI model responsibilities and trust model |
| [docs/rules/CLI.md](docs/rules/CLI.md) | CLI design rules and permitted commands |
| [docs/rules/GRAPH.md](docs/rules/GRAPH.md) | Knowledge graph philosophy and constraints |

---

# Vision

Software systems accumulate knowledge faster than they accumulate code.

Every day developers, AI agents, reviewers, and platform teams make thousands of decisions that shape a codebase.

Today that knowledge is fragmented across:

- Git commits
- Pull Requests
- AI sessions
- Code reviews
- Issue trackers
- Chat applications
- Documentation
- Human memory

Over time the reasoning behind a system disappears.

New engineers are forced into archaeology:

```bash
git log
git blame
search slack
search jira
ask teammates
```

to answer simple questions such as:

- Why does this code exist?
- Why was this dependency introduced?
- Why was this architecture chosen?
- What migration created this system?
- What incident caused this refactor?
- Which files were involved in the original implementation?
- What decisions led to this change?

Git can answer:

> What changed?

Lore exists to answer:

> Why?

---

# Problem

Modern software development is no longer performed exclusively by humans.

Repositories are increasingly shaped by:

- AI coding assistants
- autonomous agents
- CI systems
- platform automation
- multiple engineering teams

These systems generate enormous amounts of context that is lost immediately after execution.

Examples:

- files inspected
- commands executed
- tests run
- plans generated
- migrations performed
- reviews completed
- implementation decisions

The final commit survives.

The reasoning does not.

This creates a growing organizational memory problem.

As teams scale:

- onboarding becomes slower
- architecture knowledge becomes tribal
- debugging becomes harder
- migrations become riskier
- AI loses historical context
- engineering decisions become invisible

Lore exists to preserve that knowledge.

---

# What Is Lore?

Lore is a local-first engineering memory system.

Lore observes software development activity as discrete **Tasks**, compresses those Tasks into durable **Blobs** of engineering knowledge, groups related Blobs into long-lived **Nodes**, and builds a queryable **Knowledge Graph** from those relationships.

Unlike Git, Lore does not attempt to store source code history.

Git already solves that problem extremely well.

Instead, Lore captures:

- engineering intent
- implementation context
- repository evolution
- architectural decisions
- agent activity
- development provenance

Lore transforms development activity into structured knowledge.

---

# Core Philosophy

## Git Stores Code

Git stores:

- blobs
- trees
- commits
- branches
- merges

Git answers:

> What changed?

---

## Lore Stores Knowledge

Lore stores:

- tasks (observed actions)
- blobs (units of work)
- nodes (engineering topics)
- relationships between all of the above

Lore answers:

> Why did it change?

---

## Tasks Are Temporary

Raw development tasks are noisy.

Examples:

```txt
write auth.go
write auth.go

run go test

write oauth.go
commit abc123
```

Most of these become irrelevant quickly.

Lore treats raw tasks as temporary telemetry.

---

## Blobs Are Permanent

Tasks are compressed into durable Blobs.

Example:

```txt
OAuth Provider Implementation

User Intent:
Add Google OAuth support to replace legacy token auth

Observed Actions:
- FileWrite  internal/auth/oauth.go
- FileWrite  internal/session/manager.go
- Command    "go test ./..."
- CommitCreated  abc123

Summary:
Implemented OAuth2 provider flow and callback handling.

Recap:
Authentication subsystem migrated toward provider-based login.

Trust: AgentTruth (source: agent:claude)
Commits: abc100..abc123
```

Knowledge survives. Noise does not.

---

## Observed and Inferred Are Kept Separate

Every Blob separates two categories:

**Observed** — what Lore saw happen (deterministic):
- Which files were written
- Which commands were run
- Which commits were made
- Time range

**Inferred** — interpretation of why it happened (AI-sourced):
- User's intent
- Summary of the work
- Bigger picture significance
- Topic classification

These two categories are never mixed in storage or output.

---

# How Lore Works

Lore operates in four stages. See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the full specification.

---

## Tier 1 — Tasks (Temporary)

Lore observes repository activity via Git hooks installed on `lore init`.

Examples:

- file writes
- file deletions
- terminal commands
- commits
- branch switches
- merges
- AI agent tool actions
- agent session recaps

Tasks are temporary telemetry — the raw signal, not the product.

---

## Tier 2 — Blobs (Permanent)

Tasks are automatically compressed into Blobs after each commit.

Blobs are the primary artifact. They contain what was observed (deterministic) and what was interpreted (AI-sourced), always clearly separated.

If an AI agent (Claude, Cursor, OpenHands) emits a session recap, that recap becomes the Blob's interpretation directly — higher trust than Lore's own inference.

---

## Tier 3 — Nodes (Permanent)

Related Blobs are grouped into long-lived Nodes — engineering topics that collect the story of ongoing work.

Example:

```txt
Node: OAuth Migration
├── Blob: Auth system investigation
├── Blob: OAuth provider implementation
├── Blob: OAuth callback flow
└── Blob: OAuth test coverage
```

---

## Tier 4 — Knowledge Graph (Permanent)

Relationships between Nodes, Blobs, files, and commits form a queryable graph.

```txt
Node: OAuth Migration
├── Contains  Blob: OAuth Provider Implementation
│             ├── Modified  internal/auth/oauth.go
│             └── Produced  commit abc123
└── Contains  Blob: OAuth Investigation
              └── Modified  internal/auth/jwt.go
```

The graph is a navigational index over the knowledge, not the product itself.

---

# Trust Model

Lore tracks the confidence and source of all interpreted information.

**Level 1 — Ground Truth:** Git commits, file writes, commands, timestamps. Cannot be wrong.

**Level 2 — Agent Truth:** Recaps provided by the AI agent that did the work. High trust. Preferred over Lore inference.

**Level 3 — Lore Inference:** Lore's local AI analyzing observed tasks when no agent recap exists. Lower trust. Always distinguishable from observed facts.

---

# Relationship To Git

Lore is not a Git replacement.

Lore depends on Git.

```txt
Git Repository
       │
       ▼ (hooks produce Tasks)
      Lore
       │  Tasks → Blobs → Nodes → Graph
       ▼
Knowledge Graph
```

Git stores code history.

Lore stores engineering history.

---

# Relationship To AI

Lore is model-agnostic.

Any agent capable of emitting actions and a session recap can contribute knowledge.

Examples of agents:
- Claude Code
- Cursor
- OpenHands
- Any agent using `lore hook agent-recap`

Lore captures observed actions.

When an agent provides a recap, Lore uses it directly (Trust Level 2).

When no recap exists, Lore infers from actions (Trust Level 3).

---

# Relationship To Atlas

Atlas understands software structure.

Atlas models:
- services
- APIs
- dependencies
- ownership
- runtime relationships
- architectural topology

Lore understands software evolution.

Lore models:
- tasks
- blobs
- nodes
- implementation journeys
- engineering decisions

Together they answer:

```txt
What is this?   (Atlas)

and

Why does it exist?   (Lore)
```

Lore provides the temporal context Atlas cannot derive from source code alone.

---

# Long-Term Goal

The long-term objective is not creating another development tool.

The objective is creating a repository memory layer.

A system capable of preserving:

- engineering intent
- architectural evolution
- implementation knowledge
- organizational context

across:

- humans
- AI agents
- teams
- repositories
- years of development

Software systems evolve.

Their history should remain understandable.

---

# Example Queries

MVP commands:

```bash
lore log
```

List all Blobs — what engineering work has been done on this repository?

---

```bash
lore why auth.go
```

Why does this file exist? Show the Blobs that created and shaped it.

---

```bash
lore trace oauth.go
```

Show the chronological history of this file.

---

```bash
lore graph
```

Show the knowledge graph — Nodes containing Blobs referencing Files.

---

```bash
lore show abc1234
```

Full detail for one Blob, with observed and interpreted sections separated.

---

Post-MVP:

```bash
lore nodes
```

List all long-lived engineering topic Nodes.

---

```bash
lore explain authentication
```

Show Blobs, Nodes, and decisions related to authentication.

---

# Design Principles

## Local First

Lore should work entirely offline.

Knowledge belongs to the repository.

---

## Git Compatible

Git remains the canonical source of code history.

Lore augments Git.

Never replaces it.

---

## Incremental

Lore continuously updates.

No expensive recomputation.

---

## Observed and Inferred Are Separate

Observed facts and inferred interpretations are always stored and displayed separately.

Trust level is always recorded.

---

## Blobs Over Tasks

Store durable Blobs.

Discard task noise.

---

## Agent Recaps First

When an agent provides a recap, prefer it over Lore's own inference.

The actor has more context than the observer.

---

## Human Readable

Every knowledge artifact should be understandable by engineers.

---

## Open

Lore should be built as an open ecosystem.

Repository knowledge should not be locked to a single vendor.

---

# Internal Mantras

## Git stores code.

## Lore stores context.

## Tasks are temporary.

## Blobs are permanent.

## Observed and inferred never mix.

## Agent recaps beat Lore inference.

## Understanding is more valuable than history.

## Every system has a story.

## Lore preserves that story.
