BINARY  := sablier
PKG     := github.com/espcaa/sablier
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X $(PKG)/internal/version.Version=$(VERSION) \
	-X $(PKG)/internal/version.Commit=$(COMMIT) \
	-X $(PKG)/internal/version.Date=$(DATE)

.PHONY: all build run install test test-server lint docs generate tidy clean server db dev website release

all: build

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/$(BINARY)
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY)-server ./cmd/$(BINARY)-server

# every platform into dist/, named like install.sh and sablier update expect
PLATFORMS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64
release:
	rm -rf dist && mkdir -p dist
	for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; ext=; [ $$os = windows ] && ext=.exe; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" \
			-o dist/$(BINARY)_$${os}_$${arch}$$ext ./cmd/$(BINARY) || exit 1; \
	done
	cd dist && shasum -a 256 $(BINARY)_* > checksums.txt

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
