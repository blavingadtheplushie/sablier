BINARY  := sablier
PKG     := github.com/espcaa/sablier
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X $(PKG)/internal/version.Version=$(VERSION) \
	-X $(PKG)/internal/version.Commit=$(COMMIT) \
	-X $(PKG)/internal/version.Date=$(DATE)

.PHONY: all build run install test test-server lint docs generate tidy clean server db dev website

all: build

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/$(BINARY)
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY)-server ./cmd/$(BINARY)-server

# local postgres on 127.0.0.1:5433
db:
	docker compose up -d --wait postgres

server: db website
	go run ./cmd/$(BINARY)-server

dev: db
	@trap 'kill 0' EXIT; \
	(cd website && npx astro dev --port 4321) & \
	SABLIER_SITE_PROXY=http://localhost:4321 air

website:
	cd website && npm run build

# server tests against a throwaway sablier_test database
test-server: db
	-docker exec sablier_db createdb -U sablier sablier_test 2>/dev/null
	SABLIER_TEST_DATABASE_URL=postgres://sablier:sablier@localhost:5433/sablier_test?sslmode=disable go test -count=1 ./internal/server/...

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
	templ generate

docs:
	go run ./tools/gendocs

tidy:
	go mod tidy

clean:
	rm -rf bin/
