# Lore — engineering memory for your repository

> Git stores what changed. Lore stores why it changed.

---

## Install

```bash
# Homebrew (recommended)
brew tap meredian-labs/tap
brew install lore

# Or via go install
go install github.com/meredian-labs/lore/cmd/lore@latest

# glh is a symlink installed alongside lore — no extra step needed
```

---

## What it does

- **Captures engineering context automatically** — Git hooks and agent integrations record file writes, commands, commits, and AI session recaps without any extra steps.
- **Compresses activity into durable Blobs** — after each commit, Lore extracts a structured record of what was done and why, with observed facts and AI interpretation clearly separated.
- **Builds a queryable knowledge graph** — Blobs are grouped into subsystem Nodes and indexed so you can ask "why does this file exist?" and get a real answer.

---

## Quickstart

```bash
# 1. Initialize Lore in your repository
cd my-project
lore init

# 2. Work normally and commit
glh commit -m "feat: implement OAuth provider"

# 3. See what Lore captured
lore log

# 4. Understand why a file exists
lore why internal/auth/oauth.go

# 5. Read the full context for a blob
lore show abc1234
```

---

## Agent Integrations

`lore init` auto-configures supported agents. After init, agent file writes, commands, and session recaps are captured automatically.

| Agent | Setup | What's tracked |
|-------|-------|----------------|
| Claude Code | `lore init` (auto) | File writes, commands, bash tool calls, session recap via Stop hook |
| Cursor | `lore init` (auto) | File edits, commands via MCP |
| Windsurf | `lore init` (auto) | File edits, commands via MCP |
| OpenHands | `lore hook agent-recap agent:openhands` | MCP tool calls, session recap |
| Any agent | `lore hook` / MCP integration | Custom integration via hook protocol |

When an agent provides a session recap, Lore uses it directly as `AgentTruth` (trust level 2) — higher confidence than Lore's own inference.

---

## Use glh as your daily git driver

`glh` is a drop-in git wrapper. All git flags pass through unchanged. Lore tasks are emitted automatically.

```bash
# Commit and prompt for a session recap
glh commit -m "fix: correct JWT expiry" --recap

# Annotated git log — ● marks commits with a Lore blob
glh log --oneline

# git status with pending Lore tasks footer
glh status
```

---

## Commands

| Command | Description |
|---------|-------------|
| `lore init` | Initialize `.lore/`, install git hooks, configure agents |
| `lore status` | Repository state: blob counts, nodes, pending tasks, LLM status |
| `lore log` | List blobs newest-first |
| `lore log --all` | File-explorer tree grouped by subsystem |
| `lore show <id>` | Full blob detail with Observed and Interpreted sections |
| `lore why <file>` | All blobs that modified this file, newest first |
| `lore trace <file>` | Chronological file history across blobs |
| `lore graph` | ASCII knowledge graph: subsystems → blobs → files |
| `lore record <note>` | Emit a note task included in the next blob |
| `lore doctor` | Check git repo, hooks, agent integrations, and LLM availability |
| `lore node create <name>` | Create a new subsystem node |
| `lore node list` | List all nodes with blob counts |
| `lore node show <name>` | Show blobs assigned to a node |
| `lore assign <id> <node>` | Assign a blob to a subsystem |
| `glh commit [--recap]` | git commit + Lore task + optional recap prompt |
| `glh log` | git log annotated with Lore blob markers |
| `glh status` | git status + pending Lore tasks footer |
| `glh <anything>` | Pure git passthrough |

---

## How it works

Lore operates in four tiers:

```
Git hooks + agent integrations
          │
          ▼
  Tier 1: Tasks          ← atomic observed actions (ephemeral, ~30 days)
          │  on commit: blob extraction
          ▼
  Tier 2: Blobs          ← units of engineering work (permanent)
          │  node resolution
          ▼
  Tier 3: Nodes          ← long-lived subsystems (permanent)
          │  graph derivation
          ▼
  Tier 4: Knowledge Graph ← navigational index (derived, permanent)
```

All data is stored locally in `.lore/lore.db` (SQLite). Nothing leaves your machine without explicit configuration.

---

## Documentation

Full documentation: [lore](https://meredian-labs.github.io/lore/)

- [CLI Reference](https://meredian-labs.github.io/lore/cli/init)
- [Architecture](https://meredian-labs.github.io/lore/architecture/storage)
- [Trust Model](https://meredian-labs.github.io/lore/architecture/trust-model)

---

## License

MIT. See [LICENSE](LICENSE).

Contributions welcome — open an issue or pull request.
