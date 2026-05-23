---
layout: default
title: lore node
parent: CLI Reference
nav_order: 10
---
# lore node

Manage subsystem nodes — long-lived engineering topics that group related blobs.

## Subcommands

### lore node create

```bash
lore node create "<name>"
```

Create a new subsystem node.

```bash
lore node create "Authentication"
lore node create "Billing"
lore node create "API Gateway"
```

Node names are case-insensitive for lookups but displayed as entered.

### lore node list

```bash
lore node list
```

List all nodes with blob count and status.

```
Authentication    12 blobs  active
Billing           8 blobs   active
Session Mgmt      5 blobs   active
API Gateway       3 blobs   archived
```

### lore node show

```bash
lore node show "<name>"
```

Show blobs assigned to a node, newest first.

```bash
lore node show "Authentication"
```

```
Node: Authentication  (12 blobs)

abc1234  OAuth Provider Impl    Feature   2026-05-20  [AgentTruth]   3 files
def5678  JWT Expiry Fix         BugFix    2026-05-18  [LoreInferred]  2 files
ghi9012  Auth Middleware        Refactor  2026-05-15  [AgentTruth]   5 files
...
```

## What nodes are

Nodes represent stable, long-lived areas of your codebase — subsystems, services, or domains. They are _not_ tasks, migrations, or individual bug fixes. Those are blobs.

**Examples of nodes (correct):**
- `Authentication`
- `Billing`
- `Session Management`
- `API Gateway`

**Examples that are blobs, not nodes:**
- ~~OAuth Migration~~ → a blob, assigned to the `Authentication` node
- ~~JWT Expiry Fix~~ → a blob, assigned to the `Authentication` node
- ~~Billing Database Migration~~ → a blob, assigned to the `Billing` node

## Assigning blobs to nodes

After creating a node, use [`lore assign`](assign) to associate blobs with it:

```bash
lore assign abc1234 Authentication
```

## Node status

Nodes can be `active` or `archived`. Archived nodes are hidden from `lore graph` by default but still queryable. Archiving is a post-MVP feature.

## See also

- [`lore assign`](assign) — assign a blob to a node
- [`lore graph`](graph) — visualize nodes and their blobs
- [`lore log --all`](log) — tree view organized by node
