---
id: assign
title: lore assign
sidebar_label: assign
sidebar_position: 11
---

# lore assign

Assign a blob to a subsystem node.

## Usage

```bash
lore assign <blob-id> <node-name>
```

The blob ID can be a 7-character prefix (same as `lore show`). The node name is case-insensitive.

## Examples

```bash
lore assign abc1234 Authentication
lore assign def5678 "API Gateway"
lore assign ghi9012 billing          # case-insensitive
```

## What it does

`lore assign` sets `blobs.primary_node_id` for the given blob to the target node's ID. This assignment is recorded at **trust level 3 (HumanAssertion)** — the highest trust level for blob-to-node assignments, above any automatic inference.

After assignment, the blob appears in:
- `lore node show <node>` — listed under the node
- `lore graph` — nested under the node in the ASCII tree
- `lore log --all` — shown in the correct node section
- `lore show <id>` — "Part of: NodeName" in the output

## Finding unassigned blobs

`lore status` shows how many blobs are unassigned:

```
Unassigned Blobs: 4
  hint: use 'lore assign <id> <subsystem>' or 'lore node create <name>'
```

`lore log --all` shows an **Unassigned** section at the bottom of the tree for blobs without a node.

## Trust level

Assignments made via `lore assign` use `trust_level=3` (HumanAssertion). This is considered authoritative — it takes precedence over any automatic node inference Lore may add in future versions.

## Workflow

```bash
# 1. See what's unassigned
lore status

# 2. Review the blobs
lore log -n 20

# 3. Create nodes if they don't exist
lore node create "Authentication"

# 4. Assign
lore assign abc1234 Authentication
lore assign def5678 Authentication
lore assign ghi9012 Billing
```

## See also

- [`lore node`](node) — create and manage nodes
- [`lore log --all`](log) — tree view showing node assignments
- [`lore graph`](graph) — visual overview of nodes and blobs
