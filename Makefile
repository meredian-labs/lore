BINARY      = lore
VERSION    ?= dev
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS     = -ldflags "-s -w \
  -X github.com/nishchay/lore/internal/cli.Version=$(VERSION) \
  -X github.com/nishchay/lore/internal/cli.Commit=$(COMMIT) \
  -X github.com/nishchay/lore/internal/cli.BuildDate=$(BUILD_DATE)"
INSTALL_DIR ?= /usr/local/bin

.PHONY: build test lint install install-glh clean release snapshot

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/lore

test:
	go test ./...

lint:
	golangci-lint run ./...

install:
	go install $(LDFLAGS) ./cmd/lore

# Symlink glh -> lore so the same binary handles both CLIs.
install-glh: install
	ln -sf $(shell go env GOPATH)/bin/lore $(INSTALL_DIR)/glh

clean:
	rm -f lore glh

release:
	goreleaser release --clean

snapshot:
	goreleaser release --snapshot --clean --skip=publish
