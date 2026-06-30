NIX_CONFIG ?= extra-experimental-features = nix-command flakes
export NIX_CONFIG

NIX_DEV_SHELL ?= path:.
NIX_DEVELOP = nix develop $(NIX_DEV_SHELL) --command

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
VERSION_PACKAGE = github.com/sund3RRR/ion/internal/version
LDFLAGS = -X $(VERSION_PACKAGE).Version=$(VERSION) -X $(VERSION_PACKAGE).Commit=$(COMMIT) -X $(VERSION_PACKAGE).Date=$(DATE)

.PHONY: generate generate-proto generate-sql test lint check build build-static build-dynamic build-dev clean

generate: generate-proto generate-sql

generate-proto:
	$(NIX_DEVELOP) buf generate

generate-sql:
	$(NIX_DEVELOP) sqlc generate -f pkg/ion/store/sqlc.yaml

test:
	$(NIX_DEVELOP) go test -v -cover -race -count=1 ./...

lint:
	$(NIX_DEVELOP) golangci-lint run

check: generate test lint

build:
	nix build path:.

build-static:
	nix build path:.#static

build-dynamic:
	nix build path:.#dynamic

build-dev:
	$(NIX_DEVELOP) mkdir -p bin
	$(NIX_DEVELOP) go build -ldflags "$(LDFLAGS)" -o bin/ion ./cmd/ion

clean:
	rm -rf bin result result-*
