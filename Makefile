BINARY     := mdv
BUILD_DIR  := dist
PKG        := github.com/devekkx/module-dependency-visualizer
CMD        := ./cmd/mdv

VERSION    ?= dev
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X '$(PKG)/internal/config.version=$(VERSION)' \
           -X '$(PKG)/internal/config.commit=$(COMMIT)'   \
           -X '$(PKG)/internal/config.buildDate=$(BUILD_DATE)'

.PHONY: all build install uninstall test test-race cover lint vet gosec clean tidy help

all: lint test build

## build: compile the binary to dist/mdv
build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) $(CMD)

## install: build and install mdv to $(GOPATH)/bin (or ~/go/bin)
install:
	go install -ldflags "$(LDFLAGS)" $(CMD)

## uninstall: remove mdv from $(GOPATH)/bin
uninstall:
	rm -f $(shell go env GOPATH)/bin/$(BINARY)

## test: run all tests
test:
	go test ./...

## test-race: run all tests with race detector
test-race:
	go test -race ./...

## cover: run tests and generate coverage report
cover:
	go test -race -coverprofile=coverage.out -coverpkg=./internal/... ./...
	go tool cover -func=coverage.out

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## vet: run go vet
vet:
	go vet ./...

## gosec: run gosec security scanner
gosec:
	gosec ./...

## tidy: tidy and verify go.mod
tidy:
	go mod tidy
	go mod verify

## clean: remove build artifacts
clean:
	rm -rf $(BUILD_DIR) coverage.out

## help: show this help
help:
	@grep -E '^## ' Makefile | sed 's/^## //'
