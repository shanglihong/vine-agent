BINARY   := vine-agent
CMD_PATH := ./cmd/vine-agent

.PHONY: build run test lint tidy

build:
	go build -o bin/$(BINARY) $(CMD_PATH)

run:
	go run $(CMD_PATH)

test:
	go test ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy
