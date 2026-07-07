# Message 模块设计文档

Message 模块是系统的对话实体与数据载体，定义了智能体对话过程中各阶段消息（常规消息、流式增量消息、工具调用、工具返回等）的标准实体结构，是系统各组件和网络传输的数据通用契约。

---

## 1. 架构定位

在系统的三层 DDD 架构中，Message 模块属于**领域层（Domain Layer）**。
- **职责**：
  - 定义了不可变的对话消息结构体及相关行为。
  - 为整个系统（如应用服务、持久化存储、大模型适配器、网络接口层）提供统一的值对象与实体标准。
- **无依赖性**：作为基础的数据契约，它只依赖 Go 语言的标准库，**严禁**依赖任何基础设施（数据库、第三方 LLM SDK、网络框架等）或其它业务领域包。

---

## 2. 核心组件与实体

模块内核心数据结构分布在两个文件中：

### 2.1 对话消息实体：[message.go](../domain/message/message.go)

- **角色枚举：`Role`**：
  - `RoleSystem` ("system")：系统级提示词，设定智能体人设与规则约束。
  - `RoleUser` ("user")：用户输入的对话文本。
  - `RoleAssistant` ("assistant")：智能体生成的回复或工具调用指令。
  - `RoleTool` ("tool")：工具执行完毕后回传的执行结果数据。
- **消息结构体：`Message`**：
  - `Role`：该条消息的发起者角色。
  - `Content`：消息主体文本内容。
  - `ReasoningContent`：推理思维链内容（主要由 DeepSeek R1 思考阶段产生）。
  - `ToolCalls`：助理消息中携带的工具调用指令数组。
  - `ToolCallID`：当角色为 `tool` 时，此字段标示对应的那次工具调用，用作结果的绑定配对。
- **工具调用详情：`ToolCall` 与 `FunctionCall`**：
  - 规定了工具的 `ID`、`Type` (固定为 "function") 以及调用的具体 `Function`（函数名 `Name` 与 JSON 编码的入参 `Arguments`）。

##### 内聚行为方法
- `IsAssistant()` / `IsToolCall()` / `HasToolCalls()`：封装对消息角色的判断，提供自解释的聚合状态查验。
- `Merge(other *ToolCall)`：支持在流式接收大模型片段时，对多路工具调用的参数进行累加拼接。
- `MergeStreamToolCall(tcs, delta)`：流式合并辅助函数，根据流分片索引（Index）将分片动态拼装回 `ToolCall` 列表中。

### 2.2 流式传输契约：[stream.go](../domain/message/stream.go)

- **流消息类型：`StreamMessageType`**：
  - `StreamMessageTextDelta` ("text_delta")：常规文本生成的增量片段。
  - `StreamMessageReasoningDelta` ("reasoning_delta")：模型思考链的增量片段。
  - `StreamMessageToolCall` ("tool_call")：工具调用的增量通知。
  - `StreamMessageToolResult` ("tool_result")：工具执行完毕的异步结果通知。
- **流消息载体：`StreamMessage`**：
  - 作为流式 API 统一向前端推送的结构体，包裹了 `Type` 以及可选的文本 `Content`、工具调用 `ToolCall` 或执行结果 `StreamToolResult`。
- **流消息读取器接口：`StreamMessageReader`**：
  - `Recv() (*StreamMessage, error)`：阻塞式读取增量片段，若流结束应返回 `io.EOF`。
  - `Close() error`：释放底层连接及 IO 资源。

---

## 3. 具体实现与适配

### 流消息拼装器：`ReadAndAssembleMessage`

在 [stream.go](../domain/message/stream.go) 中提供了一个核心的高内聚工具函数：
- **职责**：接收一个 `StreamMessageReader`，循环调用 `Recv` 提取所有的流片段。
- **拼装逻辑**：
  - 自动累加拼接 `StreamMessageTextDelta` 的 `Content`；
  - 自动累加拼接 `StreamMessageReasoningDelta` 的 `Content`；
  - 调用 `MergeStreamToolCall` 实时汇聚多个并发的工具调用参数；
  - 支持传入 `callback func(*StreamMessage)`，在接收到分片时实时向调用者回调，方便前端实时推送或事件总线广播。
- **产出**：当读取到 `io.EOF` 时，将这些分片融合成一个最终且完整的 `RoleAssistant` 类型的 `Message` 实体返回。

---

## 4. 测试与 Mock

### 4.1 Mock 说明

- 由于 `Message` 和 `StreamMessage` 是纯数据实体和通用值对象，且其接口 `StreamMessageReader` 的实现非常轻量（如 `mockStreamReader`、`deepseekStreamReaderAdapter`），因此没有在 Message 模块内部使用 `gomock` 自动生成 Mock 代码。

### 4.2 单元测试覆盖

- **覆盖策略**：
  - 针对流数据拼接（`Merge`、`MergeStreamToolCall`）和 `ReadAndAssembleMessage` 的稳定性测试，已在以下外部测试中得到了充分的覆盖：
    - `infra/client/deepseek` 及大模型适配器测试 [deepseek_test.go](../domain/chat/chat_model/deepseek_test.go)（通过内存流模拟对这些函数的集成测试）。
    - 智能体服务层与交互层应用测试。
- 由于是无副作用的纯内存对象操作，逻辑完全是确定性的，在业务的系统集成与模型驱动层进行了隐式验证。
