BIN := whoop-cli

.PHONY: build test install release setup

build:
	go build -o ./dist/$(BIN) ./cmd/whoop-cli

test:
	go test ./...

install:
	go install ./cmd/whoop-cli

release:
	goreleaser release --clean

setup:
	go run ./cmd/whoop-cli setup
