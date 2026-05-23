---
layout: default
title: glh
parent: CLI Reference
nav_order: 12
---
# glh

Git with Lore hooks — a drop-in git wrapper for daily use.

## Synopsis

```
glh <git-command> [flags]
glh commit [git-flags] [--recap]
glh checkout <branch>
glh switch <branch>
glh merge <branch>
glh log [git-flags]
glh status
glh <anything>
```

## Description

`glh` is a thin wrapper around `git` that emits Lore tasks for relevant operations. It is designed to be used as your daily git driver without changing how you work.

For commands Lore cares about (`commit`, `checkout`, `switch`, `merge`, `log`, `status`), `glh` runs the git command and then emits the appropriate Lore task. For everything else, `glh` passes the command through to `git` unchanged.

## Installation

`glh` ships as a separate binary alongside `lore`. After installation, symlink it into your PATH:

```bash
# Symlink glh into a directory on your PATH
ln -s $(which glh) /usr/local/bin/glh

# Or, if installed via go install:
ln -s ~/go/bin/glh /usr/local/bin/glh
```

Verify:

```bash
glh --version
# glh 0.1.0 (lore 0.1.0)
```

## How Hook Detection Works

`glh commit` checks whether the repository is already wired to Lore's Git hooks before deciding whether to emit tasks manually:

- If `git config core.hooksPath` returns `.lore/hooks`, the repository is Lore-initialized. The post-commit hook will fire automatically — `glh` does **not** call `lore hook commit-created` again.
- If `core.hooksPath` is not set (non-Lore repository), `glh commit` calls `lore hook commit-created` explicitly after the commit completes.

This means `glh` is safe to use in both Lore-initialized and plain Git repositories.

## Subcommands

### glh commit

```
glh commit [git-flags] [--recap]
```

Runs `git commit` with all provided flags. After the commit, if the repository does not have `core.hooksPath=.lore/hooks`, emits a `CommitCreated` task directly.

The `--recap` flag prompts for a session recap after the commit:

```
$ glh commit -m "feat: implement OAuth provider" --recap

[main abc123] feat: implement OAuth provider
 3 files changed, 187 insertions(+), 42 deletions(-)

-- Lore Recap --
Describe what you were working on (leave blank to skip):
> Replaced legacy token auth with Google OAuth2. Added callback handler and
  updated session manager to use provider tokens.

Recap recorded. This will be used as AgentTruth for the next blob extraction.
```

The recap text is stored as an `AgentRecap` task with `trust_level=2`.

### glh checkout / glh switch

```
glh checkout <branch>
glh switch <branch>
```

Runs the standard git command and emits a `BranchSwitch` task recording the previous branch, the new branch, and the timestamp.

```
$ glh checkout feature/billing

Switched to branch 'feature/billing'
[lore: BranchSwitch recorded — main → feature/billing]
```

### glh merge

```
glh merge <branch>
```

Runs `git merge` and emits a `MergeEvent` task recording the source branch, target branch, and resulting commit SHA.

```
$ glh merge feature/oauth

Merge made by the 'ort' strategy.
 internal/auth/oauth.go | 187 ++++++++++
[lore: MergeEvent recorded — feature/oauth → main @ abc456]
```

### glh log

```
glh log [git-flags]
```

Runs `git log` with all provided flags, then annotates commit lines with a `●` marker where a Lore blob exists.

```
$ glh log --oneline

● abc123  feat: implement OAuth provider flow
  def456  chore: update dependencies
● ghi789  fix: correct JWT expiry calculation
```

See [`lore log`](./log) for full documentation of `glh log` output.

### glh status

```
glh status
```

Runs `git status` and appends a Lore footer showing the number of pending tasks awaiting the next blob extraction.

```
$ glh status

On branch main
Your branch is up to date with 'origin/main'.

Changes not staged for commit:
  modified:   internal/auth/oauth.go

-- Lore --
Pending tasks: 4 (FileWrite ×3, Command ×1)
hint: commit to trigger blob extraction
```

### glh \<anything\>

Any git command not listed above is passed through to `git` unchanged. `glh` is a transparent proxy:

```
glh push origin main
glh pull --rebase
glh stash pop
glh rebase -i HEAD~3
glh diff --stat
```

These produce identical output to running `git` directly.

## Flags

| Flag | Command | Description |
|------|---------|-------------|
| `--recap` | `commit` | Prompt for session recap after commit |
| `--no-lore` | all | Skip all Lore task emission (pure git passthrough) |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `LORE_SKIP=1` | Disable all Lore task emission for this command |

## See Also

- [`lore log`](./log) — `lore log` and `glh log` output reference
- [`lore init`](./init) — sets `core.hooksPath` so Git hooks fire automatically
- [`lore record`](../cli/record) — emit a note task without committing
