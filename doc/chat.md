# Chat 模块系统设计与架构

## 1. 架构定位
`domain/chat` 模块位于系统的**领域层（Domain Layer）**。
其核心职责是：
- 统一定义大语言模型（LLM）对话的抽象契约接口。
- 为上层（应用层等）提供屏蔽具体模型供应商的通用对话调用能力。
- 支持流式（Stream）与非流式（Generate）响应生成，并提供灵活的参数配置机制（Options 模式）。

## 2. 核心组件与实体
模块内的核心接口和结构体定义在 [chat.go](../domain/chat/chat.go) 中：

- **`ChatModel` 接口**：核心对话模型契约。
  - `Generate(ctx, messages, opts...)`：单次（非流式）回复生成。
  - `Stream(ctx, messages, opts...)`：增量（流式）回复生成。
- **`StreamReader` 接口**：流式消息读取器。
  - `Recv()`：阻塞式增量读取消息片段，直至返回 `io.EOF`。
  - `Close()`：关闭底层连接并释放 IO 资源。
- **`Option` 与 `OptionFunc`**：采用 Options 模式，支持动态配置请求参数。
  - 支持配置：`Model`（覆盖默认模型）、`Temperature`（采样温度）、`MaxTokens`（最大 Token 数）、`Tools`（工具声明）、`ToolChoice`（工具选择行为）。

**依赖关系**：
- 依赖 [vine-agent/domain/message](../domain/message) 中的 `Message` 实体结构。
- 依赖 [vine-agent/domain/tool](../domain/tool) 中的 `Tool` 接口定义。

## 3. 具体实现与适配
模型适配器实现位于子目录 [chat_model](../domain/chat/chat_model) 中：

- **`deepSeekAdapter`（私有）**：
  - 实现了 `ChatModel` 接口，充当适配器将 `infra/client/deepseek` 包装为领域层通用接口。
  - 对外暴露 `NewDeepSeekAdapter(client *deepseek.Client)` 进行依赖装配。
  - 在内部将 `message.Message` 及 `tool.Tool` 转换为 `deepseek.Message` 及 `deepseek.Tool` 等传输对象（DTO）。
- **`deepseekStreamReaderAdapter`（私有）**：
  - 实现了 `StreamReader` 接口，用于适配 DeepSeek 的流式客户端，在 `Recv` 中转换增量片段（Delta）并返回统一的 `message.Message`。

## 4. 测试与 Mock

- **单元测试**：
  - 测试文件：[deepseek_test.go](../domain/chat/chat_model/deepseek_test.go)。
  - **规避测试循环导入**：测试文件声明为外部测试包 `package chat_model_test`，避免了对 `chat_model` 的循环导入。
  - **Mock 机制**：利用自定义的 `mockRoundTripper` 劫持 `http.Client`，完全在内存中模拟 DeepSeek API 响应与 SSE 流式数据，确保测试稳定、不依赖外网环境。
- **Mock 代码生成**：
  - 在 [chat.go](../domain/chat/chat.go) 头部声明了：
    ```go
    //go:generate go run github.com/golang/mock/mockgen -source=chat.go -destination=./mock/chat_mock.go -package=mock
    ```
  - 支持在项目根目录运行 `go generate ./...` 一键生成 Mock 产物，并将 Mock 结构体生成在子包 `mock` 中，避免对外部工具的强依赖。
