# Rule: CLI Design

> Parent: [../ARCHITECTURE.md](../ARCHITECTURE.md)

---

## The Governing Principle

Lore's CLI must feel like Git.

Git is the model because:
- It does not require a running daemon to use
- It works via a single binary that reads/writes a local directory
- Commands operate on a repository, not a service
- Observation (tracking) is automatic after `git init`
- Users interact with the output of Git's work (log, diff, blame), not the internals (objects, refs)

Lore applies the same principles. Users interact with **Blobs** and **Nodes**, not with Tasks, extraction windows, or graph edges.

---

## Automatic Observation

After `lore init`:
- Git hooks are installed automatically
- Repository activity is captured as Tasks without any additional commands
- No background daemon is required for MVP
- No `lore watch` command exists in MVP

---

## No Manual Session Lifecycle

The following patterns are **forbidden** in MVP:

```bash
lore start              ← forbidden
lore stop               ← forbidden
lore session start      ← forbidden
lore session end        ← forbidden
lore session close      ← forbidden
```

Blob extraction windows are internal state. Users never start or stop them. Extraction happens automatically on commit.

If a user needs to annotate what they are working on:

```bash
lore record "investigating JWT expiry bug"
```

This emits a `Note` task, included in the next Blob extraction.

---

## No Raw Task Exposure

The following are **forbidden** in MVP:

```bash
lore tasks              ← forbidden
lore tasks --last 20    ← forbidden
lore events             ← forbidden (old name — also forbidden)
```

Tasks are internal telemetry. Users interact with Blobs.

---

## No HTTP Server in MVP

```bash
lore serve              ← forbidden
lore server start       ← forbidden
```

Lore is a CLI tool. Agent integration in MVP is via direct hook calls.

---

## MVP Command Set

```bash
lore init                       # initialize .lore/ and install git hooks
lore status                     # show what lore knows about this repository
lore log                        # list blobs newest-first (like git log)
lore show <id>                  # full detail for one blob
lore why <file>                 # why does this file exist?
lore trace <file>               # chronological history of a file
lore graph                      # ASCII graph (subsystems → blobs → files)
lore record <note>              # emit a Note task (developer annotation)
lore doctor                     # check prerequisites and diagnose issues

# Node management (subsystem lifecycle)
lore node create <name>         # create a new subsystem node
lore node list                  # list all subsystem nodes
lore node show <name>           # show blobs assigned to a subsystem

# Blob assignment
lore assign <blob-id> <node>    # assign a blob to a subsystem (HumanAssertion, trust=3)
```

Internal command (not user-facing, not in `--help` top level):

```bash
lore hook <kind> [args]        # called by git hooks only
lore hook agent-recap <source> # called by agent Stop hook only
```

---

## `lore log` Output

Lists Blobs newest-first, like `git log`.

```
$ lore log

abc1234  OAuth Provider Impl   Feature     2026-05-20  trust=AgentTruth   3 files
def5678  Fix JWT expiry        BugFix      2026-05-18  trust=LoreInferred  2 files
ghi9012  Auth middleware       Refactor    2026-05-15  trust=AgentTruth   5 files
```

Optional flags (MVP):
- `--limit N` — show N most recent blobs (default: 20)
- `--json` — machine-readable JSON output

---

## `lore show <id>` Output

Full detail for one Blob, with observed and inferred sections clearly separated.

```
$ lore show abc1234

ID:           abc1234
Title:        OAuth Provider Implementation
Kind:         Feature
Trust:        AgentTruth (source: agent:claude)

── Observed ────────────────────────────────────────
Started:      2026-05-20 09:14
Ended:        2026-05-20 16:42
Commits:      abc100..abc123

Files Modified:
  internal/auth/oauth.go
  internal/session/manager.go

Files Deleted:
  internal/auth/token_legacy.go

Commands:
  go test ./internal/auth/...
  go test ./...

── Interpreted ─────────────────────────────────────
User Intent:  Add Google OAuth support to replace legacy token auth
Summary:      Implemented OAuth2 provider flow and callback handling.
              Session integration updated to use provider tokens.
Recap:        Authentication subsystem migrated to provider-based login.
              This eliminates maintenance of the legacy token system.
Tags:         auth, oauth, session, provider

── Part of ─────────────────────────────────────────
Node: OAuth Migration (confidence: 0.95)
```

The `── Observed ──` and `── Interpreted ──` sections must always be visually distinct.

---

## `lore why <file>` Output

```
$ lore why internal/auth/oauth.go

OAuth Provider Implementation (Feature)  2026-05-20  [AgentTruth]
  Replaced legacy token auth with OAuth2 provider flow.
  Commits: abc100..abc123
  Node: OAuth Migration

Fix JWT expiry (BugFix)  2026-05-18  [LoreInferred]
  Corrected token expiry calculation.
  Commits: def789
```

---

## `lore status` Output

```
$ lore status

Repository: /Users/nishchay/projects/myapp
Initialized: 2026-05-10

Blobs: 12
  Feature:       4   (3 AgentTruth, 1 LoreInferred)
  BugFix:        3   (1 AgentTruth, 2 LoreInferred)
  Refactor:      2   (2 AgentTruth)
  Migration:     1   (1 AgentTruth)
  Investigation: 2   (0 AgentTruth, 2 LoreInferred)

Subsystems (Nodes): 3
  Authentication   (8 blobs, active)
  Billing          (3 blobs, active)
  Session Mgmt     (1 blob,  active)

Unassigned Blobs: 2
  hint: use 'lore assign <id> <subsystem>' or 'lore node create <name>'

Pending Tasks: 3 (next extraction on next commit)

LLM: ollama/llama3 (available)
```

---

## Post-MVP Command Candidates

These are explicitly deferred:

```bash
lore explain <concept>     # full-text search across blobs (deferred)
lore history <node>        # temporal query for a subsystem (deferred)
lore rebuild-graph         # graph reconstruction from blobs+nodes (deferred)
lore re-extract <id>       # re-run AI extraction on a blob (deferred)
lore stats                 # storage statistics (deferred)
lore node suggest          # Lore suggests Node assignments for unassigned blobs (deferred)
```

Note: `lore node suggest` is the post-MVP entry point for automatic Node inference. It will propose assignments but require explicit `lore assign` confirmation to persist them at trust_level=3.

---

## Output Formatting

**Default:** Human-readable, ANSI color on TTY, auto-disabled when piped.

**`--json`:** All query commands support `--json` for machine-readable output.

**`--no-color`:** Also disabled automatically when `NO_COLOR` env var is set.

**Trust level display:**
- `[AgentTruth]` — displayed in high-trust color (green or default)
- `[LoreInferred]` — displayed in inference color (yellow or dim)

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (see stderr) |
| 2 | Usage error (bad arguments) |
| 128 | Not a lore repository |

---

## Error Messages

```
error: not a lore repository (or any parent up to mount point /)
hint: run 'lore init' to initialize

error: no blobs found for 'auth.go'
hint: run 'lore status' to see what lore has captured
```

Errors go to stderr. Output goes to stdout.
