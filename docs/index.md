---
layout: default
title: Home
nav_order: 1
---
# Lore

> Git stores what changed. Lore stores why it changed.

Lore is a repository-local knowledge engine that observes development activity and turns it into permanent, queryable engineering memory.

It captures:

- **Tasks** — atomic observed actions (file writes, commands, commits, agent activity)
- **Blobs** — compressed units of engineering work with full context
- **Nodes** — long-lived subsystems that collect the story of ongoing work
- **Knowledge Graph** — navigational index of relationships across all of the above

---

## Quick Start

```bash
# Initialize lore in your repo
lore init

# Work normally — lore captures everything automatically
git commit -m "oauth provider"

# See what lore knows
lore log

# Understand a file
lore why internal/auth/oauth.go
```

---

## Documentation

- [Introduction](introduction.md)
- [Installation](installation.md)
- [Quick Start](quickstart.md)

## Core Concepts

- [Tasks](concepts/tasks.md)
- [Blobs](concepts/blobs.md)
- [Nodes](concepts/nodes.md)
- [Knowledge Graph](concepts/graph.md)

## Agent Integrations

- [Claude Code](integrations/claude-code.md)
- [Cursor](integrations/cursor.md)
- [OpenHands](integrations/openhands.md)
- [Custom Agents](integrations/custom-agent.md)

## CLI Reference

- [lore init](cli/init.md)
- [lore log](cli/log.md)
- [lore show](cli/show.md)
- [lore why](cli/why.md)
- [lore trace](cli/trace.md)
- [lore graph](cli/graph.md)
- [lore status](cli/status.md)
- [lore record](cli/record.md)
- [lore doctor](cli/doctor.md)
- [lore node](cli/node.md)
- [lore assign](cli/assign.md)
- [glh](cli/glh.md)

## Architecture

- [Storage Tiers](architecture/storage.md)
- [Schema](architecture/schema.md)
- [Trust Model](architecture/trust-model.md)
