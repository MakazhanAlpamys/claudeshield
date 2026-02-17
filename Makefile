.PHONY: build install test lint clean docker-sandbox run

BINARY_NAME=claudeshield
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X github.com/claudeshield/claudeshield/cmd/claudeshield/cmd.version=$(VERSION)"

# Build the binary
build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/claudeshield

# Install to $GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/claudeshield

# Run tests
test:
	go test -v -race ./...

# Run linter
lint:
	golangci-lint run ./...

# Build sandbox Docker image
docker-sandbox:
	docker build -t claudeshield/sandbox:latest ./docker/sandbox

# Run the TUI
run: build
	./bin/$(BINARY_NAME) ui

# Initialize a test project
init: build
	./bin/$(BINARY_NAME) init

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Download dependencies
deps:
	go mod download
	go mod tidy

# Cross-compile for all platforms
release:
	goreleaser release --clean

# Dev release (snapshot, no publish)
snapshot:
	goreleaser release --snapshot --clean
