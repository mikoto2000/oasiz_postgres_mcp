BINARY := oasiz-postgres-mcp
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || date +%Y%m%d%H%M%S)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test clean

build:
	mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/oasiz-postgres-mcp

test:
	go test ./...

clean:
	rm -rf bin dist package
