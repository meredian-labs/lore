---
layout: default
title: Introduction
nav_order: 2
---
# Lore

**Git stores what changed. Lore stores why it changed.**

## The Problem

Six months after a team ships a feature, the reasoning is gone. Why was this file restructured? What was the investigation that preceded the migration? Who decided this module boundary, and why?

`git log` shows commits. `git blame` shows who touched a line. Neither tells you what the engineer was trying to accomplish, what they discovered along the way, or how this file relates to the broader system.

That context lives in Slack threads, Jira tickets, Notion pages, and aging memory. Lore captures it automatically, at the source, as the work happens.

## What Lore Is

Lore is a **local-first engineering memory system**. It runs entirely on your machine, requires no network connection, and stores everything in a single SQLite file at `.lore/lore.db`.

It observes your development activity — file writes, commands, commits, and AI agent summaries — and compresses that activity into durable **Blobs**: structured, human-readable records of engineering work. Related Blobs are grouped into long-lived **Nodes** that represent the stable subsystems of your codebase.

## How It Differs from Git

| Git | Lore |
|-----|------|
| Records what changed (diffs) | Records why it changed (intent) |
| Commit-level granularity | Work-session granularity |
| Manual authorship (commit messages) | Automatic capture + AI interpretation |
| Optimized for code retrieval | Optimized for context retrieval |

Lore does not replace Git. It reads Git and adds a layer of meaning on top.

## What It Looks Like

```
$ lore log

a3f1c82  OAuth provider implementation   Feature      2026-05-20  [AgentTruth]  3 files
8d2b019  Fix JWT token expiry edge case  BugFix       2026-05-18  [LoreInferred] 2 files
c91a447  Auth middleware refactor        Refactor     2026-05-15  [AgentTruth]  5 files
f3e8d21  Billing service extraction      Migration    2026-05-12  [AgentTruth]  9 files
```

Each line is a Blob — a compressed unit of engineering work with observed facts (files, commits, timestamps) and interpreted meaning (title, summary, intent) kept clearly separate.

```
$ lore why internal/auth/oauth.go

OAuth provider implementation (Feature)  2026-05-20  [AgentTruth]
  Replaced legacy token auth with OAuth2 provider flow.
  Commits: abc100..abc123  Node: Authentication

Fix JWT token expiry edge case (BugFix)  2026-05-18  [LoreInferred]
  Corrected expiry calculation off-by-one in refresh path.
  Commits: def456
```

## Key Properties

- **No daemon.** No background process. No `lore start` / `lore stop`.
- **No network.** All AI inference runs locally via Ollama (optional).
- **No lock-in.** Everything is in SQLite. Export any time.
- **Agent-aware.** Claude, Cursor, and Windsurf automatically contribute high-trust recaps.
