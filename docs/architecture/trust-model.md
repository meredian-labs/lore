---
layout: default
title: Trust Model
parent: Architecture
nav_order: 3
---
# Trust Model

Lore tracks the confidence and provenance of all interpreted information. Every blob carries a trust level that tells you how much to rely on its interpreted fields.

## Why Trust Levels Exist

Lore captures both **observed facts** (what happened) and **interpreted meaning** (why it happened). Observed facts are deterministic. Interpreted meaning varies in reliability depending on its source.

Trust levels make this provenance explicit and permanent. You always know whether a blob's title and summary came from the agent that did the work, from Lore's AI, or from a commit message heuristic.

## Trust Level Table

| Level | Name | Source | When It's Set |
|-------|------|--------|---------------|
| 1 | **GroundTruth** | Git and task observation | File paths, commit SHAs, timestamps, commands. Always set by Lore from observed facts. Cannot be wrong. |
| 2 | **AgentTruth** | AI agent session recap | An AI agent (Claude Code, Cursor, OpenHands) emitted an `AgentRecap` task in the extraction window. Lore uses the recap directly without invoking its own AI. |
| 3 | **HumanAssertion** | Explicit user command | User ran `lore assign` or `lore node create`. Applies to `primary_node_id` and node names, not to blob interpretation fields. |
| 4 | **LoreInference** | Lore's local AI or heuristics | No agent recap was available. Lore used Ollama or commit message heuristics to infer title, summary, kind, and tags. |

## What Each Level Covers

Trust levels 1 and 3 apply to specific fields. Trust levels 2 and 4 apply to the interpreted section of a blob.

| Field | Trust Level | Notes |
|-------|-------------|-------|
| `started_at`, `ended_at` | 1 — GroundTruth | Always from task timestamps |
| `commit_start`, `commit_end` | 1 — GroundTruth | Always from Git via hook |
| `blob_files` (paths) | 1 — GroundTruth | Always from task records |
| `blob_commands` | 1 — GroundTruth | Always from task records |
| `title`, `summary`, `recap`, `user_intent`, `kind`, `tags` | 2 — AgentTruth | When `AgentRecap` task exists |
| `title`, `summary`, `recap`, `user_intent`, `kind`, `tags` | 4 — LoreInference | When no `AgentRecap` task exists |
| `primary_node_id` | 3 — HumanAssertion | Set by `lore assign` |
| `inferred_reasoning` | 4 — LoreInference | Set only by Lore's local AI; always NULL for AgentTruth blobs |

## Why Agent Recaps Beat Lore Inference

The agent that performed the work has more context than an observer analyzing its actions:

- It knows what it was trying to accomplish
- It knows what worked and what had to be abandoned
- It can describe the reasoning behind architecture decisions
- It has access to the full conversation context

Lore's fallback inference can only observe file paths, command strings, and commit messages. It will always have less signal than the actor.

When an `AgentRecap` task exists in the extraction window, Lore's local AI is never invoked for that blob. The agent's summary is used directly at `trust_level=2`.

## Heuristic Fallback (trust_level=4, ai_source=lore:heuristic)

When no AI model is available (Ollama not running), Lore falls back to rule-based extraction:

| Signal | Inference |
|--------|-----------|
| Commit message starts with "fix" or "bug" | kind=BugFix |
| Commit message starts with "feat" or "add" | kind=Feature |
| Commit message contains "migrate" or "migration" | kind=Migration |
| Commit message starts with "refactor" or "chore" | kind=Refactor |
| No commit in window, mostly reads | kind=Investigation |
| File paths contain "arch", "design", "adr" | kind=Architecture |
| Default (no signal) | kind=Feature |

Heuristic blobs still contain correct observed fields. The interpretation is weaker, but the blob is always valid and queryable.

## How Trust Affects Display

### lore log

Trust level appears as a label on each line:

```
abc1234  OAuth Provider Impl   Feature  2026-05-20  [AgentTruth]    3 files
def5678  Fix JWT expiry        BugFix   2026-05-18  [LoreInferred]  2 files
```

On a TTY, labels are color-coded:
- `[AgentTruth]` — default or green
- `[LoreInferred]` — yellow or dim

### lore show

Trust level appears in the header and drives the visual separation:

```
Trust:   AgentTruth (source: agent:claude)

── Observed ──────────────────────────────────────
...

── Interpreted ───────────────────────────────────
...
```

When trust is `LoreInferred`, an additional `Inferred Reasoning` field may appear under `── Interpreted ──` showing Lore's reasoning process.

### lore why / lore trace

Trust label appears inline with each result:

```
OAuth Provider Implementation  (Feature)  2026-05-20  [AgentTruth]
Fix JWT expiry                 (BugFix)   2026-05-18  [LoreInferred]
```

## AI Source Values

The `ai_source` column records the specific system that generated the interpreted fields:

| ai_source | trust_level | Meaning |
|-----------|-------------|---------|
| `agent:claude` | 2 | Claude Code provided the session recap |
| `agent:cursor` | 2 | Cursor provided the session recap |
| `agent:openhands` | 2 | OpenHands provided the session recap |
| `lore:ollama` | 4 | Lore used Ollama for local inference |
| `lore:heuristic` | 4 | Lore used commit message heuristics (no AI) |

## Provenance Is Permanent

Trust level and `ai_source` are set when a blob is created and never changed. Even if a higher-trust source becomes available later (e.g., you configure Ollama after the fact), the original blob retains its original provenance. Re-extraction is a post-MVP feature (`lore re-extract <id>`).

## See Also

- [Storage Architecture](./storage) — observed vs. inferred storage rule
- [Schema](./schema) — `trust_level` and `ai_source` column definitions
- [AI integration concepts](/integrations/custom-agent) — internal AI model responsibility rules
