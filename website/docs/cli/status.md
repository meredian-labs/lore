---
title: lore status
---

# lore status

Show the current lore state of the repository.

## Usage

```bash
lore status [--json]
```

## Output

```
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

## Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |

## What it shows

- **Repository path** and initialization date (from the `initialized_at` meta key)
- **Blob counts** broken down by kind and trust level
- **Nodes** with blob count and active/archived status
- **Unassigned blobs** with a hint to assign them
- **Pending tasks** — tasks captured since the last extraction; these will be included in the next blob
- **LLM** — whether the configured Ollama endpoint is reachable

## See also

- [`lore doctor`](doctor.md) — full installation health check
- [`lore log`](log.md) — list all blobs
