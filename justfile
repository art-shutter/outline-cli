set shell := ["bash", "-euo", "pipefail", "-c"]
set dotenv-load := true
set positional-arguments := true

os_default := lowercase(shell("uname -s"))
arch_default := shell("uname -m")

default:
    @just defaults
    @echo ""
    @just --list

defaults:
    @echo "OS: {{ os_default }}"
    @echo "Arch: {{ arch_default }}"
    @echo "{{ shell("go version") }}"

build version="development" os=os_default arch=arch_default: defaults
    GOOS="{{ os }}" GOARCH="{{ arch }}" \
      go build -ldflags="-X main.Version={{version}} -s -w" \
      -trimpath -o "build/outline-cli-{{ os }}-{{ arch }}" ./cmd/outline-cli

build-all version="development":
    just build {{ version }} darwin arm64 \
      build {{ version }} darwin amd64 \
      build {{ version }} linux arm64 \
      build {{ version }} linux amd64

publish-assets release_id:
    gh release upload {{ release_id }} build/outline-cli-* --clobber

clean:
    rm -f outline-cli
    rm -rf build/

test:
    go test -v ./...

mod:
    go mod tidy
    go mod verify

fmt:
    go fmt ./...

lint:
    golangci-lint run ./...

all: clean lint build test
