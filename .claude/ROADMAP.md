# Lore — Build Roadmap

> Language: Go
> Architecture: [ARCHITECTURE.md](./ARCHITECTURE.md)
> Schema: [SCHEMA.md](./SCHEMA.md)
> Rules: [rules/](./rules/)
>
> Each phase produces a usable artifact. No half-finished skeletons.
> Read ARCHITECTURE.md before reading this document.

---

## Phase Summary

| Phase | Name | Ends When |
|-------|------|-----------|
| 0 | Foundation | `lore init` creates `.lore/`, binary compiles |
| 1 | Storage Layer | Four-tier SQLite schema, all CRUD tested |
| 2 | Task Capture | Git hooks emit Tasks on commit/branch/merge |
| 3 | Blob Extraction + Agent Recap Ingestion | Blobs created automatically; agent recaps ingested |
| 4 | Node Creation | `lore node create` and `lore assign` work; blobs assigned to subsystems |
| 5 | Query Interface | All MVP commands work end-to-end |
| 6 | Agent Integration | Claude Code hooks populate Lore; recap hook wired |
| 7 | Distribution | `brew install lore` works on a clean Mac |

Post-MVP (outside this roadmap):
- Phase 8: Atlas integration
- Phase 9: Multi-repo federation
- Phase 10: Cloud sync (opt-in)

---

## Tracking Lore with Lore

> The best way to understand what Lore produces is to run it on itself while you build it.
> This section tells you exactly when to initialize, what to do at each phase, and what output to expect.
> By the time you finish Phase 5 you will have a real knowledge graph of your own build history.

### When to Initialize

Run `lore init` inside the Lore repository **at the end of Phase 0** — immediately after the binary first compiles and `lore init` itself works. The git hooks install then, so every commit you make during Phases 1–7 gets captured automatically. Do not wait until Phase 3; you will lose the early build history.

```bash
# End of Phase 0 — do this once
cd /path/to/lore
./lore init

# Verify hooks are installed
cat .git/hooks/post-commit
# → #!/bin/sh
# → # Managed by lore — do not edit manually
# → lore hook commit
```

At this point `lore status` shows all zeros. That is correct.

---

### What You Will See Phase by Phase

#### After Phase 0 (Foundation)

```bash
$ lore status

Repository: /path/to/lore
Initialized: 2026-05-23

Blobs: 0

Subsystems (Nodes): 0

Pending Tasks: 0

LLM: ollama/llama3 (not checked — run 'lore doctor')
```

Nothing to see yet. But the hooks are live. Every commit you make from here on will be captured.

---

#### After Phase 1 (Storage Layer)

You will have made several commits implementing the schema, store package, and tests. The hooks are firing and accumulating Tasks, but extraction has not been built yet. Tasks are piling up in `tasks` table.

```bash
$ lore status

Repository: /path/to/lore
Initialized: 2026-05-23

Blobs: 0

Subsystems (Nodes): 0

Pending Tasks: 8   ← commits from Phase 1 work, waiting for extraction

LLM: ollama/llama3 (not checked — run 'lore doctor')
```

Still no Blobs. That is expected — extraction is Phase 3. The pending task count tells you the hooks are working correctly.

---

#### After Phase 2 (Task Capture)

More commits. Pending task count grows. You can now also annotate your work in progress:

```bash
# While working on the post-checkout hook implementation
lore record "implementing branch switch detection in hook checkout"

# While debugging diff-tree parsing
lore record "git diff-tree name-status parsing edge case with renames"
```

These Note tasks will be folded into the next Blob extraction alongside the commit Tasks.

```bash
$ lore status

Pending Tasks: 14   ← Phase 0 + Phase 1 + Phase 2 commits + your notes
```

---

#### After Phase 3 (Blob Extraction)

**This is where it gets real.** The extraction pipeline is now built. Every new commit triggers extraction. All 14 pending tasks from Phases 0–2 get extracted retroactively on the first commit that lands in Phase 3.

```bash
$ lore log

a1b2c3d4  Implement SQLite schema migr...  Feature     2026-05-23  [LoreInferred]   12 files
e5f6a7b8  Add store CRUD methods and t...  Feature     2026-05-24  [LoreInferred]    8 files
c9d0e1f2  Git hook installation and co...  Feature     2026-05-25  [LoreInferred]    5 files
g3h4i5j6  Task capture: commit, branch...  Feature     2026-05-26  [LoreInferred]    6 files
k7l8m9n0  Blob extraction pipeline and...  Feature     2026-05-27  [LoreInferred]    9 files
```

The first few Blobs will be `[LoreInferred]` because Phase 6 (agent recap) is not wired yet. The heuristic extractor will classify most things as `Feature` since the commit messages start with "Add", "Implement", etc. That is fine — the observed fields (files modified, commits, time range) are accurate regardless of trust level.

Try the detail view:

```bash
$ lore show a1b2c3d4

ID:           a1b2c3d4-...
Title:        Implement SQLite schema migrations
Kind:         Feature
Trust:        LoreInferred (source: lore:heuristic)

── Observed ────────────────────────────────────────
Started:      2026-05-23 09:14
Ended:        2026-05-23 16:42
Commits:      abc100..abc123

Files Modified:
  internal/store/store.go
  internal/store/migrate.go
  internal/store/migrations/001_initial.sql
  internal/store/migrations/002_blob_tasks.sql
  internal/store/migrations/003_nodes.sql
  internal/store/migrations/004_graph.sql
  internal/store/store_test.go

Commands:
  go test ./internal/store/...
  go test ./...

── Interpreted ─────────────────────────────────────
User Intent:  (none)
Summary:      Modified 7 file(s). Ran 2 command(s). Produced 1 commit(s).
Recap:        (none)
Tags:         store, migrations

── Part of ─────────────────────────────────────────
Node: (unassigned)
      hint: use 'lore assign a1b2c3d4 <subsystem>'
```

The observed section is fully accurate. The interpreted section is thin because no agent recap existed — that improves in Phase 6.

---

#### After Phase 4 (Node Creation)

Create nodes that reflect the actual subsystem structure of Lore itself, then assign the blobs you have so far:

```bash
lore node create "Storage Layer"
lore node create "Task Capture"
lore node create "Blob Extraction"
lore node create "CLI"
lore node create "Git Integration"

# Assign existing blobs
lore assign a1b2c3d4 "Storage Layer"   # schema + migrations blob
lore assign e5f6a7b8 "Storage Layer"   # CRUD methods blob
lore assign c9d0e1f2 "Git Integration" # hook installation blob
lore assign g3h4i5j6 "Task Capture"    # hook commit/checkout/merge blob
lore assign k7l8m9n0 "Blob Extraction" # extraction pipeline blob
```

Now the graph is meaningful:

```bash
$ lore graph

Subsystem: Storage Layer
├── Implement SQLite schema migr...  (Feature, 2026-05-23)  [LoreInferred]
│   ├── Modified  internal/store/migrate.go
│   └── Modified  internal/store/migrations/001_initial.sql
└── Add store CRUD methods and t...  (Feature, 2026-05-24)  [LoreInferred]
    ├── Modified  internal/store/store.go
    └── Modified  internal/store/store_test.go

Subsystem: Task Capture
└── Task capture: commit, branch...  (Feature, 2026-05-26)  [LoreInferred]
    ├── Modified  internal/cli/hook.go
    └── Modified  internal/task/task.go

Subsystem: Blob Extraction
└── Blob extraction pipeline and...  (Feature, 2026-05-27)  [LoreInferred]
    ├── Modified  internal/blob/window.go
    └── Modified  internal/blob/heuristic.go

Subsystem: Git Integration
└── Git hook installation and co...  (Feature, 2026-05-25)  [LoreInferred]
    ├── Modified  internal/git/hooks.go
    └── Modified  internal/git/root.go

Subsystem: CLI
  (no blobs assigned yet)

Unassigned Blobs: 0
```

---

#### After Phase 5 (Query Interface)

Now try the full query set against the Lore repo itself:

```bash
# Why does the store package exist?
$ lore why internal/store/store.go

Implement SQLite schema migrations  (Feature)  2026-05-23  [LoreInferred]
  Modified 7 file(s). Ran 2 command(s). Produced 1 commit(s).
  Commits: abc100..abc123
  Node: Storage Layer

Add store CRUD methods and tests  (Feature)  2026-05-24  [LoreInferred]
  Modified 8 file(s). Ran 3 command(s). Produced 1 commit(s).
  Commits: abc124..abc131
  Node: Storage Layer
```

```bash
# Chronological history of the extraction pipeline
$ lore trace internal/blob/window.go

Blob extraction pipeline and heuristic extractor  (Feature)  2026-05-27  [LoreInferred]
  Modified 9 file(s). Ran 4 command(s). Produced 2 commit(s).
  Commits: abc140..abc148
  Node: Blob Extraction
```

```bash
# Full status
$ lore status

Repository: /path/to/lore
Initialized: 2026-05-23

Blobs: 8
  Feature:   8   (0 AgentTruth, 8 LoreInferred)

Subsystems (Nodes): 5
  Storage Layer    (2 blobs, active)
  Task Capture     (1 blob,  active)
  Blob Extraction  (1 blob,  active)
  Git Integration  (1 blob,  active)
  CLI              (0 blobs, active)

Unassigned Blobs: 3
  hint: use 'lore assign <id> <subsystem>' or 'lore node create <name>'

Pending Tasks: 0

LLM: ollama/llama3 (available)
```

---

#### After Phase 6 (Agent Integration)

If you built with Claude Code, re-running the same session with the `Stop` hook now wired means future commits produce `[AgentTruth]` Blobs with real summaries. You can also re-extract existing blobs manually post-MVP with `lore re-extract <id>`.

New blobs from Phase 6 work will look like:

```bash
$ lore show p1q2r3s4

ID:           p1q2r3s4-...
Title:        Wire Claude Code Stop hook for agent recap ingestion
Kind:         Feature
Trust:        AgentTruth (source: agent:claude)

── Observed ────────────────────────────────────────
Started:      2026-05-28 11:03
Ended:        2026-05-28 14:22
Commits:      abc180..abc183

Files Modified:
  internal/cli/hook.go
  internal/cli/init.go
  internal/task/recap.go

Commands:
  go test ./...
  go build ./cmd/lore

── Interpreted ─────────────────────────────────────
User Intent:  Connect the Claude Code Stop hook so agent session recaps are stored
              as AgentRecap tasks and produce Trust Level 2 blobs
Summary:      Implemented lore hook agent-recap reading from CLAUDE_STOP_HOOK_PAYLOAD
              env variable. Added AgentRecapPayload validation and RecapIngester
              integration. Updated lore init to write .claude/settings.json with
              the Stop hook configuration.
Recap:        Agent integration is now complete. Lore can receive structured recaps
              from Claude Code sessions and produce higher-trust blobs that describe
              intent rather than just observed file changes.
Tags:         agent, recap, hooks, claude

── Part of ─────────────────────────────────────────
Node: CLI
```

The difference between a `[LoreInferred]` blob and an `[AgentTruth]` blob is immediately visible. This is the product working correctly on itself.

---

### Suggested Node Structure for the Lore Repository

Use these nodes when you initialize Lore on itself:

```bash
lore node create "Storage Layer"    # internal/store/ — SQLite, migrations, CRUD
lore node create "Task Capture"     # internal/task/, internal/git/ — hooks, task emission
lore node create "Blob Extraction"  # internal/blob/ — window, LLM, heuristic, recap
lore node create "Graph"            # internal/graph/ — builder, queries, adjacency
lore node create "Node Resolution"  # internal/node/ — node types, assignment
lore node create "CLI"              # internal/cli/, cmd/lore/ — all user-facing commands
lore node create "Config"           # internal/config/ — TOML loading, defaults
```

These map directly to the package structure. Each subsystem in the code becomes a Node in Lore.

---

### Why This Matters

By the time you finish Phase 5 you will have answered:

- **Why does `internal/store/migrate.go` exist?** → `lore why internal/store/migrate.go`
- **What changed in the storage layer over time?** → `lore node show "Storage Layer"`
- **What was the intent behind the heuristic extractor?** → `lore show <blob-id>` (AgentTruth after Phase 6)
- **Which files were touched most during the build?** → visible in `lore graph`

This is not a demo scenario. It is real engineering history about a real system, produced automatically as a side effect of building normally.

---

## Phase 0 — Foundation

**Goal:** Go project compiles. `lore init` creates `.lore/` with an empty database.

### Repository Structure

```
lore/
├── cmd/lore/           # main entrypoint
├── internal/
│   ├── store/          # SQLite storage layer
│   ├── task/           # task types and emission
│   ├── blob/           # blob types and extraction pipeline
│   ├── node/           # node resolution
│   ├── graph/          # knowledge graph queries
│   ├── git/            # git hook installation and metadata reading
│   ├── config/         # config loading
│   └── cli/            # command implementations
├── docs/               # architecture documents
├── Makefile
└── go.mod
```

No `internal/session/` package. No `internal/event/` package. The old terminology does not appear in code.

### Deliverables

- `go.mod` initialized
- Cobra CLI skeleton: `lore --help` works
- `lore init` creates `.lore/lore.db` and `.lore/config.toml`
- `lore init` adds `.lore/` to `.gitignore`
- SQLite via `modernc.org/sqlite` (pure Go, no CGO)
- Makefile: `build`, `test`, `lint`, `install`
- `golangci-lint` configured

### Dependencies (Phase 0)

```
github.com/spf13/cobra
modernc.org/sqlite
github.com/google/uuid
```

### Exit Criteria

`lore --help` shows command list. `lore init` creates `.lore/` with a valid SQLite database.

### File Structure (Exact)

Every file to create in Phase 0:

```
lore/
├── cmd/lore/
│   └── main.go                    # package main — calls cli.Execute(), handles exit codes
├── internal/
│   ├── cli/
│   │   ├── root.go                # cobra root command, PersistentPreRunE, findLoreRoot()
│   │   ├── init.go                # runInit(cmd, args) — 9-step init sequence
│   │   ├── status.go              # runStatus(cmd, args) — Phase 0: stub printing zero counts
│   │   └── hook.go                # runHook(cmd, args) — Hidden: internal use only
│   ├── config/
│   │   ├── config.go              # Config struct, Load(), WriteDefault(), Defaults()
│   │   └── config_test.go         # round-trip TOML load tests
│   ├── store/
│   │   └── store.go               # skeleton: Open(path string) (*Store, error) only
│   ├── git/
│   │   ├── root.go                # FindGitRoot() (string, error)
│   │   └── hooks.go               # InstallHooks(gitRoot, loreRoot string) error
│   └── task/
│       └── kind.go                # TaskKind type + 11 const values
├── docs/                          # existing
├── .claude/                       # existing
├── CLAUDE.md                      # existing
├── Readme.md                      # existing
├── Makefile
├── .golangci.yml
└── go.mod
```

### `go.mod` Content

```
module github.com/nishchay/lore

go 1.22

require (
    github.com/google/uuid v1.6.0
    github.com/spf13/cobra v1.8.1
    modernc.org/sqlite v1.30.0
)
```

Module path must match the GitHub repository exactly. No v2 suffix. Add `github.com/BurntSushi/toml` for config loading (add to this list).

### Makefile Targets

```makefile
BINARY   = lore
VERSION ?= dev
LDFLAGS  = -ldflags "-X github.com/nishchay/lore/internal/cli.Version=$(VERSION)"

.PHONY: build test lint install clean release snapshot

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/lore

test:
	go test ./...

lint:
	golangci-lint run ./...

install:
	go install $(LDFLAGS) ./cmd/lore

clean:
	rm -f $(BINARY)

release:
	goreleaser release --clean

snapshot:
	goreleaser release --snapshot --clean --skip=publish
```

`Version` in `internal/cli` defaults to `"dev"`. Goreleaser injects the real version at release time.

### Cobra Root Command Structure

`internal/cli/root.go`:

```go
var rootCmd = &cobra.Command{
    Use:           "lore",
    Short:         "Local-first engineering memory system",
    SilenceUsage:  true,
    SilenceErrors: true,
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        // Skip repo check for "init" and "doctor"
        if cmd.Name() == "init" || cmd.Name() == "doctor" {
            return nil
        }
        _, err := findLoreRoot()
        return err
    },
}

func Execute() error { return rootCmd.Execute() }

func init() {
    rootCmd.AddCommand(initCmd, statusCmd, hookCmd)
    hookCmd.Hidden = true
}
```

`findLoreRoot()` walks parent directories looking for `.lore/`. Returns `ErrNotALoreRepo` if none found. This error maps to exit code 128 in `main.go`.

### `lore init` Step-by-Step Implementation

Function: `func runInit(cmd *cobra.Command, args []string) error`

Steps in order:
1. Call `git.FindGitRoot()` — walk parent directories for `.git/`. Return `error: not a git repository` if not found.
2. Compute `loreRoot = filepath.Join(gitRoot, ".lore")`.
3. Check if `.lore/` already exists. If yes, print `lore: already initialized` and return nil (idempotent).
4. `os.MkdirAll(filepath.Join(loreRoot, "cache"), 0755)` — creates `.lore/` and `.lore/cache/`.
5. Call `store.Open(filepath.Join(loreRoot, "lore.db"))` — runs all migrations, inserts `meta` rows.
6. Write `config.toml` via `config.WriteDefault(loreRoot)`.
7. Call `git.InstallHooks(gitRoot, loreRoot)` — writes three hook scripts.
8. Append `.lore/` to `.gitignore` in `gitRoot` — read the file first; only append if the line is not already present.
9. Print success message to stdout.

### `config.toml` Format (Written by `lore init`)

```toml
# Lore repository configuration
# Generated by lore init — edit to customize

[llm]
provider = "ollama"
model    = "llama3"
endpoint = "http://localhost:11434"

[extraction]
min_tasks           = 1
task_retention_days = 30

[node_resolution]
min_confidence        = 0.4
tag_overlap_threshold = 0.5

[output]
color = true
```

Global config at `~/.config/lore/config.toml` uses identical keys. Per-repo config overrides global. `config.Load(loreRoot)` merges global then local TOML.

### `Config` Go Struct

```go
// internal/config/config.go
type Config struct {
    LLM struct {
        Provider string `toml:"provider"`
        Model    string `toml:"model"`
        Endpoint string `toml:"endpoint"`
    } `toml:"llm"`
    Extraction struct {
        MinTasks          int `toml:"min_tasks"`
        TaskRetentionDays int `toml:"task_retention_days"`
    } `toml:"extraction"`
    NodeResolution struct {
        MinConfidence       float64 `toml:"min_confidence"`
        TagOverlapThreshold float64 `toml:"tag_overlap_threshold"`
    } `toml:"node_resolution"`
    Output struct {
        Color bool `toml:"color"`
    } `toml:"output"`
}
```

`Defaults()` returns a `Config` with the values shown in the TOML above.

### `.golangci.yml` Content

```yaml
linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - goimports
    - misspell

linters-settings:
  goimports:
    local-prefixes: github.com/nishchay/lore

issues:
  exclude-rules:
    - path: _test\.go
      linters: [errcheck]
```

### Git Hook Script Content

`InstallHooks` writes exact bytes for each file and `chmod 0755` each:

**`.git/hooks/post-commit`:**
```sh
#!/bin/sh
# Managed by lore — do not edit manually
lore hook commit
```

**`.git/hooks/post-checkout`:**
```sh
#!/bin/sh
# Managed by lore — do not edit manually
lore hook checkout "$1" "$2" "$3"
```

**`.git/hooks/post-merge`:**
```sh
#!/bin/sh
# Managed by lore — do not edit manually
lore hook merge
```

If a hook file already exists and does NOT start with `# Managed by lore`, read its content and append the lore call on a new line rather than overwriting — this preserves existing hook logic written by other tools.

### `lore init` Success Output

```
Initialized lore repository in /path/to/repo/.lore/
  Database:    .lore/lore.db
  Config:      .lore/config.toml
  Git hooks:   post-commit, post-checkout, post-merge

Run 'lore doctor' to verify the installation.
```

---

## Phase 1 — Storage Layer

**Goal:** Four-tier SQLite schema implemented and tested. All storage operations correct before any task capture is built.

### Schema to Implement

From [SCHEMA.md](./SCHEMA.md):

- `tasks` table (Tier 1, ephemeral, with `trust_level` column — 1/2/3/4)
- `blobs` table (Tier 2, permanent, with `primary_node_id` and observed/inferred separation)
- `blob_tasks` join table (transient)
- `blob_files` table (Tier 2 join, permanent)
- `blob_commands` table (Tier 2 join, permanent)
- `nodes` table (Tier 3, permanent, `created_by="user"` in MVP)
- `graph_nodes` table (Tier 4, derived, includes `Subsystem` kind)
- `graph_edges` table (Tier 4, derived)
- `meta` table (repository metadata)

### Storage Package API

```go
// internal/store/store.go
type Store struct { db *sql.DB }

func Open(path string) (*Store, error)

// Tier 1
func (s *Store) InsertTask(ctx, t task.Task) error
func (s *Store) PendingTasks(ctx) ([]task.Task, error)
func (s *Store) MarkTasksExtracted(ctx, blobID string, taskIDs []string) error
func (s *Store) PurgeExtractedTasks(ctx, olderThan time.Duration) error

// Tier 2
func (s *Store) InsertBlob(ctx, b blob.Blob) error
func (s *Store) BlobByID(ctx, id string) (blob.Blob, error)
func (s *Store) BlobsByFile(ctx, path string) ([]blob.Blob, error)
func (s *Store) BlobLog(ctx, limit int) ([]blob.Blob, error)

// Tier 3
func (s *Store) InsertNode(ctx, n node.Node) error
func (s *Store) AssignBlobToNode(ctx, nodeID, blobID string, confidence float64) error
func (s *Store) NodesByBlob(ctx, blobID string) ([]node.Node, error)

// Tier 4
func (s *Store) InsertGraphNode(ctx, n graph.Node) error
func (s *Store) InsertGraphEdge(ctx, e graph.Edge) error
func (s *Store) UpsertGraphEdge(ctx, e graph.Edge) error
```

### Exit Criteria

All store functions have passing unit tests. `lore status` prints: blob count, node count, graph edge count (all zeros on a fresh repo).

### Migration File Names and Content

All SQL files live in `internal/store/migrations/` and are embedded via `//go:embed migrations/*.sql`.

**`001_initial.sql`** — Creates: `tasks`, `blobs`, `blob_files`, `blob_commands`, `meta` plus all their indexes (exact DDL from SCHEMA.md). Final rows:

```sql
INSERT INTO meta (key, value) VALUES ('schema_version', '1');
INSERT INTO meta (key, value) VALUES ('initialized_at', CAST(strftime('%s','now') AS INTEGER) * 1000000000);
INSERT INTO meta (key, value) VALUES ('git_root', '');
```

**`002_blob_tasks.sql`** — Creates `blob_tasks`:

```sql
CREATE TABLE IF NOT EXISTS blob_tasks (
    blob_id  TEXT NOT NULL,
    task_id  TEXT NOT NULL,
    PRIMARY KEY (blob_id, task_id)
);
CREATE INDEX IF NOT EXISTS idx_blob_tasks_blob ON blob_tasks(blob_id);
CREATE INDEX IF NOT EXISTS idx_blob_tasks_task ON blob_tasks(task_id);
UPDATE meta SET value = '2' WHERE key = 'schema_version';
```

**`003_nodes.sql`** — Creates `nodes` (full DDL from SCHEMA.md). Updates `schema_version` to `'3'`.

**`004_graph.sql`** — Creates `graph_nodes` and `graph_edges` (full DDL from SCHEMA.md). Updates `schema_version` to `'4'`.

### Migration Runner Pattern

```go
// internal/store/migrate.go

//go:embed migrations/*.sql
var migrationsFS embed.FS

func (s *Store) migrate() error {
    // 1. Read current schema_version from meta (0 if table missing)
    // 2. List all *.sql files from migrationsFS, sort by name
    // 3. For each file with index > current version:
    //    - Begin transaction
    //    - db.Exec(sqlContent)
    //    - Commit
    // 4. After all migrations complete, the schema_version is already updated
    //    by each migration's UPDATE statement
}
```

Each migration file runs in its own transaction. Failure rolls back that migration only; prior migrations remain committed. Running `Open()` on an already-migrated database is idempotent — no migrations run if schema_version is current.

### Complete Store Interface (All Method Signatures)

```go
// internal/store/store.go

type Store struct {
    db *sql.DB
}

func Open(path string) (*Store, error)
func (s *Store) Close() error

// Tier 1 — Tasks
func (s *Store) InsertTask(ctx context.Context, t task.Task) error
func (s *Store) PendingTasks(ctx context.Context) ([]task.Task, error)
func (s *Store) MarkTasksExtracted(ctx context.Context, blobID string, taskIDs []string) error
func (s *Store) PurgeExtractedTasks(ctx context.Context, olderThan time.Duration) error
func (s *Store) PurgeOldTasks(ctx context.Context, olderThan time.Duration) error

// Tier 2 — Blobs
func (s *Store) InsertBlobWithRelations(ctx context.Context, b blob.Blob, files []BlobFile, commands []BlobCommand, taskIDs []string) error
func (s *Store) BlobByID(ctx context.Context, id string) (blob.Blob, error)
func (s *Store) ResolveBlobIDPrefix(ctx context.Context, prefix string) (string, error)
func (s *Store) BlobsByFile(ctx context.Context, path string) ([]blob.Blob, error)
func (s *Store) BlobsByFileChron(ctx context.Context, path string) ([]blob.Blob, error)
func (s *Store) BlobLog(ctx context.Context, limit int) ([]blob.Blob, error)
func (s *Store) BlobCount(ctx context.Context) (int, error)
func (s *Store) BlobCountByKind(ctx context.Context) (map[string]int, error)
func (s *Store) BlobCountByTrust(ctx context.Context) (map[int]int, error)
func (s *Store) SetBlobNode(ctx context.Context, blobID, nodeID string) error
func (s *Store) BlobFiles(ctx context.Context, blobID string) ([]BlobFile, error)
func (s *Store) BlobCommands(ctx context.Context, blobID string) ([]BlobCommand, error)
func (s *Store) UnassignedBlobs(ctx context.Context, limit int) ([]blob.Blob, error)
func (s *Store) LastExtractionTime(ctx context.Context) (int64, error)

// Tier 3 — Nodes
func (s *Store) InsertNode(ctx context.Context, n node.Node) error
func (s *Store) NodeByTitle(ctx context.Context, title string) (node.Node, error)
func (s *Store) NodeByID(ctx context.Context, id string) (node.Node, error)
func (s *Store) ListNodes(ctx context.Context) ([]node.Node, error)
func (s *Store) BlobsForNode(ctx context.Context, nodeID string) ([]blob.Blob, error)
func (s *Store) NodeBlobCount(ctx context.Context, nodeID string) (int, error)

// Tier 4 — Graph
func (s *Store) UpsertGraphNode(ctx context.Context, n GraphNode) (string, error)
func (s *Store) UpsertGraphEdge(ctx context.Context, e GraphEdge) error
func (s *Store) GraphNodeCount(ctx context.Context) (int, error)
func (s *Store) GraphEdgeCount(ctx context.Context) (int, error)
func (s *Store) GraphEdgesFrom(ctx context.Context, fromID string) ([]GraphEdge, error)

// Meta
func (s *Store) SetMeta(ctx context.Context, key, value string) error
func (s *Store) GetMeta(ctx context.Context, key string) (string, error)
```

Supporting types in `internal/store/types.go`:

```go
type BlobFile struct {
    BlobID string
    Path   string
    Role   string // "written" | "deleted" | "renamed_from" | "renamed_to"
}

type BlobCommand struct {
    BlobID  string
    Command string
    TS      int64
}

type GraphNode struct {
    ID    string
    Kind  string // "Topic" | "Blob" | "File" | "Commit" | "Concept"
    Label string
    Ref   string
}

type GraphEdge struct {
    ID       string
    FromID   string
    ToID     string
    Relation string
    Weight   int
}
```

### Error Type Definitions

```go
// internal/store/errors.go
var (
    ErrNotFound      = errors.New("not found")
    ErrAlreadyExists = errors.New("already exists")
    ErrAmbiguous     = errors.New("ambiguous prefix")
)

// internal/git/errors.go
var ErrNotAGitRepo = errors.New("not a git repository")

// internal/cli/errors.go
var (
    ErrNotALoreRepo = errors.New("not a lore repository (or any parent up to mount point /)")
    ErrUsage        = errors.New("usage error")
)
```

Exit-code switch in `cmd/lore/main.go`:

```go
func main() {
    if err := cli.Execute(); err != nil {
        switch {
        case errors.Is(err, cli.ErrNotALoreRepo):
            fmt.Fprintf(os.Stderr, "error: not a lore repository (or any parent up to mount point /)\nhint: run 'lore init' to initialize\n")
            os.Exit(128)
        case errors.Is(err, cli.ErrUsage):
            os.Exit(2)
        default:
            fmt.Fprintf(os.Stderr, "error: %v\n", err)
            os.Exit(1)
        }
    }
}
```

### Test Cases Per Method

File: `internal/store/store_test.go`. All tests use `:memory:` SQLite:

```go
s, _ := store.Open(":memory:")
defer s.Close()
```

Required test functions:

```go
func TestInsertAndFetchTask(t *testing.T)             // insert task, PendingTasks returns it
func TestMarkTasksExtracted(t *testing.T)              // after mark, PendingTasks omits it
func TestPurgeExtractedTasks(t *testing.T)             // purge with 0 duration removes extracted tasks
func TestInsertBlobAndFetch(t *testing.T)              // InsertBlobWithRelations + BlobByID round-trip
func TestBlobsByFile(t *testing.T)                     // insert blob+files, query by path
func TestBlobsByFile_SuffixMatch(t *testing.T)         // "oauth.go" matches "internal/auth/oauth.go"
func TestBlobLog_OrderNewestFirst(t *testing.T)        // multiple blobs, verify descending order
func TestInsertNode_UniqueTitle(t *testing.T)          // second insert same title → ErrAlreadyExists
func TestSetBlobNode(t *testing.T)                     // sets primary_node_id on blob
func TestUpsertGraphEdge_Weight(t *testing.T)          // same edge twice → weight=2
func TestMigration_RunsInOrder(t *testing.T)           // fresh :memory: db runs all 4 migrations
func TestMigration_Idempotent(t *testing.T)            // Open on same :memory: db does not re-run
func TestBlobCountByKind(t *testing.T)                 // 2 Feature + 1 BugFix → correct map
func TestBlobCountByTrust(t *testing.T)                // trust=2 and trust=4 blobs → correct map
func TestResolveBlobIDPrefix_Ambiguous(t *testing.T)   // two blobs with same prefix → ErrAmbiguous
func TestResolveBlobIDPrefix_NotFound(t *testing.T)    // unknown prefix → ErrNotFound
func TestInsertBlobWithRelations_AtomicRollback(t *testing.T) // force failure mid-write, verify no partial insert
```

### `lore status` Zero-State Output

On a fresh repo immediately after `lore init` with no commits:

```
Repository: /path/to/repo
Initialized: 2026-05-23

Blobs: 0

Subsystems (Nodes): 0

Pending Tasks: 0

LLM: ollama/llama3 (not checked — run 'lore doctor')
```

This output must remain valid at all subsequent phases. Zero counts must never error.

---

## Phase 2 — Task Capture

**Goal:** Git hooks installed by `lore init` automatically record Tasks. No separate daemon required.

### Observation Architecture

MVP observation is exclusively via Git hooks. No file system watcher. No `lore watch` command.

Agent `FileWrite` and `Command` tasks come from agent hooks (Phase 6) and `lore record`.

**Rationale:** Commits are the natural high-signal event boundary. File system watching introduces daemon complexity that conflicts with Lore's Git-like design. Commit-boundary signals are sufficient for quality Blob extraction.

### Git Hooks Installed by `lore init`

```bash
.git/hooks/post-commit     # emits CommitCreated task
.git/hooks/post-checkout   # emits BranchSwitch task
.git/hooks/post-merge      # emits MergeEvent task
```

Hooks call `lore hook <kind> [args]` — a thin internal command not shown in user-facing help.

For `CommitCreated`, `lore hook commit` reads:
- Commit SHA, message, author via `git log -1 --format=...`
- Changed files via `git diff-tree --no-commit-id -r --name-only HEAD`

These are stored as `trust_level=1` (GroundTruth).

`prepare-commit-msg` is **not** installed. Lore does not modify commit messages.

### Manual Annotation

```bash
lore record "investigating JWT expiry bug in session.go"
```

Emits a `Note` task with `trust_level=1`. Included in the next blob extraction.

### Exit Criteria

Initialize a test repo with `lore init`. Make three commits across two branches. `lore status` shows three `CommitCreated` tasks and two `BranchSwitch` tasks.

### Task Go Struct (Complete)

```go
// internal/task/task.go
type Task struct {
    ID            string
    Kind          TaskKind
    Path          string   // file path (if applicable)
    Detail        string   // command text, commit hash, recap JSON, note text
    Source        string   // "human" | "agent:claude" | "agent:cursor" | "agent:openhands" | "ci" | "hook"
    TrustLevel    int      // 1 | 2 | 3 | 4
    TS            int64    // unix nanoseconds
    Extracted     bool
    ExtractedInto string   // blobs.id (empty until extracted)
}

// internal/task/kind.go
type TaskKind string

const (
    KindFileWrite     TaskKind = "FileWrite"
    KindFileDelete    TaskKind = "FileDelete"
    KindFileRename    TaskKind = "FileRename"
    KindCommand       TaskKind = "Command"
    KindCommitCreated TaskKind = "CommitCreated"
    KindBranchSwitch  TaskKind = "BranchSwitch"
    KindMergeEvent    TaskKind = "MergeEvent"
    KindSearchQuery   TaskKind = "SearchQuery"
    KindAgentAction   TaskKind = "AgentAction"
    KindNote          TaskKind = "Note"
    KindAgentRecap    TaskKind = "AgentRecap"
)
```

### `lore hook commit` Step-by-Step

Function: `func runHookCommit(ctx context.Context, s *store.Store) error`

1. Run `git log -1 --format=%H%n%s%n%an%n%ae%n%at` — parse into: sha, subject, authorName, authorEmail, authorUnixSec. All via `exec.CommandContext(ctx, "git", ...)` with `Dir` set to git root.
2. Run `git diff-tree --no-commit-id -r --name-status HEAD` — parse lines. Format: `<status>\t<path>` or `<status>\t<oldpath>\t<newpath>` for renames. Status values: `A`=added, `M`=modified, `D`=deleted, `R`=renamed.
3. Create `Task{Kind: KindCommitCreated, Detail: sha + "|" + subject, Source: "hook", TrustLevel: 1, TS: time.Now().UnixNano()}`.
4. For each added/modified file: `Task{Kind: KindFileWrite, Path: path, Source: "hook", TrustLevel: 1, TS: now}`.
5. For each deleted file: `Task{Kind: KindFileDelete, Path: path, Source: "hook", TrustLevel: 1, TS: now}`.
6. For each rename: `Task{Kind: KindFileRename, Detail: oldPath + " -> " + newPath, Source: "hook", TrustLevel: 1, TS: now}`.
7. Insert all tasks in a single `InsertBlobWithRelations`-style transaction (or a dedicated `InsertTasks(ctx, []task.Task)` method).
8. Immediately call `blob.ExtractIfReady(ctx, s, cfg)` — extraction runs synchronously in the hook, not deferred.

### `lore hook checkout` Step-by-Step

Arguments received from shell: `$1`=prevRef, `$2`=newRef, `$3`=flag (1=branch, 0=file).

1. If `flag != "1"`, return nil — file checkouts do not emit tasks.
2. Run `git branch --show-current` to get new branch name.
3. Emit `Task{Kind: KindBranchSwitch, Detail: prevRef + "->" + newBranch, Source: "hook", TrustLevel: 1}`.
4. Insert. **Do not trigger extraction** — BranchSwitch marks a window boundary only.

### `lore hook merge` Step-by-Step

1. Read `<gitRoot>/.git/MERGE_HEAD` to get merged commit SHA. If file absent (fast-forward), use empty string.
2. Emit `Task{Kind: KindMergeEvent, Detail: mergedSHA, Source: "hook", TrustLevel: 1}`.
3. Insert. **Do not trigger extraction.**

### `lore record` Implementation

```go
// internal/cli/record.go
var recordCmd = &cobra.Command{
    Use:   "record <note>",
    Short: "Emit a Note task (developer annotation)",
    Args:  cobra.MinimumNArgs(1),
    RunE:  runRecord,
}

func runRecord(cmd *cobra.Command, args []string) error {
    note := strings.Join(args, " ")
    t := task.Task{
        ID:         uuid.NewString(),
        Kind:       task.KindNote,
        Detail:     note,
        Source:     "human",
        TrustLevel: 1,
        TS:         time.Now().UnixNano(),
    }
    // open store, insert task
    // print: "Recorded: <note>"
}
```

### Phase 2 Test Scenarios

All tests use real temp git repos (`t.TempDir()` + `exec.Command("git", "init", ...)`). No mocking of `exec.Command`.

```go
func TestHookCommit_InsertsCommitTask(t *testing.T)
func TestHookCommit_InsertsFileWriteTasks(t *testing.T)
func TestHookCommit_InsertsFileDeleteTask(t *testing.T)
func TestHookCheckout_BranchFlag1_EmitsBranchSwitch(t *testing.T)
func TestHookCheckout_FileFlag0_NoEmission(t *testing.T)
func TestHookMerge_EmitsMergeEvent(t *testing.T)
func TestRecord_InsertsNoteTask(t *testing.T)
```

---

## Phase 3 — Blob Extraction + Agent Recap Ingestion

**Goal:** After each commit, Lore automatically extracts a Blob from accumulated Tasks. Agent recaps are ingested as Trust Level 2 when present.

### Extraction Trigger

Extraction triggers automatically on:
1. `CommitCreated` task (primary trigger — the `post-commit` hook triggers extraction internally)
2. `BranchSwitch` task (closes the current extraction window)

No `lore extract` command required. No session concept.

### Extraction Window

```sql
SELECT * FROM tasks WHERE extracted = 0 ORDER BY ts ASC
```

Extraction proceeds if the window contains at least one `CommitCreated` task.

### Recap Lookup (Priority Path)

Before invoking AI, check for an `AgentRecap` task in the window:

```sql
SELECT * FROM tasks
WHERE kind = 'AgentRecap'
  AND extracted = 0
ORDER BY ts DESC
LIMIT 1;
```

If found:
- Parse the `detail` JSON field
- Use `user_intent`, `summary`, `recap`, `kind`, `tags` directly
- Set `trust_level=2`, `ai_source` from the recap's `source` field
- Skip Ollama/heuristic entirely for this Blob

If not found → proceed to Lore fallback inference (Ollama or heuristic).

### Blob Extraction Pipeline

```
PendingTasks
    │
    ▼
WindowBuilder
    │  fills: started_at, ended_at, files_written,
    │          commands, commit_start, commit_end
    ▼
RecapLookup
    ├── AgentRecap found → RecapIngester  → Blob (trust=2)
    └── Not found        → PromptBuilder
                              ↓
                           LLMClient (Ollama) or HeuristicExtractor
                              ↓
                           ResponseParser   → Blob (trust=4)
    ▼
Store.InsertBlobWithRelations()
    ▼
GraphBuilder.UpdateFromBlob()
    ▼
Store.MarkTasksExtracted()
```

### `lore log` (first real query command)

```
$ lore log

abc1234  OAuth Provider Impl   Feature   2026-05-20  [AgentTruth]   3 files
def5678  Fix JWT expiry        BugFix    2026-05-18  [LoreInferred]  2 files
ghi9012  Auth middleware       Refactor  2026-05-15  [AgentTruth]   5 files
```

### LLM Config

```toml
# .lore/config.toml
[llm]
provider = "ollama"
model    = "llama3"
endpoint = "http://localhost:11434"
```

If Ollama unreachable: fall back to heuristic extractor, log single warning.

### Exit Criteria

On a repository with five commits across two topics, `lore log` shows two or three Blobs. At least one Blob should have `trust=AgentTruth` when Claude Code hooks are installed. Heuristic Blobs show `trust=LoreInferred`. `lore show <id>` correctly separates observed and interpreted sections.

### WindowBuilder Struct

```go
// internal/blob/window.go
type Window struct {
    Tasks        []task.Task
    StartedAt    int64     // min TS across all tasks
    EndedAt      int64     // max TS across all tasks
    CommitStart  string    // SHA of earliest CommitCreated in window
    CommitEnd    string    // SHA of latest CommitCreated in window
    FilesWritten []string  // deduplicated paths from KindFileWrite tasks
    FilesDeleted []string  // deduplicated paths from KindFileDelete tasks
    Commands     []string  // unique command strings from KindCommand tasks
    CommitMsgs   []string  // commit messages from KindCommitCreated tasks
    Sources      []string  // unique source values across all tasks
    HasCommit    bool      // true if at least one KindCommitCreated exists
    RecapTask    *task.Task // most recent KindAgentRecap by TS, nil if absent
}

func BuildWindow(tasks []task.Task) Window
```

`BuildWindow` iterates tasks once. Deduplication of `FilesWritten` uses `map[string]struct{}`. `RecapTask` is set to the last `KindAgentRecap` by TS. `HasCommit` is set true on first `KindCommitCreated` encountered.

Extraction only proceeds if `w.HasCommit == true`. Tasks without a commit are retained for the next window.

### RecapLookup

`RecapLookup` is pure Go logic operating on `Window.RecapTask` — no separate SQL query. After `BuildWindow`, if `w.RecapTask != nil`, call `RecapIngester.Ingest(w, recap)`. Otherwise proceed to `PromptBuilder`.

### RecapIngester Field Mapping

```go
// internal/blob/recap.go
type AgentRecapPayload struct {
    UserIntent string   `json:"user_intent"`
    Summary    string   `json:"summary"`
    Recap      string   `json:"recap"`
    Kind       string   `json:"kind"`
    Tags       []string `json:"tags"`
    Source     string   `json:"source"` // "agent:claude" | "agent:cursor" etc.
}

func IngestRecap(w Window, payload AgentRecapPayload) blob.Blob
```

`IngestRecap` sets:
- Interpreted fields: `Kind`, `Title` (first 100 chars of `UserIntent`, or first sentence of `Summary` if empty), `Summary`, `Recap`, `UserIntent`, `Tags`
- Provenance: `TrustLevel=2`, `AISource=payload.Source`
- Observed fields from `w`: `StartedAt`, `EndedAt`, `CommitStart`, `CommitEnd`

If `json.Unmarshal` fails on `w.RecapTask.Detail`: log warning to stderr and fall through to `HeuristicExtractor`. Never panic on malformed JSON.

### PromptBuilder Context

```go
// internal/blob/prompt.go
func BuildPrompt(w Window) string
```

Exact prompt template:

```
You are analyzing a software development session. Based on the observed actions below,
generate a structured JSON summary. Respond ONLY with valid JSON matching the schema.

Time range: <human-readable start> to <human-readable end>
Duration: <N minutes / N hours>

Files written (<count>):
<path>
...

Files deleted (<count>):
<path>
...

Commands executed (<count>):
<command>
...

Commit messages:
- <message1>
- <message2>

Sources: <comma-separated unique sources>

Respond with this exact JSON schema:
{
  "title": "string (max 100 chars)",
  "summary": "string (max 500 chars)",
  "recap": "string (max 300 chars)",
  "user_intent": "string (max 200 chars)",
  "kind": "Feature | BugFix | Migration | Investigation | Refactor | Architecture | Review | Incident",
  "tags": ["string"]
}
```

Raw file contents are never included. Commit SHAs are never included (only messages).

### LLMClient Ollama HTTP Request/Response

```go
// internal/blob/llm.go
var ErrLLMUnavailable = errors.New("LLM unavailable")

type LLMClient struct {
    Endpoint string
    Model    string
    Timeout  time.Duration // default 30s
}

type ollamaRequest struct {
    Model  string `json:"model"`
    Prompt string `json:"prompt"`
    Stream bool   `json:"stream"` // always false
    Format string `json:"format"` // "json"
}

type ollamaResponse struct {
    Response string `json:"response"`
    Done     bool   `json:"done"`
}

func (c *LLMClient) Complete(ctx context.Context, prompt string) (string, error)
func (c *LLMClient) Ping(ctx context.Context) bool  // GET /api/tags → true if HTTP 200
```

Request: `POST <endpoint>/api/generate` with JSON body. Response: single JSON object. Parse `ollamaResponse.Response` as the raw content string.

On timeout or connection refused: return `("", ErrLLMUnavailable)`. Caller immediately falls through to `HeuristicExtractor`.

### HeuristicExtractor Rules (Exact)

```go
// internal/blob/heuristic.go
func HeuristicExtract(w Window) blob.Blob
```

Kind inference — first matching rule wins, applied to each commit message (lowercased):

| Condition | Kind |
|-----------|------|
| Any commit message has prefix `fix` or `bug` | `BugFix` |
| Any commit message has prefix `feat` or `add` | `Feature` |
| Any commit message contains substring `migrat` | `Migration` |
| Any commit message has prefix `refactor` or `chore` | `Refactor` |
| Any file path in `FilesWritten` contains `/arch/`, `/design/`, or `/adr/` | `Architecture` |
| `HasCommit == false` and only `KindCommand` tasks present | `Investigation` |
| Default (no rule matched) | `Feature` |

Title: first commit message in `w.CommitMsgs`, truncated to 72 characters. If no commit message: `"Work session " + time.Unix(0, w.StartedAt).Format("2006-01-02")`.

Summary: `fmt.Sprintf("Modified %d file(s). Ran %d command(s). Produced %d commit(s).", len(w.FilesWritten), len(w.Commands), commitCount)`.

Tags: unique top-level directory names from `w.FilesWritten`. E.g. `internal/auth/oauth.go` → tag `auth` (from `strings.Split(path, "/")[0]` after stripping any leading `./`). Max 10 tags. Tags are lowercased and deduplicated.

Recap: empty string. Heuristic cannot infer significance.

Result: `TrustLevel=4`, `AISource="lore:heuristic"`.

### ResponseParser JSON Schema

```go
// internal/blob/parser.go
type LLMResponse struct {
    Title      string   `json:"title"`
    Summary    string   `json:"summary"`
    Recap      string   `json:"recap"`
    UserIntent string   `json:"user_intent"`
    Kind       string   `json:"kind"`
    Tags       []string `json:"tags"`
}

var ErrParseFailure = errors.New("failed to parse LLM response")

func ParseLLMResponse(raw string) (LLMResponse, error)
```

Validation rules enforced by `ParseLLMResponse`:
- `Title`: truncate to 100 chars if longer. If empty after truncation: derive from first 72 chars of `Summary`.
- `Kind`: validate against the 8 valid `BlobKind` values. Default to `Feature` if unrecognized or empty.
- `Tags`: max 20 entries; each tag lowercased, whitespace-stripped, empty entries removed.
- `Summary`: truncate to 500 chars.
- `Recap`: truncate to 300 chars.

If `json.Unmarshal` fails entirely: return `("", ErrParseFailure)` → caller uses `HeuristicExtractor`.

### `Blob` Go Struct and `BlobKind` Enum

```go
// internal/blob/blob.go
type BlobKind string

const (
    KindFeature       BlobKind = "Feature"
    KindBugFix        BlobKind = "BugFix"
    KindMigration     BlobKind = "Migration"
    KindInvestigation BlobKind = "Investigation"
    KindRefactor      BlobKind = "Refactor"
    KindArchitecture  BlobKind = "Architecture"
    KindReview        BlobKind = "Review"
    KindIncident      BlobKind = "Incident"
)

type Blob struct {
    ID                string
    Kind              BlobKind
    Title             string
    Summary           string
    Recap             string
    UserIntent        string
    InferredReasoning string
    Tags              []string  // stored as JSON array in DB
    TrustLevel        int
    AISource          string
    StartedAt         int64
    EndedAt           int64
    CommitStart       string
    CommitEnd         string
    PrimaryNodeID     string
    CreatedAt         int64
}
```

### GraphBuilder.UpdateFromBlob Logic

```go
// internal/graph/builder.go
func UpdateFromBlob(ctx context.Context, s *store.Store, b blob.Blob) error
```

Six steps, all using `UpsertGraphNode` / `UpsertGraphEdge`:

1. Upsert `GraphNode{Kind: "Blob", Label: b.Title, Ref: b.ID}` → get its graph node ID.
2. If `b.CommitEnd != ""`: upsert `GraphNode{Kind: "Commit", Label: b.CommitEnd, Ref: b.CommitEnd}`. Upsert edge `Blob → Commit` with `relation="Produced"`.
3. For each path in `blob_files` with `role="written"`: upsert `GraphNode{Kind: "File", Label: path, Ref: path}`. Upsert edge `Blob → File` with `relation="Modified"`.
4. For each path with `role="deleted"`: upsert `GraphNode{Kind: "File", ...}`. Upsert edge `Blob → File` with `relation="Deleted"`.
5. For each tag in `b.Tags`: upsert `GraphNode{Kind: "Concept", Label: tag, Ref: tag}`.
6. If `b.PrimaryNodeID != ""`: upsert `GraphNode{Kind: "Topic", Label: nodeTitle, Ref: b.PrimaryNodeID}`. Upsert edge `Topic → Blob` with `relation="Contains"`.

`UpsertGraphEdge` SQL: `INSERT INTO graph_edges ... ON CONFLICT(from_id, to_id, relation) DO UPDATE SET weight = weight + 1`.

### `lore log` Column Widths and Format (Exact)

```
<id:8>  <title:24>  <kind:14>  <date:10>  <trust:14>  <filecount:7>
```

- **ID**: first 8 characters of the UUID
- **Title**: left-aligned, truncated to 21 chars + `...` if over 24
- **Kind**: left-aligned, padded to 14 chars
- **Date**: `ended_at` formatted as `2006-01-02`
- **Trust**: `[AgentTruth]` or `[LoreInferred]`, padded to 14 chars
- **File count**: `N files`, right-aligned to 7 chars

Color rules (only when `UseColor()` returns true):
- `[AgentTruth]` → ANSI green (`\033[32m` + reset)
- `[LoreInferred]` → ANSI yellow (`\033[33m` + reset)

Default limit: 20. Override with `--limit N`.

### `lore show <id>` Section Layout (Exact)

```
ID:           <id>
Title:        <title>
Kind:         <kind>
Trust:        <AgentTruth|LoreInferred> (source: <ai_source>)

── Observed ────────────────────────────────────────
Started:      <started_at as "2006-01-02 15:04">
Ended:        <ended_at as "2006-01-02 15:04">
Commits:      <commit_start>..<commit_end>

Files Modified:
  <path>
  <path>

Files Deleted:
  <path>

Commands:
  <command>

── Interpreted ─────────────────────────────────────
User Intent:  <user_intent>  (or: "(none)" if empty)
Summary:      <summary — word-wrapped at 80 cols, continuation lines indented 14 spaces>
Recap:        <recap>        (or: "(none)" if empty)
Tags:         <tag1>, <tag2>, ...

── Part of ─────────────────────────────────────────
Node: <node title>
      (or: "(unassigned)" + hint line if primary_node_id is null)
```

Separator line width: 52 characters after `──`. If `--json` flag: output single JSON object matching `BlobJSON` struct (see Phase 5).

### Phase 3 Test Scenarios

```go
func TestBuildWindow_SetsTimeRange(t *testing.T)
func TestBuildWindow_FindsRecapTask(t *testing.T)
func TestBuildWindow_HasCommit_False_WhenNoCommit(t *testing.T)
func TestBuildWindow_DeduplicatesFiles(t *testing.T)
func TestIngestRecap_SetsAgentTrust(t *testing.T)
func TestIngestRecap_MalformedJSON_FallsToHeuristic(t *testing.T)
func TestHeuristicExtract_KindFromCommitMessage(t *testing.T)    // "fix: ..." → BugFix
func TestHeuristicExtract_DefaultKind_IsFeature(t *testing.T)
func TestHeuristicExtract_TagsFromDirectoryNames(t *testing.T)
func TestHeuristicExtract_TitleTruncation(t *testing.T)
func TestParseLLMResponse_ValidJSON(t *testing.T)
func TestParseLLMResponse_InvalidKindDefaultsToFeature(t *testing.T)
func TestParseLLMResponse_TitleTruncation(t *testing.T)
func TestUpdateFromBlob_CreatesGraphNodes(t *testing.T)
func TestUpdateFromBlob_EdgeWeightIncrement(t *testing.T)
func TestExtractIfReady_SkipsWhenNoCommit(t *testing.T)
func TestExtractIfReady_AgentRecapPath_SkipsOllama(t *testing.T)
func TestExtractIfReady_HeuristicPath_WhenOllamaUnavailable(t *testing.T)
```

---

## Phase 4 — Node Creation

**Goal:** Users define their repository's subsystems as Nodes. Blobs are assigned to those Nodes. Automatic Node generation is explicitly deferred.

### What Nodes Are

Nodes represent stable repository subsystems — long-lived named areas of the codebase.

```
Authentication
Billing
Session Management
API Gateway
```

Nodes are NOT investigations, migrations, bug fixes, or implementation tasks. Those are Blobs.

### Commands Introduced in This Phase

```bash
lore node create <name>         # create a subsystem node
lore node list                  # list all subsystems with blob counts
lore node show <name>           # show blobs assigned to this subsystem
lore assign <blob-id> <node>    # assign a blob to a subsystem
```

`lore assign` sets `blobs.primary_node_id` with `trust_level=3` (HumanAssertion). Human assertions are preferred over any Lore inference.

### Node Lifecycle

```
User runs: lore node create Authentication
    │
    ▼
Node created (created_by="user")
    │
User runs: lore assign abc1234 Authentication
    │
    ▼
blobs.primary_node_id = <node-id>   (trust_level=3)
    │
    ▼
Graph updated: Subsystem node → Contains → Blob node
```

### Automatic Assignment (Heuristic, Post-MVP)

Automatic Node assignment via clustering or AI inference is explicitly deferred. In MVP, all Node assignments are human-authored.

Post-MVP, `lore node suggest` will propose assignments based on file path patterns and tags. Suggestions will require `lore assign` confirmation to persist at trust_level=3.

### `lore graph` after Phase 4

```
$ lore graph

Subsystem: Authentication
├── OAuth provider impl   (Feature, 2026-04-22)  [AgentTruth]
├── Auth investigation    (Investigation, 2026-04-10)  [LoreInferred]
└── OAuth test coverage   (BugFix, 2026-04-25)  [AgentTruth]

Subsystem: Billing
└── Payment processor     (Feature, 2026-05-01)  [AgentTruth]

Unassigned Blobs: 1
└── JWT Expiry Fix  (BugFix, 2026-05-10)  [LoreInferred]
    hint: use 'lore assign <id> Authentication'
```

### Exit Criteria

User creates two Nodes (`lore node create Authentication`, `lore node create Billing`). Assigns existing Blobs to each. `lore graph` shows the correct subsystem → blob hierarchy. `lore node list` shows both subsystems with correct blob counts. `lore node show Authentication` lists all assigned blobs.

### `Node` Go Struct

```go
// internal/node/node.go
type Node struct {
    ID          string
    Title       string
    Description string
    Status      string // "active" | "archived"
    CreatedBy   string // "user" | "agent_recap" | "lore_inference"
    CreatedAt   int64
    UpdatedAt   int64
}
```

### `lore node create` Validation

```go
// internal/cli/node.go
func runNodeCreate(cmd *cobra.Command, args []string) error
```

Steps in order:
1. Trim whitespace from `args[0]`. If empty after trim: `error: node name cannot be empty`.
2. Length check: max 100 characters. If exceeded: `error: node name too long (max 100 characters)`.
3. Call `store.NodeByTitle(ctx, title)`. If found: `error: node '<title>' already exists`.
4. Generate UUID. Create `Node{Title: title, Status: "active", CreatedBy: "user", CreatedAt: now, UpdatedAt: now}`.
5. `store.InsertNode(ctx, n)`.
6. Call `graph.UpdateFromNode(ctx, store, n)` — creates `GraphNode{Kind: "Topic", Label: n.Title, Ref: n.ID}`.
7. Print: `Created node: <title>`.

Multi-word names work naturally: `lore node create "Session Management"` — cobra receives the quoted string as a single arg.

### `lore assign` Update Path

```go
// internal/cli/assign.go
func runAssign(cmd *cobra.Command, args []string) error
// args[0] = blob ID prefix (min 7 chars)
// args[1] = node title (case-insensitive match)
```

Steps in order:
1. Validate `len(args[0]) >= 7`, else: `error: blob ID prefix too short (minimum 7 characters)`.
2. Call `store.ResolveBlobIDPrefix(ctx, args[0])` — returns full ID or `ErrNotFound`/`ErrAmbiguous`.
3. Call `store.NodeByTitle` with case-insensitive match: `LOWER(title) = LOWER(?)`. If not found: `error: no node '<args[1]>'` + `hint: run 'lore node list' to see available nodes`.
4. Call `store.SetBlobNode(ctx, blobID, node.ID)` — `UPDATE blobs SET primary_node_id = ? WHERE id = ?`.
5. Call `graph.UpdateAssignment(ctx, store, blobID, node.ID)` — upsert `Contains` edge.
6. Print: `Assigned '<blob title>' to node '<node title>'`.

### `graph.UpdateAssignment` SQL

```go
// internal/graph/builder.go
func UpdateAssignment(ctx context.Context, s *store.Store, blobID, nodeID string) error
```

1. `store.UpsertGraphNode(ctx, GraphNode{Kind: "Topic", Ref: nodeID, Label: nodeTitle})` → get topic graph node ID.
2. `store.UpsertGraphNode(ctx, GraphNode{Kind: "Blob", Ref: blobID})` → get blob graph node ID.
3. `store.UpsertGraphEdge(ctx, GraphEdge{FromID: topicGNodeID, ToID: blobGNodeID, Relation: "Contains"})`.

The `UpsertGraphEdge` SQL:

```sql
INSERT INTO graph_edges (id, from_id, to_id, relation, weight)
VALUES (?, ?, ?, ?, 1)
ON CONFLICT(from_id, to_id, relation) DO UPDATE SET weight = weight + 1;
```

### `lore node list` Output

```
Subsystems (3):

  Authentication      8 blobs   active
  Billing             3 blobs   active
  Session Management  1 blob    active

hint: use 'lore assign <blob-id> <subsystem>' to assign unassigned blobs
      use 'lore node show <name>' to see blobs in a subsystem
```

Column widths: title left-aligned 24 chars, blob count right-aligned 8 chars, status 8 chars. Singular "blob" vs plural "blobs" based on count.

### `lore node show <name>` Output

```
Node: Authentication
Description: (none)
Status: active
Blobs: 8

  abc1234  OAuth Provider Impl   Feature     2026-05-20  [AgentTruth]
  def5678  Auth middleware       Refactor    2026-05-18  [AgentTruth]
  ghi9012  Fix JWT expiry        BugFix      2026-05-15  [LoreInferred]
  ...
```

Same column format as `lore log`. Uses `store.BlobsForNode(ctx, node.ID)`.

### Phase 4 Test Scenarios

```go
func TestNodeCreate_Succeeds(t *testing.T)
func TestNodeCreate_DuplicateTitle_Errors(t *testing.T)
func TestNodeCreate_EmptyName_Errors(t *testing.T)
func TestNodeCreate_NameTooLong_Errors(t *testing.T)
func TestNodeCreate_CreatesTopicGraphNode(t *testing.T)
func TestAssign_SetsBlobNode(t *testing.T)
func TestAssign_CaseInsensitiveNodeMatch(t *testing.T)
func TestAssign_PrefixBlobIDResolution(t *testing.T)
func TestAssign_AmbiguousPrefix_Errors(t *testing.T)
func TestAssign_UnknownNode_Errors(t *testing.T)
func TestAssign_PrefixTooShort_Errors(t *testing.T)
func TestUpdateAssignment_CreatesContainsEdge(t *testing.T)
func TestUpdateAssignment_WeightIncrements(t *testing.T)
```

---

## Phase 5 — Query Interface

**Goal:** All MVP commands work correctly on real repositories.

### `lore why <file>`

Shows Blobs that modified the given file, newest first. Includes trust level and Node membership.

### `lore trace <file>`

Chronological Blob history for a file.

### `lore show <id>`

Full Blob detail with clear `── Observed ──` and `── Interpreted ──` sections. Shows Node membership.

See [rules/CLI.md](./rules/CLI.md) for exact output format specifications.

### `lore graph`

ASCII graph showing Nodes → Blobs → Files. Unassigned Blobs listed separately.

### `lore status`

Counts Blobs by kind and by trust level. Shows Nodes with their Blob counts. Shows pending task count.

### Exit Criteria

All commands work on a repository with at least 10 real Blobs and 2 Nodes. `--json` flag works on all query commands. Trust levels display correctly.

### `lore why <file>` SQL (Exact)

```sql
SELECT b.id, b.kind, b.title, b.summary, b.trust_level, b.ai_source,
       b.commit_start, b.commit_end, b.started_at, b.ended_at,
       b.primary_node_id, n.title as node_title
FROM blobs b
JOIN blob_files bf ON b.id = bf.blob_id
LEFT JOIN nodes n ON b.primary_node_id = n.id
WHERE bf.path = ? OR bf.path LIKE '%/' || ?
ORDER BY b.started_at DESC;
```

Both the exact path and a suffix match are tried in a single query. If zero results:

```
error: no blobs found for '<file>'
hint: run 'lore status' to see what lore has captured
```

### `lore trace <file>` SQL

Identical to `lore why` but `ORDER BY b.started_at ASC`. Output labeled `History of <file>:` at top.

### `lore show <id>` Joins (Exact SQL Sequence)

Four sequential queries:

```sql
-- 1. Fetch blob
SELECT * FROM blobs WHERE id = ?;

-- 2. Fetch files (sorted for stable output)
SELECT path, role FROM blob_files WHERE blob_id = ? ORDER BY role, path;

-- 3. Fetch commands (sorted by time)
SELECT command, ts FROM blob_commands WHERE blob_id = ? ORDER BY ts;

-- 4. Fetch node if primary_node_id is not null
SELECT title FROM nodes WHERE id = ?;
```

Assembled into a `BlobDetail` struct before rendering.

### `lore graph` Rendering Algorithm (Exact)

```go
func RenderGraph(ctx context.Context, s *store.Store, w io.Writer) error
```

Nine steps:

1. `store.ListNodes(ctx)` — get all active nodes.
2. For each node: `store.BlobsForNode(ctx, node.ID)` — get blobs newest-first, take **first 3**.
3. For each blob: `store.BlobFiles(ctx, blob.ID)` — get all files.
4. Print `Subsystem: <node.Title>` as section header.
5. For each blob (up to 3): print tree row with `├──` for non-last, `└──` for last.
6. For each blob's files: print `│   ├── Modified  <path>` or `Deleted` under each blob.
7. After all nodes: `store.UnassignedBlobs(ctx, 5)` — newest-first, max 5.
8. Print `Unassigned Blobs:` section with same tree format.
9. For the last unassigned blob: append `hint: use 'lore assign <id> <subsystem>'` on the line after.

Tree characters:
- Not-last sibling: `├── `
- Last sibling: `└── `
- Continuation under non-last: `│   `
- Continuation under last: `    `

### `lore status` Aggregation Queries (Exact)

Eight queries run sequentially:

```sql
-- Total blob count
SELECT COUNT(*) FROM blobs;

-- Blob counts by kind
SELECT kind, COUNT(*) as cnt FROM blobs GROUP BY kind ORDER BY cnt DESC;

-- Blob counts by trust level
SELECT trust_level, COUNT(*) as cnt FROM blobs GROUP BY trust_level;

-- Active node count
SELECT COUNT(*) FROM nodes WHERE status = 'active';

-- Nodes with blob counts (for display)
SELECT n.id, n.title, n.status, COUNT(b.id) as blob_count
FROM nodes n
LEFT JOIN blobs b ON b.primary_node_id = n.id
WHERE n.status = 'active'
GROUP BY n.id
ORDER BY blob_count DESC;

-- Unassigned blob count
SELECT COUNT(*) FROM blobs WHERE primary_node_id IS NULL;

-- Pending task count
SELECT COUNT(*) FROM tasks WHERE extracted = 0;

-- Last extraction time
SELECT MAX(created_at) FROM blobs;
```

LLM availability: call `LLMClient.Ping(ctx)` — `GET /api/tags` → HTTP 200 within 3s.

### `--json` Output Struct Shapes

```go
// internal/cli/json.go

type BlobJSON struct {
    ID                string           `json:"id"`
    Kind              string           `json:"kind"`
    Title             string           `json:"title"`
    Summary           string           `json:"summary"`
    Recap             string           `json:"recap"`
    UserIntent        string           `json:"user_intent"`
    InferredReasoning string           `json:"inferred_reasoning,omitempty"`
    Tags              []string         `json:"tags"`
    TrustLevel        int              `json:"trust_level"`
    AISource          string           `json:"ai_source"`
    StartedAt         int64            `json:"started_at"`
    EndedAt           int64            `json:"ended_at"`
    CommitStart       string           `json:"commit_start"`
    CommitEnd         string           `json:"commit_end"`
    PrimaryNodeID     string           `json:"primary_node_id,omitempty"`
    CreatedAt         int64            `json:"created_at"`
    Files             []BlobFileJSON   `json:"files,omitempty"`
    Commands          []BlobCommandJSON `json:"commands,omitempty"`
    Node              *NodeRefJSON     `json:"node,omitempty"`
}

type BlobFileJSON struct {
    Path string `json:"path"`
    Role string `json:"role"`
}

type BlobCommandJSON struct {
    Command string `json:"command"`
    TS      int64  `json:"ts"`
}

type NodeRefJSON struct {
    ID    string `json:"id"`
    Title string `json:"title"`
}

type NodeJSON struct {
    ID          string `json:"id"`
    Title       string `json:"title"`
    Description string `json:"description"`
    Status      string `json:"status"`
    CreatedBy   string `json:"created_by"`
    BlobCount   int    `json:"blob_count"`
    CreatedAt   int64  `json:"created_at"`
}

type StatusJSON struct {
    Repository    string         `json:"repository"`
    InitializedAt int64          `json:"initialized_at"`
    BlobCount     int            `json:"blob_count"`
    BlobsByKind   map[string]int `json:"blobs_by_kind"`
    BlobsByTrust  map[int]int    `json:"blobs_by_trust"`
    NodeCount     int            `json:"node_count"`
    Nodes         []NodeJSON     `json:"nodes"`
    UnassignedBlobs int          `json:"unassigned_blobs"`
    PendingTasks  int            `json:"pending_tasks"`
    LLMAvailable  bool           `json:"llm_available"`
    LLMProvider   string         `json:"llm_provider"`
}
```

`lore log --json` outputs a JSON array of `BlobJSON`. All other `--json` outputs are single objects.

### Color and No-Color Detection

Add dependency: `github.com/mattn/go-isatty v0.0.20`.

```go
// internal/cli/color.go
func UseColor(cmd *cobra.Command) bool {
    if noColor, _ := cmd.Flags().GetBool("no-color"); noColor {
        return false
    }
    if os.Getenv("NO_COLOR") != "" {
        return false
    }
    if os.Getenv("TERM") == "dumb" {
        return false
    }
    return isatty.IsTerminal(os.Stdout.Fd())
}
```

ANSI constants:

```go
const (
    colorReset  = "\033[0m"
    colorGreen  = "\033[32m"
    colorYellow = "\033[33m"
    colorDim    = "\033[2m"
)
```

`--no-color` flag is a persistent flag on the root command.

### Phase 5 Test Scenarios

```go
func TestWhyQuery_ReturnsNewestFirst(t *testing.T)
func TestWhyQuery_SuffixMatch(t *testing.T)            // "oauth.go" matches "internal/auth/oauth.go"
func TestWhyQuery_NoResults_ErrorMessage(t *testing.T)
func TestTraceQuery_ReturnsOldestFirst(t *testing.T)
func TestStatusAggregation_AllCounts(t *testing.T)
func TestStatusAggregation_ZeroState(t *testing.T)     // fresh repo, no errors
func TestGraphRender_SubsystemsBeforeUnassigned(t *testing.T)
func TestGraphRender_MaxThreeBlobsPerNode(t *testing.T)
func TestGraphRender_UnassignedMax5(t *testing.T)
func TestJSONOutput_BlobLog_IsArray(t *testing.T)
func TestJSONOutput_BlobShow_IsObject(t *testing.T)
func TestColorDetection_NoColorEnv_DisablesColor(t *testing.T)
```

---

## Phase 6 — Agent Integration

**Goal:** Claude Code automatically populates Lore during AI sessions. Agent recaps produce Trust Level 2 Blobs.

### Design

Agents call `lore hook` directly. No HTTP server. No daemon.

### Claude Code Hook Integration

`lore init` writes to `.claude/settings.json`:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Edit",
        "command": "lore hook file-write \"$TOOL_INPUT_file_path\" agent:claude"
      },
      {
        "matcher": "Write",
        "command": "lore hook file-write \"$TOOL_INPUT_file_path\" agent:claude"
      },
      {
        "matcher": "Bash",
        "command": "lore hook command \"$TOOL_INPUT_command\" agent:claude"
      }
    ],
    "Stop": [
      {
        "command": "lore hook agent-recap agent:claude"
      }
    ]
  }
}
```

The `Stop` hook is the key addition over the previous design. It fires when Claude Code's session ends and emits an `AgentRecap` task with `trust_level=2`.

`lore hook agent-recap agent:claude` reads the session summary from Claude Code's environment and stores it as an `AgentRecap` task. Implementation depends on Claude Code's session summary API.

### Internal Hook Sub-Commands

```bash
lore hook commit                         # CommitCreated
lore hook checkout <prev> <new> <flag>   # BranchSwitch
lore hook merge                          # MergeEvent
lore hook file-write <path> <source>     # FileWrite
lore hook command <cmd> <source>         # Command
lore hook agent-recap <source>           # AgentRecap (reads recap from stdin/env)
```

### Exit Criteria

Run a Claude Code session that edits several files, then commits. `lore log` shows a Blob with `trust=AgentTruth` from `agent:claude`. `lore show <id>` displays the agent-provided summary and recap correctly in the `── Interpreted ──` section.

### Exact `.claude/settings.json` Content Written by `lore init`

`lore init` checks if `.claude/settings.json` already exists. If it does, it **merges** the hooks section rather than overwriting — preserving existing keys. Merging logic: read existing JSON, deep-merge hook arrays (no duplicates by matcher+command), write back with `json.MarshalIndent`.

If the file does not exist, create it with the full content shown above.

### `lore hook agent-recap` Reading Strategy

```go
func runHookAgentRecap(cmd *cobra.Command, args []string) error
// args[0] = source (e.g., "agent:claude")
```

Reading strategy (in order of preference):

1. Check `CLAUDE_STOP_HOOK_PAYLOAD` env variable — if set and non-empty, parse as JSON payload.
2. Check `CLAUDE_SESSION_SUMMARY` env variable — if set, use as `summary` field.
3. Read from stdin if stdin is not a TTY (`!isatty.IsTerminal(os.Stdin.Fd())`).
4. If none yield data: emit a minimal `AgentRecap` task with only `source` set. Log `lore: agent-recap received no summary data` to stderr. Still insert the task — it signals the session ended, enabling extraction.

### `AgentRecapPayload` Struct and Validation

```go
// internal/task/recap.go
type AgentRecapPayload struct {
    UserIntent string   `json:"user_intent"`
    Summary    string   `json:"summary"`
    Recap      string   `json:"recap"`
    Kind       string   `json:"kind"`
    Tags       []string `json:"tags"`
    Source     string   `json:"source"`
}

var validKinds = map[string]bool{
    "Feature": true, "BugFix": true, "Migration": true,
    "Investigation": true, "Refactor": true, "Architecture": true,
    "Review": true, "Incident": true,
}

func ValidateRecapPayload(p *AgentRecapPayload) {
    // Kind: if non-empty and not in validKinds, reset to ""
    // Tags: max 20 entries; each max 50 chars; empty entries removed
    // Summary: truncate to 1000 chars
    // UserIntent: truncate to 500 chars
    // Recap: truncate to 500 chars
    // Source: if empty, default to args[0] from caller
}
```

The `detail` column of the inserted task stores `json.Marshal(validated payload)`.

### Malformed Recap Handling

If `json.Unmarshal` fails on the raw input:
1. Log `lore: warning: agent-recap payload is not valid JSON — using raw text as summary` to stderr.
2. Construct `AgentRecapPayload{Summary: rawText, Source: args[0]}`.
3. Insert as `AgentRecap` task with `TrustLevel: 2`.
4. During `RecapIngester.IngestRecap`: if `UserIntent` and `Recap` are empty but `Summary` is set, use `Summary` for both the blob's `Summary` and `UserIntent`.

No panic, no abort. The extraction pipeline always runs.

### Internal Hook Sub-Commands (All Signatures)

```go
// internal/cli/hook.go — all hidden from --help
var hookCmd = &cobra.Command{Use: "hook", Hidden: true}

func runHookCommit(cmd *cobra.Command, args []string) error     // no args
func runHookCheckout(cmd *cobra.Command, args []string) error   // args: prevRef, newRef, flag
func runHookMerge(cmd *cobra.Command, args []string) error      // no args
func runHookFileWrite(cmd *cobra.Command, args []string) error  // args: path, source
func runHookCommand(cmd *cobra.Command, args []string) error    // args: command, source
func runHookAgentRecap(cmd *cobra.Command, args []string) error // args: source
```

Performance target: **< 50ms per invocation**. Hook commands must not perceptibly slow down git operations. All hooks: open store, insert task(s), optionally trigger extraction, close store, exit.

### Phase 6 Test Scenarios

```go
func TestHookAgentRecap_ValidJSON_InsertsTask(t *testing.T)
func TestHookAgentRecap_MalformedJSON_UsesRawText(t *testing.T)
func TestHookAgentRecap_EmptyInput_InsertsMinimalTask(t *testing.T)
func TestHookFileWrite_InsertsFileWriteTask(t *testing.T)
func TestHookCommand_InsertsCommandTask(t *testing.T)
func TestExtract_WithAgentRecap_SkipsOllama(t *testing.T)
func TestExtract_RecapBlobHasTrust2(t *testing.T)
func TestExtract_RecapBlobHasAgentSource(t *testing.T)
func TestSettingsJSON_WrittenByInit(t *testing.T)
func TestSettingsJSON_MergesExistingHooks(t *testing.T)
```

---

## Phase 7 — Distribution

**Goal:** Lore installable on a clean Mac and Linux via standard package managers.

### Deliverables

- `goreleaser` configuration: linux/mac/windows, arm64/amd64
- GitHub releases with changelog
- Homebrew formula: `brew install nishchay/tap/lore`
- `make install` → `/usr/local/bin/lore`
- `lore doctor`:

```
$ lore doctor

✓ Git repository found
✓ .lore/ initialized
✓ Git hooks installed (post-commit, post-checkout, post-merge)
✓ Claude Code hooks installed (PostToolUse, Stop)
✓ Ollama available (llama3)

Blobs: 15  (8 AgentTruth, 7 LoreInferred)
Nodes: 3
Last extraction: 2 hours ago
```

### Configuration

`.lore/config.toml` (per-repo), `~/.config/lore/config.toml` (global):

```toml
[llm]
provider = "ollama"
model    = "llama3"
endpoint = "http://localhost:11434"

[extraction]
min_tasks            = 1
task_retention_days  = 30

[node_resolution]
min_confidence       = 0.4    # minimum confidence to assign blob to node
tag_overlap_threshold = 0.5   # minimum tag overlap for node clustering

[output]
color = true
```

### Exit Criteria

`brew install nishchay/tap/lore && cd myrepo && lore init && lore doctor` all pass on a clean Mac. Binary under 15MB.

### `.goreleaser.yml` Structure (Exact)

```yaml
version: 2

before:
  hooks:
    - go mod tidy

builds:
  - id: lore
    binary: lore
    main: ./cmd/lore
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X github.com/nishchay/lore/internal/cli.Version={{.Version}}
      - -X github.com/nishchay/lore/internal/cli.Commit={{.ShortCommit}}
      - -X github.com/nishchay/lore/internal/cli.BuildDate={{.Date}}

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: "checksums.txt"

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "Merge pull request"

brews:
  - name: lore
    homepage: https://github.com/nishchay/lore
    description: "Local-first engineering memory system"
    repository:
      owner: nishchay
      name: homebrew-tap
    install: |
      bin.install "lore"
    test: |
      system "#{bin}/lore", "--version"
```

Binary size target: < 15MB. `CGO_ENABLED=0` and `-s -w` strip flags typically produce 8–12MB for this dependency set.

### Version Embedding (ldflags)

Three variables set by goreleaser:

```go
// internal/cli/version.go
var (
    Version   = "dev"
    Commit    = "none"
    BuildDate = "unknown"
)
```

`lore --version` output:

```
lore version dev (commit: none, built: unknown)
```

Release output:

```
lore version 0.1.0 (commit: abc1234, built: 2026-05-23T10:00:00Z)
```

### `lore doctor` Check List (Complete)

```go
// internal/cli/doctor.go
type DoctorCheck struct {
    Name    string
    Pass    bool
    Message string
    Warning bool // true = warn but don't fail
}
```

Checks in order:

1. **Git repository found** — `git.FindGitRoot()` succeeds. Required.
2. **.lore/ initialized** — `.lore/` directory and `.lore/lore.db` both exist. Required.
3. **Database schema current** — `meta.schema_version` == `"4"`. Required.
4. **Git hooks installed** — `.git/hooks/post-commit`, `post-checkout`, `post-merge` all exist, are executable, and contain `# Managed by lore`. Required.
5. **Claude Code hooks installed** — `.claude/settings.json` exists and contains `lore hook agent-recap` in the `Stop` hooks. Warning (not required).
6. **Ollama available** — `GET <endpoint>/api/tags` returns HTTP 200 within 3s. Warning (not required) — heuristic fallback is used if unavailable.

Summary lines printed after checks:

```
Blobs: <N>  (<AgentTruth count> AgentTruth, <LoreInferred count> LoreInferred)
Nodes: <N>
Last extraction: <human-readable time ago>  (or "never" if no blobs)
```

Exit code: 0 if all required checks pass. Warnings (Ollama, Claude Code hooks) do not affect exit code. Exit 1 if `.lore/` is not initialized.

### `make release` and `make snapshot` Targets

```makefile
release:
	goreleaser release --clean

snapshot:
	goreleaser release --snapshot --clean --skip=publish
```

Run `make snapshot` before tagging to verify binary sizes and cross-compilation without publishing.

### Homebrew Formula Reference

The `brews` section in `.goreleaser.yml` auto-generates the formula in `nishchay/homebrew-tap`. For documentation, the formula structure is:

```ruby
class Lore < Formula
  desc "Local-first engineering memory system"
  homepage "https://github.com/nishchay/lore"

  on_macos do
    on_arm do
      url "https://github.com/nishchay/lore/releases/download/v<VERSION>/lore_darwin_arm64.tar.gz"
      sha256 "<sha256>"
    end
    on_intel do
      url "https://github.com/nishchay/lore/releases/download/v<VERSION>/lore_darwin_amd64.tar.gz"
      sha256 "<sha256>"
    end
  end

  def install
    bin.install "lore"
  end

  test do
    system "#{bin}/lore", "--version"
  end
end
```

Goreleaser writes this file automatically. The manual version above is for reference only.

### Phase 7 Test Scenarios

```go
func TestDoctorChecks_AllPass(t *testing.T)           // integration: real temp git repo with full lore init
func TestDoctorChecks_MissingHooks_Fails(t *testing.T)
func TestDoctorChecks_NoLoreInit_Fails(t *testing.T)
func TestDoctorChecks_OllamaUnavailable_Warning(t *testing.T) // exit code still 0
func TestVersionString_ContainsVersion(t *testing.T)
func TestVersionString_DevDefault(t *testing.T)
```

Additional: `TestBinarySize` — build the binary via `exec.Command("go", "build", ...)`, check file size < 15,728,640 bytes (15MB).

---

## Technology Decisions

| Concern | Choice | Reason |
|---------|--------|--------|
| Language | Go | Single binary, cross-platform, strong stdlib |
| CLI | `github.com/spf13/cobra` | Standard for Go CLIs |
| Storage | SQLite via `modernc.org/sqlite` | Local-first, no CGO, zero deps |
| LLM | Ollama HTTP API | Local, model-agnostic, no data leaks |
| File watcher | Deferred (post-MVP) | Commit-boundary hooks are sufficient |
| TUI | Plain ASCII MVP, `bubbletea` post-MVP | Don't over-invest in visualization |
| Releases | `goreleaser` | Multi-platform binary distribution |

---

## Non-Goals for MVP

| Excluded | Why |
|----------|-----|
| `lore watch` daemon | Automatic via Git hooks |
| `lore session` commands | Internal state; not user-visible |
| `lore tasks` CLI | Tasks are internal plumbing |
| `lore serve` HTTP server | Lore is a CLI tool, not a service |
| `prepare-commit-msg` hook | Inverts Git→Lore dependency |
| Automatic Node generation | Deferred post-MVP; user creates Nodes in MVP |
| `lore node suggest` | Post-MVP; requires human confirmation to persist |
| Semantic commit splitting | Repository author responsible for commit hygiene |
| File read tracking (fsnotify) | Too noisy; deferred |
| Multi-repo federation | Single repo first |
| Cloud sync | Local-first only |
| Custom graph database | SQLite is sufficient |
| Git replacement | Never |
| Atlas as dependency | Atlas consumes Lore; Lore is standalone |

---

## Cross-Cutting Implementation Notes

### Package Import Graph (No Circular Dependencies)

Allowed import directions:

```
cmd/lore
  └── internal/cli

internal/cli
  ├── internal/store
  ├── internal/config
  ├── internal/git
  ├── internal/blob
  ├── internal/node
  ├── internal/graph
  └── internal/task

internal/blob
  ├── internal/task    (type definitions only)
  └── internal/store   (read tasks, write blobs)

internal/graph
  ├── internal/store
  ├── internal/blob    (type definitions only)
  └── internal/node    (type definitions only)

internal/node
  └── internal/store

internal/store
  ├── internal/task    (type definitions only)
  ├── internal/blob    (type definitions only)
  └── internal/node    (type definitions only)

internal/git      → (no internal deps)
internal/config   → (no internal deps)
internal/task     → (no internal deps)
```

`internal/store` imports `internal/task`, `internal/blob`, `internal/node` only for type definitions. If this creates circular import issues, introduce `internal/types` for shared structs and have all packages import from there.

### Transaction Pattern for Multi-Insert Operations

`InsertBlobWithRelations` is the **only** path for blob insertion. It runs all writes in one transaction:

```go
func (s *Store) InsertBlobWithRelations(
    ctx context.Context,
    b blob.Blob,
    files []BlobFile,
    commands []BlobCommand,
    taskIDs []string,
) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil { return err }
    defer tx.Rollback()
    // INSERT INTO blobs ...
    // INSERT INTO blob_files ... (one per file)
    // INSERT INTO blob_commands ... (one per command)
    // INSERT INTO blob_tasks ... (one per taskID)
    // UPDATE tasks SET extracted=1, extracted_into=blobID WHERE id IN (...)
    return tx.Commit()
}
```

Never call `InsertBlob` followed by separate file/command inserts outside a transaction. Partial inserts corrupt the system.

### ID Prefix Resolution

```go
// internal/store/store.go
func (s *Store) ResolveBlobIDPrefix(ctx context.Context, prefix string) (string, error) {
    if len(prefix) < 7 {
        return "", fmt.Errorf("blob ID prefix too short (minimum 7 characters)")
    }
    rows, err := s.db.QueryContext(ctx, "SELECT id FROM blobs WHERE id LIKE ? LIMIT 2", prefix+"%")
    if err != nil { return "", err }
    defer rows.Close()
    var ids []string
    for rows.Next() {
        var id string
        rows.Scan(&id)
        ids = append(ids, id)
    }
    switch len(ids) {
    case 0: return "", ErrNotFound
    case 1: return ids[0], nil
    default: return "", ErrAmbiguous
    }
}
```

Used by `lore show` and `lore assign`. Error messages:

```
error: no blob found for 'abc123f'
error: ambiguous blob ID prefix 'abc123f' — use more characters
```

### `lore status` Zero-State Guarantee

A freshly initialized repo with zero commits must satisfy all of these:
- `lore status` exits 0
- All counts are zero (blobs, nodes, tasks)
- No SQL query errors on empty tables
- "Last extraction: never" (not an error)
- LLM status shows "not checked" (lore doctor runs the actual check)

This invariant must be preserved through every phase. Test it in Phase 1 and re-run the test in every subsequent phase.

### Error Message Conventions

Error messages follow Git conventions:

```
error: <message>
hint: <suggestion>
```

Examples:
```
error: not a lore repository (or any parent up to mount point /)
hint: run 'lore init' to initialize

error: no blobs found for 'auth.go'
hint: run 'lore status' to see what lore has captured

error: node 'Payments' already exists
hint: use 'lore node list' to see all nodes

error: ambiguous blob ID prefix 'abc123f' — use more characters
```

Errors go to stderr. Output goes to stdout. Exit codes: 0=success, 1=general error, 2=usage error, 128=not a lore repository.

---

## Migration Notes (from previous architecture)

| Previous Term | New Term | Notes |
|---------------|----------|-------|
| Event | Task | Renamed. Same concept, `trust_level` column added. |
| Knowledge Object / KnowledgeNode | Blob | Renamed. Added: `summary`, `recap`, `user_intent`, `inferred_reasoning`, `trust_level`, `ai_source`. |
| (no equivalent) | Node | New tier. Long-lived topic grouping Blobs. |
| Knowledge Graph (Tier 3) | Knowledge Graph (Tier 4) | Same concept, now includes Topic nodes for Nodes. |
| `internal/event/` | `internal/task/` | Package renamed. |
| `internal/knowledge/` | `internal/blob/` | Package renamed. `internal/node/` added. |
| `knowledge` table | `blobs` table | Schema renamed + extended. |
| `events` table | `tasks` table | Schema renamed + `trust_level` added. |
| `knowledge_files` | `blob_files` | Renamed. |
| `knowledge_commands` | `blob_commands` | Renamed. |
| (no equivalent) | `blob_tasks`, `nodes`, `node_blobs` | New tables. |
