---
layout: default
title: Installation
nav_order: 3
---
# Installation

## Prerequisites

- **Go 1.22 or later** — [install Go](https://go.dev/dl/)
- **Git** — any version with hook support (`git init` must work)
- **Ollama** (optional) — for local AI inference; Lore falls back to heuristics without it

Verify your Go version:

```bash
go version
# go version go1.22.3 darwin/arm64
```

## Install via go install

```bash
go install github.com/nishchay/lore/cmd/lore@latest
```

This places the `lore` binary in `$(go env GOPATH)/bin`. Make sure that directory is on your `PATH`:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

Add that line to your shell profile (`~/.zshrc`, `~/.bashrc`, etc.) to make it permanent.

## Build from Source

```bash
git clone https://github.com/nishchay/lore.git
cd lore
make build
```

This produces a `lore` binary in the current directory. To install it to your system:

```bash
make install
# installs to $(GOPATH)/bin/lore
```

Or install to a custom location:

```bash
INSTALL_DIR=/usr/local/bin make install-glh
```

## Installing glh

`glh` is the same binary as `lore`, invoked under a different name to act as a Git-aware wrapper. It intercepts `commit`, `checkout`, `merge`, and other subcommands to emit Tasks automatically, then passes everything else through to Git unchanged.

Install via `make`:

```bash
make install-glh
# creates /usr/local/bin/glh -> $(GOPATH)/bin/lore
```

The Makefile target is:

```makefile
install-glh: install
	ln -sf $(shell go env GOPATH)/bin/lore $(INSTALL_DIR)/glh
```

Or install it manually:

```bash
ln -sf $(which lore) /usr/local/bin/glh
```

Verify the symlink:

```bash
ls -la /usr/local/bin/glh
# /usr/local/bin/glh -> /Users/you/go/bin/lore

glh --version
# glh version dev (lore/cmd/lore)
```

Once `glh` is installed, use it in place of `git` in your day-to-day workflow. It is a transparent pass-through: `glh push`, `glh diff`, `glh rebase` all behave exactly like their `git` equivalents.

## Verify the Installation

```bash
lore --version
# lore version dev

lore doctor
# checking prerequisites...
#   git          ok  (/usr/bin/git, version 2.44.0)
#   go           ok  (/usr/local/bin/go, version 1.22.3)
#   ollama       ok  (running, model: llama3)
#   glh          ok  (/usr/local/bin/glh -> /Users/you/go/bin/lore)
# all checks passed
```

`lore doctor` will warn — but not block — if Ollama is unavailable. Lore operates fully without it using heuristic inference.

## Shell Completions (Optional)

Lore uses Cobra and can generate completions for bash, zsh, fish, and PowerShell:

```bash
# zsh
lore completion zsh > "${fpath[1]}/_lore"

# bash
lore completion bash > /etc/bash_completion.d/lore

# fish
lore completion fish | source
```
