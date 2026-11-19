BINARY  := sablier
PKG     := github.com/espcaa/sablier
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X $(PKG)/internal/version.Version=$(VERSION) \
	-X $(PKG)/internal/version.Commit=$(COMMIT) \
	-X $(PKG)/internal/version.Date=$(DATE)

.PHONY: all build run install test lint docs generate tidy clean server db

all: build

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/$(BINARY)
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY)-server ./cmd/$(BINARY)-server

# local postgres on 127.0.0.1:5433
db:
	docker compose up -d --wait postgres

server: db
	go run ./cmd/$(BINARY)-server

run:
	go run ./cmd/$(BINARY) $(ARGS)

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/$(BINARY)

test:
	go test ./...

lint:
	go vet ./...
	@command -v golangci-lint >/dev/null && golangci-lint run || echo "golangci-lint not installed, skipped"

generate:
	sqlc generate

docs:
	go run ./tools/gendocs -o docs/reference

tidy:
	go mod tidy

clean:
	rm -rf bin/
