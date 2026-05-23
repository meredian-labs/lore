# Rule: AI Model Responsibilities

> Parent: [../ARCHITECTURE.md](../ARCHITECTURE.md)
> Storage context: [STORAGE.md](./STORAGE.md)

---

## The Core Rule

The AI has two distinct roles in Lore. They operate at different trust levels and must never be confused with each other.

**Role A — Agent-Provided Recaps (Trust Level 2):** The AI agent that did the work provides a structured summary. This is trusted metadata — higher than human assertions about interpretation, equal to direct agent observation.

**Role B — Lore Fallback Inference (Trust Level 4):** Lore's local AI analyzes observed Tasks and generates an interpretation when no higher-trust source exists. This is the lowest trust level.

The full trust hierarchy:
1. Ground Truth — what Lore observed directly
2. Agent Truth — what the agent reported about its own work
3. Human Assertion — what the user explicitly asserted (e.g. `lore assign`, `lore node create`)
4. Lore Inference — what Lore's AI inferred from tasks

In both AI roles, the AI **never touches** observed facts. Files, commits, timestamps, and commands are always determined by Lore from Task records.

---

## Role A — Agent-Provided Recaps

When an AI agent (Claude, Cursor, OpenHands) completes a unit of work, it may emit a structured `AgentRecap` task:

```json
{
  "user_intent": "Add Google OAuth support to replace legacy token auth",
  "summary": "Implemented OAuth2 provider flow, callback handler, and session integration.",
  "recap": "Authentication subsystem migrated toward provider-based login. This eliminates the legacy token system maintenance burden.",
  "kind": "Feature",
  "tags": ["auth", "oauth", "session", "provider"]
}
```

This recap is stored as a `kind=AgentRecap` task with `trust_level=2` (AgentTruth).

During Blob extraction, if an `AgentRecap` task exists in the window:
- Its `user_intent`, `summary`, `recap`, `kind`, and `tags` are ingested directly into the Blob.
- The resulting Blob gets `trust_level=2` and `ai_source="agent:claude"` (or whichever agent).
- Lore's local AI is **not invoked** for this Blob.

### Why Agent Recaps Have Higher Trust

The agent that performed the work has the most context:
- It knows what it was trying to accomplish
- It knows what worked and what didn't
- It can describe the reasoning behind its choices

Lore's fallback inference, by contrast, can only observe actions and guess intent. It will always have less context than the actor.

### Agent Recap Ingestion (Claude Code)

`lore init` installs a `Stop` hook in `.claude/settings.json`:

```json
{
  "hooks": {
    "Stop": [
      {
        "command": "lore hook agent-recap agent:claude"
      }
    ]
  }
}
```

The `lore hook agent-recap` command reads Claude Code's session summary (via environment variables or stdin) and emits an `AgentRecap` task. Implementation details depend on the Claude Code API for session summaries.

---

## Role B — Lore Fallback Inference

When no `AgentRecap` task exists in the extraction window (human coding session, agent without recap support, incomplete data), Lore's local AI model analyzes the observed Tasks and generates interpreted fields.

### What Lore AI Generates

| Field | Description |
|-------|-------------|
| `title` | Short, human-readable name for the work |
| `summary` | 2-5 sentences: what was done |
| `recap` | 1-3 sentences: why it matters in the bigger picture |
| `user_intent` | Best guess at what the developer was trying to accomplish |
| `inferred_reasoning` | Lore's reasoning about the work (stored separately from summary) |
| `kind` | Classification: Feature, BugFix, Migration, etc. |
| `tags` | Domain concepts |

The resulting Blob gets `trust_level=4` (LoreInference) and `ai_source="lore:ollama"` or `"lore:heuristic"`.

Note: Lore AI does **not** set `primary_node_id` directly. Node assignment via inference is suggested separately and requires user confirmation or explicit `lore assign` command to persist with trust_level=3. Lore may suggest a Node (trust=4) but the assignment is not written to `blobs.primary_node_id` without human confirmation.

### What Lore AI Does NOT Generate

| Prohibited | Correct Owner |
|-----------|---------------|
| File lists (`blob_files`) | Lore reads task records |
| Command lists (`blob_commands`) | Lore reads task records |
| Commit SHAs (`commit_start`, `commit_end`) | Lore reads Git via hooks |
| Timestamps (`started_at`, `ended_at`) | Lore reads task timestamps |
| Source attribution (`source` on tasks) | Lore reads task metadata |
| Graph edges | Graph builder derives from Blobs |
| Node assignments | Node resolver (heuristic, separate step) |

---

## AI Must Not Define Repository Truth

If the AI model is unavailable, the system must continue working correctly.

- All observed fields are populated before AI is called.
- Heuristic extraction must produce valid Blobs when no AI is available.
- Blobs created without AI must be queryable via `lore why`, `lore trace`, `lore log`.

A Blob with AI-generated title is better. A Blob without one is still valid.

---

## Heuristic Fallback (No Ollama)

When no AI model is available, Lore falls back to rule-based extraction.

**Kind inference:**

| Signal | Inference |
|--------|-----------|
| Commit message starts with "fix" or "bug" | kind=BugFix |
| Commit message starts with "feat" or "add" | kind=Feature |
| Commit message contains "migrate" or "migration" | kind=Migration |
| Commit message starts with "refactor" or "chore" | kind=Refactor |
| No commit in window, mostly reads | kind=Investigation |
| File paths contain "arch", "design", "adr" | kind=Architecture |
| Default (no signal) | kind=Feature |

**Title:** First commit message in the window, truncated to 72 characters.

**Summary:** "Modified N files. Ran M commands. Produced K commit(s)."

**Tags:** Directory names of modified files.

Blobs created by heuristic get `ai_source="lore:heuristic"` and `trust_level=4`.

---

## AI Integration Pipeline

```
PendingTasks (from store)
    │
    ▼
WindowBuilder
    │  — fills deterministic fields:
    │    started_at, ended_at, files_written,
    │    commands, commit_start, commit_end
    │
    ▼
RecapLookup
    │  — checks if AgentRecap task exists in window
    │
    ├── YES → RecapIngester
    │           — extracts: user_intent, summary, recap, kind, tags
    │           — sets: trust_level=2, ai_source=agent:<name>
    │           — skips LLMClient entirely
    │
    └── NO  → PromptBuilder
                │  — assembles structured text from observed fields
                ▼
              LLMClient  (or HeuristicExtractor if unavailable)
                │
                ▼
              ResponseParser
                │  — extracts: title, summary, user_intent, kind, tags
                │  — generates: inferred_reasoning
                │  — sets: trust_level=3, ai_source=lore:ollama|lore:heuristic
                ▼
    Blob (complete)
    │
    ▼
Store.InsertBlob()
    │
    ▼
GraphBuilder.UpdateFromBlob()
```

---

## Prompt Contract (Lore Fallback Only)

When invoking Ollama, the prompt must request structured JSON.

Required output schema:

```json
{
  "title": "string (max 100 chars)",
  "summary": "string (max 500 chars)",
  "recap": "string (max 300 chars)",
  "user_intent": "string (max 200 chars)",
  "kind": "Feature | BugFix | Migration | Investigation | Refactor | Architecture | Review | Incident",
  "tags": ["string"]
}
```

The prompt must include as context:
- List of files written and deleted (from `blob_files`)
- List of commands executed (from `blob_commands`)
- Commit messages in the window
- Time range (human-readable)
- Source attribution (which sources: human, agent, ci)

The prompt must NOT include:
- Raw file contents or diffs
- Agent conversation history
- Anything that would send sensitive repository data to a remote model

---

## Local AI Requirement

For MVP, Lore uses Ollama exclusively. No data leaves the machine via the AI integration.

If a user configures a remote API (future), they opt in explicitly. Remote AI must never be the default.

---

## Distinguishing AI Outputs

The `ai_source` and `trust_level` columns on `blobs` allow users and tools to distinguish:

| `ai_source` | `trust_level` | Meaning |
|-------------|---------------|---------|
| `agent:claude` | 2 | Claude provided the recap — highest AI quality |
| `agent:cursor` | 2 | Cursor provided the recap |
| `agent:openhands` | 2 | OpenHands provided the recap |
| `lore:ollama` | 4 | Lore inferred from tasks using Ollama |
| `lore:heuristic` | 4 | Lore inferred using commit message heuristics |

Note: Trust level 3 (HumanAssertion) applies to `primary_node_id` and Node creation, not to `ai_source`. Those fields are set by user commands, not by AI.

Post-MVP: `lore log --agent-only`, `lore log --inferred-only`, `lore re-extract <id>` (re-run inference on a Blob).
