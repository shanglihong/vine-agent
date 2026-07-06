# 变量定义
BINARY_CLI    := vine-agent
BINARY_SERVER := vine-server
CMD_CLI       := ./cmd/vine-agent
CMD_SERVER    := ./cmd/server

.PHONY: help build run test lint tidy init install-frontend run-frontend dev install

# 默认目标：显示帮助信息
help:
	@echo "可用命令列表:"
	@echo "  make install           - 一键安装后端 Go 依赖及前端 Node 依赖"
	@echo "  make build             - 编译命令行客户端"
	@echo "  make run               - 运行命令行客户端"
	@echo "  make run-server        - 启动后端 API 服务"
	@echo "  make test              - 运行 Go 后端单元测试"
	@echo "  make lint              - 运行 golangci-lint 静态检查"
	@echo "  make tidy              - 整理 Go modules 依赖"
	@echo "  make init              - 初始化 SQLite 数据库及目录"
	@echo "  make install-frontend  - 安装前端依赖包"
	@echo "  make run-frontend      - 启动前端 Vite 开发服务器"
	@echo "  make dev               - 同时启动后端和前端开发环境"

# --- Go 后端命令 ---

build:
	go build -o bin/$(BINARY_CLI) $(CMD_CLI)

run:
	go run $(CMD_CLI)

run-server:
	go run $(CMD_SERVER)

test:
	go test ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

# --- 数据库初始化 ---

init:
	@echo "Initializing SQLite database at data/db/memory.db..."
	@mkdir -p data/db
	@sqlite3 data/db/memory.db < scripts/sqlite_memory.sql
	@echo "Database initialized successfully."

# --- 前端命令 ---

install-frontend:
	cd frontend && npm install

run-frontend:
	cd frontend && npm run dev

# --- 一键依赖安装 ---

install: tidy install-frontend

# --- 本地联调一键启动 ---

dev:
	@echo "正在同时启动后端与前端服务 (按 Ctrl+C 退出)..."
	@make -j2 run-server run-frontend