---
sidebar_position: 4
---

# Custom Agent Integration

Any AI agent or automated tool can integrate with Lore. There are two integration paths: MCP (recommended for agents with MCP support) and direct hook calls (for agents or scripts that just run shell commands).

Both paths produce the same result: `trust_level=2` (AgentTruth) blobs when a recap is submitted, or `trust_level=4` (LoreInference) blobs when no recap is provided.

---

## Option A: MCP Integration

MCP (Model Context Protocol) is the recommended path for agents that support it. It provides structured tool calls with typed inputs and outputs, no shell escaping, and direct access to all of Lore's query capabilities.

### Starting the MCP Server

```bash
lore mcp agent:<yourname>
```

Replace `<yourname>` with a short identifier for your agent — for example, `agent:mybot` or `agent:pipeline`. This string becomes the `ai_source` value on every task and blob produced during the session.

The server speaks JSON-RPC 2.0 over stdio. It starts, reads requests from stdin, writes responses to stdout, and runs until the process exits.

### Configuration

Register the server in your agent's MCP config. The exact config file path varies by agent, but the structure is:

```json
{
  "mcpServers": {
    "lore": {
      "command": "lore",
      "args": ["mcp", "agent:mybot"]
    }
  }
}
```

### Available Tools

| Tool | Inputs | Description |
|------|--------|-------------|
| `record_file_write` | `path`, `source` | Record a file write task |
| `record_file_read` | `path`, `source` | Record a file read task |
| `record_command` | `command`, `source` | Record a command execution |
| `record_note` | `note` | Emit a developer annotation included in the next Blob |
| `submit_recap` | recap object (see below) | Submit an AgentRecap task |
| `query_status` | — | Repository Lore state |
| `query_log` | `limit` (optional) | Blobs newest-first |
| `query_why` | `path` | Blobs associated with a file |
| `query_blob` | `id` | Full detail for one blob |
| `query_nodes` | — | All subsystem nodes |
| `query_node` | `name` | Blobs assigned to a subsystem |

---

## Option B: Direct Hook Calls

For agents or scripts that cannot use MCP, Lore exposes the same functionality as shell commands. These are the same commands that Git hooks and Claude Code's PostToolUse hooks call internally.

### Recording a File Write

```bash
lore hook file-write <path> <source>
```

Example:

```bash
lore hook file-write internal/auth/oauth.go agent:mybot
```

### Recording a File Read

```bash
lore hook file-read <path> <source>
```

### Recording a Command

```bash
lore hook command "<command string>" <source>
```

Example:

```bash
lore hook command "go test ./internal/auth/..." agent:mybot
```

### Submitting an Agent Recap

```bash
lore hook agent-recap <source>
```

The `agent-recap` command reads JSON from stdin:

```bash
cat <<'EOF' | lore hook agent-recap agent:mybot
{
  "user_intent": "Add Google OAuth support to replace legacy token auth",
  "summary": "Implemented OAuth2 provider flow, callback handler, and session integration.",
  "recap": "Authentication subsystem migrated toward provider-based login. This eliminates the legacy token system maintenance burden.",
  "kind": "Feature",
  "tags": ["auth", "oauth", "session", "provider"]
}
EOF
```

Exit code 0 means the recap was stored. Non-zero means an error — check stderr for details.

---

## Agent Recap JSON Schema

The recap JSON submitted via `lore hook agent-recap` or the `submit_recap` MCP tool has this schema:

```json
{
  "user_intent": "string — what the user asked the agent to do (max 200 chars)",
  "summary":     "string — what the agent did, 2–5 sentences (max 500 chars)",
  "recap":       "string — why it matters in the bigger picture, 1–3 sentences (max 300 chars)",
  "kind":        "Feature | BugFix | Migration | Investigation | Refactor | Architecture | Review | Incident",
  "tags":        ["array", "of", "domain", "concepts"]
}
```

### Field Purpose

| Field | Required | Purpose |
|-------|----------|---------|
| `user_intent` | Recommended | Preserves the original goal. Shown in `lore show` under Interpreted. Useful for understanding why the work happened, not just what. |
| `summary` | Recommended | The factual description of what was done. This is the most-read field — keep it precise. |
| `recap` | Optional | The broader significance. Useful for `lore graph` and `lore why` when a file appears in multiple blobs. |
| `kind` | Recommended | Drives `lore status` breakdowns and graph coloring. Defaults to `Feature` if omitted. |
| `tags` | Optional | Drives concept node creation in the knowledge graph. Directory names are used as a fallback. |

All fields are optional. An empty recap is valid — Lore will fall back to LoreInference for any missing fields.

---

## Trust Levels

The trust level on a blob reflects the source of its interpreted fields. This is always visible in `lore log` and `lore show` output.

| Level | Name | When Applied |
|-------|------|-------------|
| 1 | GroundTruth | Observed facts: file paths, commit SHAs, timestamps, commands. Always set by Lore, never by agents. |
| 2 | AgentTruth | Agent submitted a recap via `submit_recap` or `lore hook agent-recap`. Applied to `title`, `summary`, `recap`, `user_intent`, `kind`, `tags`. |
| 3 | HumanAssertion | User ran `lore assign` or `lore node create`. Applied to `primary_node_id` and node `title`. |
| 4 | LoreInference | No recap was submitted. Lore's local Ollama model (or heuristic fallback) generated the interpreted fields. |

Trust level 2 (AgentTruth) is the best a custom agent can produce. It is preferred over LoreInference because the agent that performed the work has more context about intent and outcome than any post-hoc inference can recover.

In `lore log` output, trust levels appear as:

```
abc1234  OAuth Provider Impl  Feature  2026-05-20  trust=AgentTruth   3 files
def5678  Fix JWT expiry       BugFix   2026-05-18  trust=LoreInferred  2 files
```

In `lore show` output, the full source is shown:

```
Trust:        AgentTruth (source: agent:mybot)
```

---

## Source String Convention

The source string identifies who or what produced the data. It appears in `ai_source` on blobs and in task records.

| Pattern | Used For |
|---------|---------|
| `agent:<name>` | AI coding agents (Claude Code uses `agent:claude`, Cursor uses `agent:cursor`) |
| `human:<name>` | Human developer actions captured via `glh commit --recap` or direct annotation |
| `ci:<name>` | CI pipeline actions (e.g. `ci:github-actions`, `ci:jenkins`) |

Use `agent:<yourname>` for any AI agent. Keep `<yourname>` short, lowercase, and stable — it appears in stored records and changing it later creates inconsistency in the `ai_source` column.

---

## Complete Example: Minimal Python Agent

The following is a minimal Python agent that calls Lore hooks after each action. It uses subprocess calls (Option B) rather than MCP, keeping dependencies minimal.

```python
import subprocess
import json
import sys
import os


def lore_record_file_write(path: str, source: str = "agent:mybot") -> None:
    """Record a file write to Lore. Best-effort — never raises."""
    try:
        subprocess.run(
            ["lore", "hook", "file-write", path, source],
            check=True,
            capture_output=True,
        )
    except (subprocess.CalledProcessError, FileNotFoundError):
        pass  # Lore not installed or hook failed — continue without it


def lore_record_command(command: str, source: str = "agent:mybot") -> None:
    """Record a command execution to Lore. Best-effort — never raises."""
    try:
        subprocess.run(
            ["lore", "hook", "command", command, source],
            check=True,
            capture_output=True,
        )
    except (subprocess.CalledProcessError, FileNotFoundError):
        pass


def lore_submit_recap(
    user_intent: str,
    summary: str,
    recap: str,
    kind: str,
    tags: list[str],
    source: str = "agent:mybot",
) -> bool:
    """Submit an agent recap to Lore. Returns True on success."""
    recap_json = json.dumps({
        "user_intent": user_intent,
        "summary": summary,
        "recap": recap,
        "kind": kind,
        "tags": tags,
    })
    try:
        result = subprocess.run(
            ["lore", "hook", "agent-recap", source],
            input=recap_json,
            text=True,
            capture_output=True,
        )
        return result.returncode == 0
    except FileNotFoundError:
        return False


def write_file(path: str, content: str) -> None:
    """Write content to a file and record it in Lore."""
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w") as f:
        f.write(content)
    lore_record_file_write(path)


def run_command(command: str) -> str:
    """Run a shell command and record it in Lore."""
    lore_record_command(command)
    result = subprocess.run(
        command,
        shell=True,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise RuntimeError(f"Command failed: {command}\n{result.stderr}")
    return result.stdout


def main():
    # Example: agent adds a rate limiter to the gateway

    write_file(
        "internal/gateway/ratelimit.go",
        "package gateway\n\n// RateLimiter implements token bucket rate limiting.\n",
    )

    write_file(
        "internal/gateway/ratelimit_test.go",
        "package gateway\n\nimport \"testing\"\n\nfunc TestRateLimiter(t *testing.T) {}\n",
    )

    run_command("go test ./internal/gateway/...")
    run_command("go build ./...")

    # Submit recap when the task is complete
    success = lore_submit_recap(
        user_intent="Add rate limiting to the API gateway",
        summary=(
            "Implemented token bucket rate limiter in the gateway package. "
            "Added test coverage. All tests pass."
        ),
        recap=(
            "Gateway now enforces per-client rate limits before forwarding requests, "
            "reducing downstream load during traffic spikes."
        ),
        kind="Feature",
        tags=["gateway", "rate-limiting", "api"],
    )

    if success:
        print("Lore recap submitted. This session will produce an AgentTruth blob.")
    else:
        print("Lore not available. Work will be captured as LoreInference on next commit.")


if __name__ == "__main__":
    main()
```

The key pattern: record after each action, submit recap once at the end. Lore calls are always best-effort — if Lore is not installed or the hook fails, the agent continues normally. Lore never blocks agent operation.

---

## Integration Checklist

Before shipping an agent integration:

- [ ] Calls `lore hook file-write` (or `record_file_write`) after every file modification
- [ ] Calls `lore hook command` (or `record_command`) after every shell command
- [ ] Calls `lore hook agent-recap` (or `submit_recap`) at the end of each logical unit of work
- [ ] Uses a stable source string in `agent:<name>` format
- [ ] Does not crash or block when Lore is unavailable
- [ ] Does not send file contents or diffs to Lore (paths and command strings only)

The last point is important: Lore records **what happened** (paths, commands, commit SHAs), not **what the content was**. Never pass file contents to any Lore hook or MCP tool.
