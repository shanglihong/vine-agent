# Vine-Agent 智能体系统

Vine-Agent 是一个基于 Go 语言、DDD（领域驱动设计）架构与 React 前端构建的智能体系统。系统具备长期记忆演进能力，能自动在后台分析对话并更新用户的个人偏好与事实画像，同时支持敏感工具调用的拦截与人工确认。

---

## 快速开始

在开始运行前，请先准备好本地的 Go (>= 1.25.0)、Node.js (npm)、SQLite3 环境以及大模型 API 密钥（`DEEPSEEK_API_KEY`）。

### 1. 检查开发依赖
```bash
make check-env
```

### 2. 一键安装依赖 (Go & Node)
```bash
make install
```

### 3. 初始化 SQLite 数据库
```bash
# 默认初始化生产配置环境的数据库 (~/.vine-agent/data/db/memory.db)
make init

# 若需要初始化本地开发环境的数据库 (./data/db/memory.db)
make init ENV=dev
```

### 4. 运行服务

#### 一键启动项目 (推荐)
一键并发启动后端服务与前端开发服务器：
* **默认非开发模式启动** (数据写入 ~/.vine-agent/ 存储)：
  ```bash
  make run
  # 或者
  make dev
  ```
* **本地开发模式启动** (数据使用本地根目录 ./data/ 存储)：
  ```bash
  make run ENV=dev
  # 或者
  make dev ENV=dev
  ```
启动后，直接访问前端控制台：`http://localhost:5173/`。

#### 分步手动启动
* **运行后端服务**：
  ```bash
  # 默认生产/部署模式启动 (数据写入 ~/.vine-agent/ 存储)
  make run-server
  
  # 本地开发模式启动 (数据使用本地根目录 ./data/ 存储)
  make run-server ENV=dev
  ```
* **运行前端 Vite 服务器**：
  ```bash
  make run-frontend
  ```

---

## 命令行演示客户端

项目提供了一个纯命令行演示程序，用于单机调试记忆演化的核心逻辑：
```bash
# 编译演示客户端
make build

# 运行演示客户端
make run-cli
```

---

## 开发辅助命令

* **整理 Go modules 依赖**：`make tidy`
* **运行后端单元测试**：`make test`
* **运行静态代码检查**：`make lint`
* **强制重新初始化数据库** (警告：这会清空已有数据)：`make init-force`

---

## 设计文档

关于各模块的 DDD 架构定位与核心实体设计，可以参阅以下设计文档：

* **系统文档总览**：[.agents/rules/doc-index.md](.agents/rules/doc-index.md)
* **对话大模型适配 (Chat)**：[doc/chat.md](doc/chat.md)
* **事件总线 (Event)**：[doc/event.md](doc/event.md)
* **会话与长期记忆画像 (Memory)**：[doc/memory.md](doc/memory.md)
* **消息流实体 (Message)**：[doc/message.md](doc/message.md)
* **工具执行规范 (Tool)**：[doc/tool.md](doc/tool.md)
* **用户信息域 (User)**：[doc/user.md](doc/user.md)


## TODO
Plan(./plan.md)