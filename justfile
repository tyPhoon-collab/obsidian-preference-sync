set shell := ["sh", "-cu"]

default:
    just --list

fmt:
    gofmt -w cmd internal

test:
    go test ./...

build:
    go build -o bin/obsidian-preference-sync ./cmd/obsidian-preference-sync

check: fmt test build
