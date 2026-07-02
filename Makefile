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
# 初始化数据库前置工作：创建数据目录，并应用独立 DDL 脚本
db-init:
	@echo "Initializing SQLite database at data/memory..."
	@mkdir -p data
	@sqlite3 data/memory < scripts/sqlite_memory.sql
	@echo "Database initialized successfully."