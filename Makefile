BINARY   = lore
VERSION ?= dev
LDFLAGS  = -ldflags "-X github.com/nishchay/lore/internal/cli.Version=$(VERSION)"

.PHONY: build build-all test lint install install-all clean release snapshot

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/lore

build-all:
	go build $(LDFLAGS) -o lore ./cmd/lore
	go build $(LDFLAGS) -o glh  ./cmd/glh

test:
	go test ./...

lint:
	golangci-lint run ./...

install:
	go install $(LDFLAGS) ./cmd/lore

install-all:
	go install $(LDFLAGS) ./cmd/lore
	go install $(LDFLAGS) ./cmd/glh

clean:
	rm -f lore glh

release:
	goreleaser release --clean

snapshot:
	goreleaser release --snapshot --clean --skip=publish
