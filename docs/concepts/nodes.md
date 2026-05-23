---
id: nodes
title: Nodes
sidebar_label: Nodes
sidebar_position: 3
---

# Nodes

A Node is a long-lived, named subsystem of your codebase. It groups related Blobs over time into a coherent topic — an area of the system that has identity, history, and evolution.

**Nodes are permanent. They represent stable engineering domains, not individual tasks.**

## Nodes Are Not Blobs

This distinction matters:

| Node (correct) | Blob (not a Node) |
|----------------|-------------------|
| Authentication | OAuth provider implementation |
| Billing | Stripe integration migration |
| Session Management | JWT expiry fix |
| API Gateway | Rate limiting refactor |

A Node is a place. A Blob is something that happened in that place. The OAuth implementation Blob belongs to the Authentication Node. The JWT fix Blob also belongs to the Authentication Node. Years later, an auth token rotation Blob will also belong there.

Nodes accumulate Blobs. Blobs do not accumulate Nodes.

## Creating a Node

```bash
lore node create "Authentication"
```

```
created node: Authentication
id: n-8f3a1b2
hint: use 'lore assign <blob-id> Authentication' to assign blobs to this node
```

Node names are case-insensitive for lookup but display-cased as you provided them. You can use quoted multi-word names:

```bash
lore node create "API Gateway"
lore node create "Billing"
lore node create "Session Management"
```

## Listing Nodes

```bash
lore node list
```

```
n-8f3a1b2  Authentication     12 blobs  active  last active: 2026-05-20
n-3c9d4e5  Billing             8 blobs  active  last active: 2026-05-19
n-7a2f6b1  Session Management  5 blobs  active  last active: 2026-05-15
n-1e4d8c3  API Gateway         3 blobs  archived last active: 2026-03-02
```

## Showing a Node

```bash
lore node show Authentication
```

```
Node: Authentication
ID:   n-8f3a1b2
Blobs: 12  Status: active

── Blobs ───────────────────────────────────────────
a3f1c82  OAuth provider implementation   Feature   2026-05-20  [AgentTruth]
8d2b019  Fix JWT token expiry edge case  BugFix    2026-05-18  [LoreInferred]
c91a447  Auth middleware refactor        Refactor  2026-05-15  [AgentTruth]
f3e8d21  Legacy token removal            Migration 2026-05-12  [AgentTruth]
d2c9a11  Auth service extraction         Refactor  2026-04-28  [AgentTruth]
...

── Files ────────────────────────────────────────────
Most modified files in this node:
  internal/auth/oauth.go          (7 blobs)
  internal/auth/middleware.go     (5 blobs)
  internal/session/manager.go     (4 blobs)
  internal/auth/middleware_test.go (3 blobs)
```

## Assigning Blobs to a Node

Blob assignment is a **HumanAssertion** (trust level 3). It is explicit and intentional.

```bash
lore assign a3f1c82 Authentication
```

```
assigned blob a3f1c82 → Authentication  [HumanAssertion, trust=3]
```

You can also assign by Node ID:

```bash
lore assign a3f1c82 n-8f3a1b2
```

Assignment is idempotent. Assigning an already-assigned Blob to the same Node does nothing. Reassigning to a different Node updates `primary_node_id` and records the change.

## Unassigned Blobs

Blobs that have not been assigned to a Node are valid — `primary_node_id` is nullable. They appear in `lore log` and `lore show` normally, and are surfaced in `lore status` and `lore graph` as unassigned.

```bash
lore status
# ...
# Unassigned Blobs: 3
#   hint: use 'lore assign <id> <node>' or 'lore node create <name>'
```

## A Node Over Time

Here is what an Authentication Node looks like after six months of active development:

```
Node: Authentication  (12 blobs, active)

2026-05-20  OAuth provider implementation      Feature     [AgentTruth]
2026-05-18  Fix JWT token expiry edge case     BugFix      [LoreInferred]
2026-05-15  Auth middleware refactor           Refactor    [AgentTruth]
2026-05-12  Legacy token removal               Migration   [AgentTruth]
2026-04-28  Auth service extraction            Refactor    [AgentTruth]
2026-04-10  Auth system investigation          Investigation [LoreInferred]
2026-03-22  Add MFA support                    Feature     [AgentTruth]
2026-03-14  Session token rotation             Feature     [AgentTruth]
2026-02-28  Rate limiting on auth endpoints    Feature     [AgentTruth]
2026-02-10  Auth service initial extraction    Migration   [AgentTruth]
2026-01-20  Password hashing upgrade           BugFix      [AgentTruth]
2025-12-04  Original auth implementation       Feature     [LoreInferred]
```

This is the history that `git log` cannot give you. Not a list of commits — a list of decisions, each with intent and context.

## Node Assignment Trust

Node assignment uses trust level 3 (HumanAssertion), which is higher than Lore's own inference (trust 4). When you explicitly assign a Blob to a Node, that assignment is treated as ground truth for navigation and graph construction.

Post-MVP, Lore will offer `lore node suggest` to propose assignments for unassigned Blobs — but those suggestions will always require explicit `lore assign` confirmation to persist.
