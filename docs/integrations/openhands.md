---
layout: default
title: OpenHands
parent: Agent Integrations
nav_order: 3
---
# OpenHands Integration

Lore integrates with OpenHands via the MCP protocol. The Lore MCP server exposes tools for recording file activity, commands, and session recaps. An OpenHands agent configured with the Lore MCP server can capture engineering knowledge as it works, producing AgentTruth-level blobs in the Lore knowledge graph.

---

## MCP Server

The Lore MCP server for OpenHands starts with:

```bash
lore mcp agent:openhands
```

This runs a JSON-RPC 2.0 server over stdio. OpenHands connects to it as a tool provider. The `agent:openhands` source string identifies tasks emitted during the session and sets `trust_level=2` (AgentTruth) on the resulting Blob.

---

## Config File

OpenHands reads MCP server configurations from its project-level config. Add the Lore server to your OpenHands MCP configuration file.

For a repository-local configuration, create or edit `.openhands/mcp.json`:

```json
{
  "mcpServers": {
    "lore": {
      "command": "lore",
      "args": ["mcp", "agent:openhands"]
    }
  }
}
```

If OpenHands uses a workspace-level config file (check your OpenHands version's documentation for the exact path — commonly `~/.openhands/config.toml` or a project `.openhands_instructions` file), add the Lore MCP server there.

Confirm the server is reachable after configuring:

```bash
lore doctor
```

Look for:

```
✓  MCP server: lore mcp agent:openhands starts successfully
```

---

## Session Workflow

A typical OpenHands session with Lore follows this pattern:

### 1. Agent receives a task

The user assigns a task to OpenHands, for example: "Add rate limiting to the API gateway."

### 2. Agent records activity during the session

As the agent edits files and runs commands, it calls Lore's MCP record tools:

After editing a file:
```
record_file_write(path="internal/gateway/ratelimit.go", source="agent:openhands")
```

After running a command:
```
record_command(command="go test ./internal/gateway/...", source="agent:openhands")
```

To leave an annotation mid-session:
```
record_note(note="Chose token bucket over sliding window due to lower memory overhead")
```

Notes are included in the blob's interpreted section and give Lore — and future developers — insight into decisions made during the session.

### 3. Agent submits a recap at the end

When the agent finishes the assigned task, it calls `submit_recap`:

```
submit_recap({
  "user_intent": "Add rate limiting to the API gateway to prevent abuse",
  "summary": "Implemented token bucket rate limiter in the gateway layer. Added per-client and global limits configurable via TOML. Updated integration tests.",
  "recap": "API gateway now enforces rate limits at the entry point, protecting downstream services from traffic spikes and reducing abuse surface.",
  "kind": "Feature",
  "tags": ["gateway", "rate-limiting", "security", "api"]
})
```

### 4. On the next Git commit, a Blob is created

When the developer commits the work, Lore's post-commit hook fires and creates a Blob. Because an `AgentRecap` task exists in the extraction window, the Blob is created with `trust_level=2` (AgentTruth) and the recap fields from step 3.

The blob is immediately queryable:

```bash
lore log
```

```
abc1234  Add rate limiting to API gateway  Feature  2026-05-20  trust=AgentTruth  4 files
```

```bash
lore show abc1234
```

```
ID:           abc1234
Title:        Add rate limiting to API gateway
Kind:         Feature
Trust:        AgentTruth (source: agent:openhands)

── Observed ────────────────────────────────────────
Started:      2026-05-20 14:02
Ended:        2026-05-20 15:44
Commits:      abc100..abc124

Files Modified:
  internal/gateway/ratelimit.go
  internal/gateway/ratelimit_test.go
  config/gateway.toml
  docs/gateway.md

Commands:
  go test ./internal/gateway/...
  go build ./...

── Interpreted ─────────────────────────────────────
User Intent:  Add rate limiting to the API gateway to prevent abuse
Summary:      Implemented token bucket rate limiter in the gateway layer.
              Added per-client and global limits configurable via TOML.
              Updated integration tests.
Recap:        API gateway now enforces rate limits at the entry point,
              protecting downstream services from traffic spikes and
              reducing abuse surface.
Tags:         gateway, rate-limiting, security, api
```

---

## Available MCP Tools

| Tool | Description |
|------|-------------|
| `record_file_write` | Record a file write task (path, source) |
| `record_file_read` | Record a file read task (path, source) |
| `record_command` | Record a command execution (command, source) |
| `record_note` | Emit a developer annotation included in the next Blob |
| `submit_recap` | Submit an AgentRecap task to close the current unit of work |
| `query_status` | Return repository Lore state (blob counts, pending tasks) |
| `query_log` | List blobs newest-first |
| `query_why` | Return blobs associated with a file |
| `query_blob` | Return full detail for one blob |
| `query_nodes` | List all subsystem nodes |
| `query_node` | Return blobs assigned to a subsystem |

The query tools let OpenHands answer questions about prior work before starting a new task — for example, calling `query_why` on a file to understand why it was written before modifying it.

---

## Recap Field Reference

| Field | Type | Purpose |
|-------|------|---------|
| `user_intent` | string | What the user asked the agent to do |
| `summary` | string | What the agent did (2–5 sentences) |
| `recap` | string | Why it matters in the bigger picture (1–3 sentences) |
| `kind` | string | One of: `Feature`, `BugFix`, `Migration`, `Investigation`, `Refactor`, `Architecture`, `Review`, `Incident` |
| `tags` | array of strings | Domain concepts relevant to the work |

All fields are optional but `kind` and `tags` significantly improve the usefulness of `lore graph` and `lore why` output. At minimum, provide `user_intent` and `summary`.
