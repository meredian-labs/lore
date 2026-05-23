---
sidebar_position: 1
---

# Claude Code Integration

Lore integrates with Claude Code at three levels: Git hooks that capture every file edit and command, a `Stop` hook that submits a structured session recap at the end of each agent turn, and an MCP server that lets the agent query Lore directly during a session.

After `lore init`, all of this is configured automatically.

---

## What `lore init` Writes

`lore init` creates or updates `.claude/settings.json` in your repository root. A complete install looks like this:

```json
{
  "mcpServers": {
    "lore": {
      "command": "lore",
      "args": ["mcp", "agent:claude"]
    }
  },
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Edit",
        "hooks": [
          {
            "type": "command",
            "command": "lore hook file-write \"$CLAUDE_TOOL_INPUT_FILE_PATH\" agent:claude"
          }
        ]
      },
      {
        "matcher": "Write",
        "hooks": [
          {
            "type": "command",
            "command": "lore hook file-write \"$CLAUDE_TOOL_INPUT_FILE_PATH\" agent:claude"
          }
        ]
      },
      {
        "matcher": "Read",
        "hooks": [
          {
            "type": "command",
            "command": "lore hook file-read \"$CLAUDE_TOOL_INPUT_FILE_PATH\" agent:claude"
          }
        ]
      },
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "lore hook command \"$CLAUDE_TOOL_INPUT_COMMAND\" agent:claude"
          }
        ]
      }
    ],
    "Stop": [
      {
        "type": "command",
        "command": "lore hook agent-recap agent:claude"
      }
    ]
  }
}
```

If `.claude/settings.json` already exists, `lore init` merges the `hooks` and `mcpServers` entries into the existing file rather than overwriting it.

---

## What Each Hook Does

### PostToolUse — Edit and Write

Fires after every `Edit` or `Write` tool call. Records the modified file path as a `FileWrite` task with `trust_level=2` (AgentTruth) and `source=agent:claude`.

```
lore hook file-write <path> agent:claude
```

These file write tasks become the `blob_files` observed fields on the next Blob. The AI never infers which files were changed — Lore reads them directly from these task records.

### PostToolUse — Read

Fires after every `Read` tool call. Records the file path as a `FileRead` task.

```
lore hook file-read <path> agent:claude
```

File reads are lower-signal than writes. They inform the `Investigation` kind heuristic (a session with many reads and no commits) but are not shown in `lore show` output by default.

### PostToolUse — Bash

Fires after every `Bash` tool call. Records the command string as a `Command` task.

```
lore hook command "<command>" agent:claude
```

Commands appear in the `── Observed ──` section of `lore show` output:

```
Commands:
  go test ./internal/auth/...
  go build ./...
  git diff HEAD~1
```

### Stop — Agent Recap

Fires at the end of each Claude Code agent turn (when the agent stops and returns control to the user). This is the most important hook.

```
lore hook agent-recap agent:claude
```

The `Stop` hook reads a structured JSON summary from the Claude Code session and emits an `AgentRecap` task with `trust_level=2`. When Lore extracts the next Blob, it finds this recap and uses it directly — skipping the local Ollama fallback entirely.

The recap JSON has this shape:

```json
{
  "user_intent": "Add Google OAuth support to replace legacy token auth",
  "summary": "Implemented OAuth2 provider flow, callback handler, and session integration.",
  "recap": "Authentication subsystem migrated toward provider-based login. This eliminates the legacy token system maintenance burden.",
  "kind": "Feature",
  "tags": ["auth", "oauth", "session", "provider"]
}
```

The resulting Blob will show `trust=AgentTruth` in `lore log` output, indicating the interpretation came from the agent that did the work rather than from Lore's inference.

---

## MCP Server

`lore init` registers `lore mcp agent:claude` as an MCP server named `lore`. This starts a JSON-RPC 2.0 server over stdio that Claude Code connects to automatically.

The MCP server gives the agent read/write access to Lore's knowledge graph during a session — without shell escaping, without subprocess management, and with structured return types.

### Available Tools

| Tool | Description |
|------|-------------|
| `record_file_write` | Record a file write task (path, source) |
| `record_file_read` | Record a file read task (path, source) |
| `record_command` | Record a command execution task (command, source) |
| `record_note` | Emit a Note task — developer annotation included in the next Blob |
| `submit_recap` | Submit an AgentRecap task directly (bypasses Stop hook; use when ending a sub-task) |
| `query_status` | Return Lore repository status (blob counts, node counts, pending tasks) |
| `query_log` | List blobs newest-first, equivalent to `lore log` |
| `query_why` | Return blobs associated with a file path, equivalent to `lore why <file>` |
| `query_blob` | Return full detail for one blob by ID, equivalent to `lore show <id>` |
| `query_nodes` | List all subsystem nodes, equivalent to `lore node list` |
| `query_node` | Return blobs assigned to a subsystem node, equivalent to `lore node show <name>` |

The `record_*` tools are lower-level equivalents of the PostToolUse hooks. When both are active (normal operation after `lore init`), the hooks are preferred — they fire automatically without requiring the agent to call a tool. The MCP `record_*` tools exist for cases where the agent wants explicit control or needs to annotate something outside of a tool use boundary.

`submit_recap` is useful when an agent completes a well-defined sub-task mid-session and wants to force a clean Blob boundary before continuing. This is optional — the Stop hook handles the normal case.

---

## Verification

After `lore init`, run:

```bash
lore doctor
```

A healthy Claude Code install shows:

```
lore doctor

✓  .lore/ directory exists
✓  SQLite database initialized
✓  Git post-commit hook installed
✓  .claude/settings.json exists
✓  PostToolUse hook: Edit → lore hook file-write
✓  PostToolUse hook: Write → lore hook file-write
✓  PostToolUse hook: Read → lore hook file-read
✓  PostToolUse hook: Bash → lore hook command
✓  Stop hook → lore hook agent-recap
✓  MCP server registered: lore → lore mcp agent:claude
✓  Ollama available: llama3 (fallback inference ready)

All checks passed.
```

After running Claude Code on your repository for the first time, verify blobs were created:

```bash
lore log
```

```
abc1234  OAuth Provider Impl   Feature     2026-05-20  trust=AgentTruth   3 files
def5678  Fix JWT expiry        BugFix      2026-05-18  trust=AgentTruth   2 files
```

`trust=AgentTruth` confirms the Stop hook fired and the agent recap was ingested. If you see `trust=LoreInferred`, the Stop hook did not fire for that session — check the manual setup section below.

---

## Manual Setup

If `lore init` did not modify `.claude/settings.json` (for example, because you are adding Lore to an existing repository with a pre-existing settings file), add the following entries by hand.

If `.claude/settings.json` does not exist, create it:

```json
{
  "mcpServers": {
    "lore": {
      "command": "lore",
      "args": ["mcp", "agent:claude"]
    }
  },
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Edit",
        "hooks": [
          {
            "type": "command",
            "command": "lore hook file-write \"$CLAUDE_TOOL_INPUT_FILE_PATH\" agent:claude"
          }
        ]
      },
      {
        "matcher": "Write",
        "hooks": [
          {
            "type": "command",
            "command": "lore hook file-write \"$CLAUDE_TOOL_INPUT_FILE_PATH\" agent:claude"
          }
        ]
      },
      {
        "matcher": "Read",
        "hooks": [
          {
            "type": "command",
            "command": "lore hook file-read \"$CLAUDE_TOOL_INPUT_FILE_PATH\" agent:claude"
          }
        ]
      },
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "lore hook command \"$CLAUDE_TOOL_INPUT_COMMAND\" agent:claude"
          }
        ]
      }
    ],
    "Stop": [
      {
        "type": "command",
        "command": "lore hook agent-recap agent:claude"
      }
    ]
  }
}
```

If `.claude/settings.json` already exists with other content, merge in the `mcpServers.lore` entry and the four `PostToolUse` matchers and the `Stop` entry. Do not remove existing hooks.

After editing the file, verify with `lore doctor`.

---

## Trust Level

All data emitted by the Claude Code integration carries `trust_level=2` (AgentTruth). This is the second-highest trust level in Lore's hierarchy:

| Level | Name | Source |
|-------|------|--------|
| 1 | GroundTruth | Lore observed directly (file paths, commit SHAs, timestamps) |
| 2 | AgentTruth | Agent reported about its own work |
| 3 | HumanAssertion | User explicitly asserted (e.g. `lore assign`, `lore node create`) |
| 4 | LoreInference | Lore's local AI inferred from tasks |

AgentTruth blobs are displayed with `[AgentTruth]` in `lore show` output and as `trust=AgentTruth` in `lore log`.
