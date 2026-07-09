# Tool 模块设计文档

Tool 模块是智能体（Agent）扩展能力的声明与执行中心，负责统一智能体外部工具调用的元数据契约定义，并提供通用的参数规范验证逻辑，使大语言模型能够准确地感知、抉择并执行特定的本地及网络接口。

---

## 1. 架构定位

在系统的三层 DDD 架构中，Tool 模块划分为：
- **领域层（Domain Layer）**：定义在 [tool.go](../domain/tool/tool.go)，规定了工具抽象接口 `Tool`、元数据 `Definition` 结构体，以及通用的参数校验器 `ValidateArguments`。
- **基础设施层（Infrastructure Layer）**：定义在 [infra/tools/](../infra/tools/) 目录下，提供了各种具体工具（如网页检索 `WebSearchTool`、网页爬取 `WebCrawlTool`）的具体物理执行逻辑。
- **职责划分**：领域层专注于工具的接口抽象规范与基于 JSON Schema 的入参强约束校验；基础设施层则处理物理调用逻辑（如网络调用、解析 JSON、静态数据返回），不感知智能体的编排调度。

---

## 2. 核心组件与实体

### 2.1 工具执行契约：[Tool](../domain/tool/tool.go)
大模型工具的核心契约接口：
```go
type Tool interface {
	// Info 返回工具定义元数据
	Info() Definition
	// Execute 执行具体业务，传入 JSON 字符串参数，返回结果
	Execute(ctx context.Context, args string) (string, error)
}
```

### 2.2 工具定义元数据：[Definition](../domain/tool/tool.go)
描述工具本身及其入参规范，以便序列化后提供给 LLM 识读：
- `Name`：工具的全局唯一命名（例如 `get_weather`）。
- `Description`：工具功能职责描述，LLM 依靠该描述判断在何种语境下进行调用。
- `Parameters`：入参强约束定义，通常是一个符合 JSON Schema 规范的 `map[string]any`，描述参数类型、字段及必要性。
- `RequiresConfirmation`：标记该工具是否属于高危/敏感操作（如删除用户、清空画像等），决定上层是否需要触发人工确认阻断流。

### 2.3 参数强验证：`ValidateArguments`
公用的静态辅助验证函数：
- 基于 `github.com/xeipuuv/gojsonschema` 库。
- 在工具执行前（或在基础设施的具体实现里），用于检验大模型传回的 JSON 格式 `Arguments` 字符串是否完美契合 Definition 中 Parameters 所描述的 JSON Schema。如果校验失败，立即提前抛错拒绝执行。

---

## 3. 具体实现与适配

物理工具的实现存放在基础设施包 [infra/tools/](../infra/tools/) 中：

### 3.1 网页检索工具：[WebSearchTool](../infra/tools/web_search.go)
- **名称**：`web_search`。
- **参数**：包含 `query` 必填参数（`string`，搜索关键字）。
- **逻辑**：无人工确认阻断。用于在互联网中检索与 `query` 相关的最新网页资讯，解析并返回结构化搜索结果。

### 3.2 网页爬取工具：[WebCrawlTool](../infra/tools/web_crawl.go)
- **名称**：`fetch_webpage`。
- **参数**：包含 `url` 必填参数（`string`，要爬取的网页地址）。
- **逻辑**：无人工确认阻断。访问指定的 `url`，提取网页的纯文本内容，过滤掉 HTML 标签、脚本及样式，以供大模型阅读。

---

## 4. 测试与 Mock

### 4.1 接口与 Mock

- 由于 `Tool` 属于高度内聚且自包含的具体执行算子，其接口调用直接以切片列表（`[]tool.Tool`）的形式装配进 API Handler 或智能体服务中，且测试时通常直接传入具体的 Mock 适配器模型（如 `mockChatModel`）来测试其分支，因此当前并未对 `Tool` 接口使用 mockgen 额外生成单元 Mock 类。

### 4.2 单元测试与验证

- **验证手段**：
  - 工具的正确装配与参数提取已在 [main.go](../cmd/server/main.go) 中实例化并注入到 API 路由层中。
  - 在接口调用链的整体集成测试、以及在 `mockChatModel` 中模拟触发工具调用逻辑（如“搜索”、“网页”等关键词输入），验证了工具的物理执行以及 `ValidateArguments` 校验逻辑是否正常运转。
