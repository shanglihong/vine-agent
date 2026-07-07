# 变量定义
BINARY_CLI    := vine-agent
BINARY_SERVER := vine-server
CMD_CLI       := ./cmd/vine-agent
CMD_SERVER    := ./cmd/server

# 数据库存储路径配置（动态从 config.yaml 解析，未配置时默认使用 ~/.vine-agent/db/memory.db）
DB_PATH := $(shell grep -E 'sqlite_db_path:' config.yaml 2>/dev/null | awk '{print $$2}' | tr -d '"' | tr -d "'" | tr -d ' ')
ifeq ($(DB_PATH),)
  DB_PATH := data/db/memory.db
endif
# 将 ~ 或相对路径通过 shell 展开为绝对路径，方便 Makefile 进行 -f 检测与 mkdir
DB_PATH := $(shell eval echo '$(DB_PATH)')
DB_DIR := $(shell dirname '$(DB_PATH)' 2>/dev/null)
# DB_PATH := ./data/db/memory.db
# DB_DIR := ./data/db

.PHONY: help build run test lint tidy init init-force install-frontend run-frontend dev install check-env

# 默认目标：显示帮助信息
help:
	@echo "可用命令列表:"
	@echo "  make check-env         - 检查本地开发环境依赖 (Go, Node, npm, SQLite3) 并检验 Go 版本"
	@echo "  make install           - 一键安装后端 Go 依赖及前端 Node 依赖"
	@echo "  make build             - 编译命令行客户端"
	@echo "  make run               - 运行命令行客户端"
	@echo "  make run-server        - 启动后端 API 服务"
	@echo "  make test              - 运行 Go 后端单元测试"
	@echo "  make lint              - 运行 golangci-lint 静态检查"
	@echo "  make tidy              - 整理 Go modules 依赖"
	@echo "  make init              - 初始化 SQLite 数据库及目录 (若已存在则跳过)"
	@echo "  make init-force        - 强制重新初始化 SQLite 数据库 (覆盖已有数据)"
	@echo "  make install-frontend  - 安装前端依赖包"
	@echo "  make run-frontend      - 启动前端 Vite 开发服务器"
	@echo "  make dev               - 同时启动后端和前端开发环境"

# --- 环境依赖检测 ---
REQUIRED_TOOLS := go node npm sqlite3

check-env:
	@echo "🔍 正在检查系统依赖环境..."
	@MISSING_TOOLS=""; \
	for tool in $(REQUIRED_TOOLS); do \
		if ! command -v $$tool >/dev/null 2>&1; then \
			MISSING_TOOLS="$$MISSING_TOOLS $$tool"; \
		fi; \
	done; \
	if [ -n "$$MISSING_TOOLS" ]; then \
		echo "❌ 缺少以下必备工具:$$MISSING_TOOLS"; \
		echo "💡 请先手动安装缺少的工具后重试。"; \
		echo ""; \
		exit 1; \
	fi; \
	CURR_RAW=$$(go version | awk '{print $$3}' | sed 's/go//'); \
	CURR_VER=$$(echo "$$CURR_RAW" | awk -F. '{printf "%d.%d.%d", $$1, $$2, $$3}'); \
	REQ_RAW=$$(grep -E '^go ' go.mod | awk '{print $$2}'); \
	REQ_VER=$$(echo "$$REQ_RAW" | awk -F. '{printf "%d.%d.%d", $$1, $$2, $$3}'); \
	CURR_VAL=$$(echo "$$CURR_VER" | awk -F. '{print $$1*10000 + $$2*100 + $$3}'); \
	REQ_VAL=$$(echo "$$REQ_VER" | awk -F. '{print $$1*10000 + $$2*100 + $$3}'); \
	if [ $$CURR_VAL -lt $$REQ_VAL ]; then \
		echo "❌ Go 版本过低！当前版本: go$$CURR_RAW, 项目 go.mod 要求最低版本: go$$REQ_RAW"; \
		echo "💡 请升级您的 Go 编译器以确保兼容性。"; \
		echo ""; \
		exit 1; \
	fi; \
	echo "✅ 所有基础依赖检查通过 (Go >= $$REQ_RAW, Node, npm, SQLite3)."

# --- Go 后端命令 ---

build: check-env
	go build -o bin/$(BINARY_CLI) $(CMD_CLI)

run: check-env
	go run $(CMD_CLI)

run-server: check-env
	go run $(CMD_SERVER)

test: check-env
	go test ./...

lint: check-env
	golangci-lint run ./...

tidy: check-env
	go mod tidy

# --- 数据库初始化 ---

init: check-env
	@mkdir -p $(DB_DIR)
	@if [ -f $(DB_PATH) ]; then \
		echo "ℹ️ 数据库文件 $(DB_PATH) 已存在，跳过初始化。"; \
		echo "💡 如果您确需重新初始化（这会清除所有已有数据！），请运行: make init-force"; \
	else \
		echo "正在初始化 SQLite 数据库 $(DB_PATH)..."; \
		sqlite3 $(DB_PATH) < scripts/sqlite_memory.sql; \
		echo "✅ 数据库初始化成功。"; \
	fi

init-force: check-env
	@echo "⚠️ 正在强行重新初始化数据库..."
	@mkdir -p $(DB_DIR)
	@rm -f $(DB_PATH)
	@sqlite3 $(DB_PATH) < scripts/sqlite_memory.sql
	@echo "✅ 数据库重新初始化成功。"

# --- 前端命令 ---

install-frontend: check-env
	cd frontend && npm install

run-frontend: check-env
	cd frontend && npm run dev

# --- 一键依赖安装 ---

install: check-env tidy install-frontend

# --- 本地联调一键启动 ---

dev: check-env
	@echo "正在同时启动后端与前端服务 (按 Ctrl+C 退出)..."
	@make -j2 run-server run-frontend