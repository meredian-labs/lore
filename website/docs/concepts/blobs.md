---
title: Blobs
---

# Blobs

A Blob is the primary artifact of Lore. It is a compressed, human-readable record of one unit of engineering work — a feature implementation, a bug fix, a refactor, an investigation — along with everything Lore observed while that work was happening.

**Blobs are permanent. They are never automatically deleted.**

## The Core Principle: Observed vs Interpreted

Every Blob contains two categories of data that are always stored separately and always displayed separately:

| Category | Trust Level | What It Contains | How It Is Populated |
|----------|-------------|-----------------|-------------------|
| **Observed** | 1 — GroundTruth | File paths, commit SHAs, timestamps, commands | Lore reads directly from Git hooks and task records |
| **Interpreted** | 2, 3, or 4 | Title, summary, user intent, kind, tags | AI agent recap, or Lore's local inference |

Observed fields are deterministic. Given the same sequence of Tasks, Lore will always produce the same observed fields. Interpretation may vary across AI models or runs — which is why the trust level is recorded.

AI-generated content is never written into observed-fact columns. Observed facts are never transformed by AI before storage.

## Blob Kinds

| Kind | Description | Example |
|------|-------------|---------|
| `Feature` | New capability added to the system | OAuth provider implementation |
| `BugFix` | Correction of a defect or regression | JWT expiry off-by-one fix |
| `Migration` | Data, schema, or system migration | User table Postgres migration |
| `Investigation` | Exploration without an immediate code change | Profiling memory allocation in auth middleware |
| `Refactor` | Structural improvement without behavior change | Auth middleware layer extraction |
| `Architecture` | Design decision or structural proposal | ADR: adopt service mesh for inter-service auth |
| `Review` | Code review session | Review: billing service PR #142 |
| `Incident` | Response to a production issue | Oncall: auth service 500s on OAuth callback |
| `Checkpoint` | Periodic snapshot of ongoing work | Daily checkpoint: auth migration day 3 |

Kind is inferred by AI or heuristic rules. It can be corrected post-hoc via `lore re-extract` (post-MVP).

## Trust Levels

Every Blob records the provenance of its interpreted fields:

| Trust Level | Label | `ai_source` values | Meaning |
|-------------|-------|-------------------|---------|
| 1 | GroundTruth | — | Observed fields only; applies to task data, not blob interpretation |
| 2 | AgentTruth | `agent:claude`, `agent:cursor` | An AI agent that did the work provided the recap directly |
| 3 | HumanAssertion | — | A human explicitly asserted this (applies to Node assignments, not blob fields) |
| 4 | LoreInference | `lore:ollama`, `lore:heuristic`, `lore:system` | Lore inferred the interpretation from observed tasks |

**AgentTruth (2) is always preferred over LoreInference (4).** The agent that performed the work has the most context about what was attempted and why. When an `AgentRecap` Task exists in the extraction window, Lore uses it directly and skips local AI inference entirely.

**LoreInference quality** depends on what is available:
- `lore:ollama` — Ollama is running; Lore submitted a structured prompt and parsed the JSON response
- `lore:heuristic` — Ollama is unavailable; Lore used commit message keywords and directory names to classify the work
- `lore:system` — Lore used internal rule-based patterns

## The Full Blob Structure

```
$ lore show c91a447

ID:           c91a447
Title:        Auth middleware refactor
Kind:         Refactor
Trust:        AgentTruth (source: agent:claude)

── Observed ────────────────────────────────────────
Started:      2026-05-15 10:22
Ended:        2026-05-15 14:55
Commits:      fa3e8..c91a44

Files Modified:
  internal/auth/middleware.go
  internal/auth/middleware_test.go
  internal/session/manager.go
  internal/session/manager_test.go
  cmd/lore/main.go

Files Deleted:
  internal/auth/legacy_token.go

Commands:
  go test ./internal/auth/...
  go test ./...
  go vet ./...

── Interpreted ─────────────────────────────────────
User Intent:  Extract auth handling from main handler into a discrete middleware layer
Summary:      Moved authentication logic out of the top-level HTTP handler into a
              dedicated middleware. Session manager updated to use the new interface.
              Legacy token validation removed; all paths now go through OAuth flow.
Recap:        Authentication subsystem now has a clean boundary. This unblocks the
              planned rate-limiting work, which can now hook into the middleware chain
              without touching business logic.
Tags:         auth, middleware, session, refactor

── Part of ─────────────────────────────────────────
Node: Authentication (confidence: 0.97)
```

The `── Observed ──` and `── Interpreted ──` sections are always visually distinct. You can trust everything in Observed absolutely. Interpreted is AI-sourced and carries its provenance label.

## Blob Lifecycle

1. **Tasks accumulate** as you work (file writes, commands, commits)
2. **Extraction triggers** on `CommitCreated` (the `post-commit` hook)
3. **Observed fields are populated** deterministically from task records and Git
4. **Interpreted fields are populated** from `AgentRecap` (preferred) or local AI inference
5. **Blob is written** to `.lore/lore.db` permanently
6. **Graph is updated** to reflect the new Blob's file and commit relationships
7. **Tasks are marked absorbed** and will be purged after the retention period

A valid Blob must have at least one associated commit or file write. Blobs are independent of Tasks after extraction — Tasks can be purged without affecting Blobs.

## Querying Blobs

```bash
lore log                    # list all blobs, newest first
lore log --limit 5          # show 5 most recent
lore log --json             # machine-readable output
lore show <id>              # full detail for one blob
lore why <file>             # all blobs that touched a file
lore trace <file>           # same, chronological order (oldest first)
```
