# Lore — Project Instructions for Claude Code

> This file is read automatically by Claude Code agents working on this project.
> It provides the architecture rules, terminology, and constraints every agent must follow.

---

## What This Project Is

Lore is a **local-first engineering memory system** built in Go.

- It observes development activity (via Git hooks and agent hooks)
- It compresses that activity into durable **Blobs** of engineering knowledge
- It groups related Blobs into long-lived **Nodes** (engineering topics)
- It builds a queryable **Knowledge Graph** from those relationships

**Git stores what changed. Lore stores why it changed.**

---

## Read These Before Working

| Document | When to Read |
|----------|--------------|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Before any implementation work — the authoritative reference |
| [docs/SCHEMA.md](docs/SCHEMA.md) | Before touching storage code |
| [docs/rules/STORAGE.md](docs/rules/STORAGE.md) | Before writing any data model code |
| [docs/rules/AI.md](docs/rules/AI.md) | Before touching extraction or AI integration |
| [docs/rules/CLI.md](docs/rules/CLI.md) | Before adding or changing any CLI commands |
| [docs/rules/GRAPH.md](docs/rules/GRAPH.md) | Before touching graph queries or graph construction |
| [docs/ROADMAP.md](docs/ROADMAP.md) | Before starting a new phase |

Also available at `.claude/` (same content, local to this folder):
- `.claude/ARCHITECTURE.md`
- `.claude/SCHEMA.md`
- `.claude/rules/STORAGE.md`
- `.claude/rules/AI.md`
- `.claude/rules/CLI.md`
- `.claude/rules/GRAPH.md`
- `.claude/ROADMAP.md`

---

## Core Terminology

Use these terms consistently in code, comments, and documentation:

| Term | Definition | Old Term (do not use) |
|------|-----------|----------------------|
| **Task** | Atomic observable action (FileWrite, CommitCreated, etc.) | Event |
| **Blob** | One meaningful unit of engineering work | Knowledge Object, KnowledgeNode |
| **Node** | Long-lived engineering topic grouping Blobs | (no prior equivalent) |
| **Knowledge Graph** | Navigational index derived from Blobs and Nodes | (unchanged) |
| `internal/task/` | Task types and emission | `internal/event/` |
| `internal/blob/` | Blob types and extraction | `internal/knowledge/` |
| `internal/node/` | Node resolution | (new) |

Never use the old terms in new code. If you encounter them in existing code, rename them.

---

## Architecture Rules (Non-Negotiable)

### 1. Four-tier pipeline

```
Tasks (Tier 1, ephemeral)
  → Blobs (Tier 2, permanent)
    → Nodes (Tier 3, permanent)
      → Knowledge Graph (Tier 4, permanent)
```

### 2. No manual session lifecycle

There is no `lore start`, `lore stop`, `lore session end`. Observation is automatic after `lore init`. Do not introduce these patterns.

### 3. Observed and inferred are always separate

Deterministic fields (file paths, commit SHAs, timestamps, commands) are never mixed with AI-generated fields (title, summary, recap, user_intent). They live in separate columns. They display in separate sections.

### 4. Trust levels are always recorded

Every Blob has:
- `trust_level`: 1=GroundTruth, 2=AgentTruth, 3=LoreInference
- `ai_source`: "agent:claude", "agent:cursor", "lore:ollama", "lore:heuristic"

### 5. Agent recaps beat Lore inference

If an `AgentRecap` task exists in the extraction window, use it directly. Do not invoke Ollama. Agent trust level (2) is always preferred over Lore inference trust level (3).

### 6. Tasks are internal plumbing

No `lore tasks` CLI command. No user-facing exposure of raw tasks. Users see Blobs, not Tasks.

### 7. No HTTP server in MVP

No `lore serve`. Lore is a CLI tool, not a service. Agent integration is via direct `lore hook` calls.

### 8. No custom graph database

SQLite adjacency tables are sufficient. No Neo4j, no DGraph, no custom engine.

### 9. No Git internals modification

No `prepare-commit-msg` hook. Lore consumes Git; it does not write to Git.

---

## Package Structure

```
lore/
├── cmd/lore/           # main entrypoint (Cobra root command)
├── internal/
│   ├── store/          # SQLite storage layer
│   ├── task/           # task types, TaskKind enum, task emission
│   ├── blob/           # blob types, extraction pipeline, BlobKind enum
│   ├── node/           # node types, node resolution
│   ├── graph/          # graph queries (SQL-based traversal)
│   ├── git/            # git hook installation, commit metadata reading
│   ├── config/         # TOML config loading
│   └── cli/            # cobra command implementations
├── docs/               # architecture documents (canonical)
├── .claude/            # same docs, accessible to Claude Code agents
├── Claude.md           # this file
├── Readme.md           # user-facing README
├── Makefile
└── go.mod
```

---

## SQL Table Names

| Table | Tier | Purpose |
|-------|------|---------|
| `tasks` | 1 | Atomic observed actions (ephemeral) |
| `blobs` | 2 | Units of engineering work (permanent) |
| `blob_tasks` | 2 | Tasks absorbed by a blob (purged with tasks) |
| `blob_files` | 2 | Files associated with a blob |
| `blob_commands` | 2 | Commands run during a blob |
| `nodes` | 3 | Long-lived engineering topics |
| `node_blobs` | 3 | Blob → Node assignments |
| `graph_nodes` | 4 | Graph node index |
| `graph_edges` | 4 | Graph edge index |
| `meta` | — | Repository metadata |

---

## MVP Command Set

```bash
lore init               # initialize .lore/, install git hooks
lore status             # repository lore state
lore log                # list blobs newest-first
lore show <id>          # full blob detail
lore why <file>         # why does this file exist?
lore trace <file>       # chronological blob history for a file
lore graph              # ASCII knowledge graph
lore record <note>      # emit a Note task
lore doctor             # prerequisites check
```

Internal (not user-facing):
```bash
lore hook <kind> [args]
```

---

## What NOT to Build

- `lore watch` daemon
- `lore session` commands
- `lore tasks` / `lore events` commands
- `lore serve` HTTP server
- `prepare-commit-msg` Git hook
- Custom graph database
- Anything that replaces Git
- Anything that requires a network connection to function

---

## Go Dependencies

```
github.com/spf13/cobra       # CLI framework
modernc.org/sqlite           # pure-Go SQLite (no CGO)
github.com/google/uuid       # UUID generation
```

Post-Phase-5:
```
github.com/charmbracelet/bubbletea   # interactive TUI (deferred)
github.com/charmbracelet/lipgloss    # TUI styling (deferred)
```

---

## Testing Conventions

- All `internal/store/` functions must have unit tests before moving to the next phase.
- Use real SQLite (`:memory:` database) in tests — no mocks.
- Test Blob extraction with both agent recap (Trust Level 2) and heuristic fallback (Trust Level 3) paths.

---

## Style Notes

- Follow standard Go conventions (`gofmt`, `golangci-lint`).
- No comments explaining what the code does — only comments explaining non-obvious constraints or invariants.
- Error messages follow Git conventions: `error: <message>\nhint: <suggestion>`.
- All user-visible output: observed fields and interpreted fields must always be visually distinct.
