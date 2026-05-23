---
title: lore record
---

# lore record

Emit a `Note` task that will be included in the next blob extraction.

## Usage

```bash
lore record "<note>"
```

## Examples

```bash
lore record "investigating JWT expiry bug — started at the token refresh path"
lore record "decided to use PKCE flow instead of implicit grant for security"
lore record "TODO: refactor session manager after this PR merges"
```

## What it does

`lore record` inserts a `KindNote` task with `trust_level=1` (GroundTruth) into the pending task queue. The note is a developer annotation — a free-text observation about what you are doing or why.

On the next `git commit`, this note is included in the extraction window and becomes part of the resulting blob's context. When an AI recap is present, the note supplements the agent's interpretation. When Lore falls back to heuristic extraction, the note provides critical signal that would otherwise be inferred from file changes alone.

## Use cases

- **Context before a commit**: annotate a non-obvious decision before you commit
- **Investigation notes**: record what you discovered mid-session
- **Intention markers**: explain why a refactor is happening

## Notes appear in blobs

After the next extraction, you can see the note in `lore show <id>`:

```
── Interpreted ─────────────────────────────────────────────────────
User Intent:  Investigating JWT expiry bug in token refresh path
Summary:      Traced expiry issue to a timezone offset in token
              generation. Fixed by normalising to UTC before comparison.
Notes:        "investigating JWT expiry bug — started at the token refresh path"
Tags:         auth, jwt, token, bug
```

## See also

- [`lore show`](show.md) — view a blob's full detail including notes
- [`glh commit --recap`](glh.md) — provide a recap at commit time
