BINARY   = lore
VERSION ?= dev
LDFLAGS  = -ldflags "-X github.com/nishchay/lore/internal/cli.Version=$(VERSION)"

.PHONY: build test lint install clean release snapshot

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/lore

test:
	go test ./...

lint:
	golangci-lint run ./...

install:
	go install $(LDFLAGS) ./cmd/lore

clean:
	rm -f $(BINARY)

release:
	goreleaser release --clean

snapshot:
	goreleaser release --snapshot --clean --skip=publish
