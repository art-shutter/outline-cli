# Default recipe to run when no arguments are provided
default:
    @just --list

# Build with version (usage: just build-version v1.0.0), default is dev
build version="dev":
    go build -ldflags="-X main.Version={{version}} -s -w" -o build/outline-cli ./cmd/outline-cli

# Clean build artifacts
clean:
    rm -f outline-cli
    rm -rf build/

# Run tests
test:
	go test -v ./...

# Run mod tidy and mod verify
mod:
    go mod tidy
    go mod verify

# Format code
fmt:
    go fmt ./...

# Run linter
lint:
    golangci-lint run ./...

# Build and test
all: clean lint build test
