# Cursor Integration

Lore integrates with Cursor via its MCP (Model Context Protocol) support. After `lore init`, the Lore MCP server is registered in your repository's `.cursor/mcp.json` file. Once enabled in Cursor's settings, the agent can query Lore's knowledge graph and record activity during a session.

---

## What `lore init` Writes

`lore init` creates `.cursor/mcp.json` in your repository root:

```json
{
  "mcpServers": {
    "lore": {
      "command": "lore",
      "args": ["mcp", "agent:cursor"]
    }
  }
}
```

This tells Cursor to start `lore mcp agent:cursor` as a stdio MCP server whenever it opens this repository. The server exposes Lore's 11 tools to the Cursor agent.

---

## Enabling in Cursor

After `lore init` writes `.cursor/mcp.json`, you need to enable MCP in Cursor's settings:

1. Open **Cursor Settings** (`Cmd+,` on macOS)
2. Navigate to **Features**
3. Find **MCP** and ensure it is toggled on
4. Restart Cursor (or reload the window with `Cmd+Shift+P` → `Reload Window`)

Cursor will pick up `.cursor/mcp.json` automatically when it reloads. The `lore` server will appear in the MCP panel as connected.

To verify the server started:

```bash
lore doctor
```

Look for:

```
✓  .cursor/mcp.json exists
✓  MCP server registered: lore → lore mcp agent:cursor
```

---

## Using Lore Tools in Cursor Agent Mode

Once the MCP server is connected, all 11 Lore tools are available in Cursor's agent mode. The agent can call them directly.

### Query Tools (read Lore's knowledge)

| Tool | Equivalent CLI | What It Returns |
|------|---------------|-----------------|
| `query_status` | `lore status` | Blob counts, node counts, pending tasks |
| `query_log` | `lore log` | List of blobs newest-first |
| `query_why` | `lore why <file>` | Blobs associated with a specific file path |
| `query_blob` | `lore show <id>` | Full detail for one blob |
| `query_nodes` | `lore node list` | All subsystem nodes |
| `query_node` | `lore node show <name>` | Blobs assigned to a subsystem |

Example: asking the agent "why does `internal/auth/oauth.go` exist?" — the agent can call `query_why` with `path=internal/auth/oauth.go` and return the history of blobs that modified that file.

### Record Tools (write to Lore)

| Tool | What It Does |
|------|-------------|
| `record_file_write` | Record that a file was written (path, source) |
| `record_file_read` | Record that a file was read (path, source) |
| `record_command` | Record that a command was executed (command, source) |
| `record_note` | Emit a developer annotation included in the next Blob |
| `submit_recap` | Submit an AgentRecap task for the current session |

The Cursor agent should call `record_file_write` after modifying a file, `record_command` after running a shell command, and `submit_recap` when it finishes a unit of work.

### Submitting a Recap

At the end of a task, the agent should call `submit_recap` with a structured recap:

```json
{
  "user_intent": "Refactor billing module to use the new payment gateway",
  "summary": "Replaced Stripe v1 calls with the new payment-gateway client. Updated tests.",
  "recap": "Billing subsystem now uses the unified payment-gateway abstraction, removing direct Stripe SDK dependency.",
  "kind": "Refactor",
  "tags": ["billing", "payment-gateway", "stripe"]
}
```

This produces a Blob with `trust=AgentTruth` — the highest-trust AI interpretation Lore can record.

---

## Limitation: No PostToolUse Hooks

Cursor does not support PostToolUse hooks (the automatic per-tool-call hooks that Claude Code uses). This means file writes and commands are not captured automatically after each tool call.

For file and command tracking in Cursor, the agent must call the MCP record tools explicitly:

- Call `record_file_write` after editing a file
- Call `record_command` after running a shell command

This requires the agent to be instructed to do so. You can add a standing instruction in your Cursor rules file (`.cursorrules` or the rules section in Cursor settings):

```
After every file edit, call the lore MCP tool record_file_write with the path of the file you edited.
After every shell command, call the lore MCP tool record_command with the command you ran.
At the end of your response, call submit_recap with a structured summary of what you did.
```

Without these instructions, Lore will still create Blobs on each Git commit (via the Git post-commit hook installed by `lore init`), but the per-file and per-command granularity inside the Blob will be lower.

---

## Manual Setup

If `lore init` did not create `.cursor/mcp.json`, create it yourself:

```bash
mkdir -p .cursor
```

Then create `.cursor/mcp.json` with:

```json
{
  "mcpServers": {
    "lore": {
      "command": "lore",
      "args": ["mcp", "agent:cursor"]
    }
  }
}
```

If `.cursor/mcp.json` already exists with other MCP servers, add the `lore` entry to the existing `mcpServers` object:

```json
{
  "mcpServers": {
    "existing-server": {
      "command": "some-command",
      "args": []
    },
    "lore": {
      "command": "lore",
      "args": ["mcp", "agent:cursor"]
    }
  }
}
```

After saving, reload the Cursor window. Run `lore doctor` to confirm the configuration is detected.

---

## Trust Level

Activity recorded via the Cursor MCP integration carries `trust_level=2` (AgentTruth) when the agent calls `submit_recap`, and `trust_level=2` for individual `record_*` tasks. This is the same trust level as the Claude Code integration.

If no recap is submitted for a session, Lore falls back to `trust_level=4` (LoreInference) using the local Ollama model to interpret the observed tasks at commit time.
