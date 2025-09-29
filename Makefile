# Makefile for rayo project

.PHONY: test fmt lint build clean dist dist-all release

# Build targets
BINARY_NAME=rayoc
BUILD_DIR=build

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS = -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

test:
	go test ./...

fmt:
	go fmt ./...

lint:
	golint ./...

build:
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/rayoc

# Cross-compilation targets
dist-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/rayoc

dist-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/rayoc

dist-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/rayoc

dist-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/rayoc

dist-windows-amd64:
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/rayoc

dist-windows-arm64:
	GOOS=windows GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe ./cmd/rayoc

# Build all platforms
dist-all: dist-linux-amd64 dist-linux-arm64 dist-darwin-amd64 dist-darwin-arm64 dist-windows-amd64 dist-windows-arm64

# Create distribution packages
dist: build
	mkdir -p $(BUILD_DIR)/dist
	cp $(BUILD_DIR)/$(BINARY_NAME) $(BUILD_DIR)/dist/
	cp README.md $(BUILD_DIR)/dist/
	cp LICENSE $(BUILD_DIR)/dist/ 2>/dev/null || true
	cd $(BUILD_DIR) && tar -czf $(BINARY_NAME)-$(shell uname -s | tr '[:upper:]' '[:lower:]')-$(shell uname -m).tar.gz dist/

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Development helpers
test-golden:
	go test -run TestGolden ./internal/testutil

install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

# Release helper
release: clean dist-all
	@echo "Release artifacts created in $(BUILD_DIR)/"
	@ls -la $(BUILD_DIR)/*rayoc*
