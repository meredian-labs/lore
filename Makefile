BINARY   = lore
VERSION ?= dev
LDFLAGS  = -ldflags "-X github.com/nishchay/lore/internal/cli.Version=$(VERSION)"
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
