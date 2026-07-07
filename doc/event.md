# Event 模块设计文档

Event 模块是系统的事件总线与解耦中心，用于支持业务组件（如记忆进化、日志收集等非主要流程行为）之间的异步事件发布与订阅，从而实现系统各层级之间的低耦合交互。

---

## 1. 架构定位

在系统的三层 DDD 架构中，Event 模块跨越领域层与基础设施层：
- **领域层（Domain Layer）**：定义在 [interface.go](../domain/event/interface.go)，规定了领域事件 `Event`、事件处理器 `Handler`（以及函数式适配器 `HandlerFunc`）以及事件发布订阅和生命周期管理的 `Publisher`、`Subscriber`、`EventBus` 契约。
- **基础设施层（Infrastructure Layer）**：定义在 [memory.go](../infra/event/memory.go)，提供了基于内存 Channel 的异步并发安全事件总线实现 `InMemoryEventBus`。
- **职责划分**：领域层只声明事件的流转契约，不感知底层的传输媒介与缓冲控制逻辑；基础设施层负责并发协程的调度、背压机制（队列限制）、优雅退出以及 Panic 的安全性捕获，不参与任何具体的业务决策。

---

## 2. 核心组件与实体

### 2.1 领域事件：[Event](../domain/event/interface.go)
代表发生过的业务事实。任何实现该接口的对象均可被发布到总线中：
```go
type Event interface {
	ID() string            // 事件唯一标识
	Name() string          // 事件名称（即 Topic）
	OccurredAt() time.Time // 事件发生时间
	Payload() any          // 事件携带的 Payload 数据
}
```

### 2.2 事件处理器：[Handler](../domain/event/interface.go) 与 [HandlerFunc](../domain/event/interface.go)
用于处理特定 Topic 事件的消费者契约：
- `Handler` 接口：只包含一个 `Handle(ctx context.Context, ev Event) error` 方法。
- `HandlerFunc` 适配类型：支持直接将普通的 Go 函数转换为 `Handler` 接口以简化订阅声明。

### 2.3 订阅与发布接口：[EventBus](../domain/event/interface.go)
融合了发布、订阅与生命周期的统一总线：
- `Publisher`：提供异步/同步发布能力（`Publish(ctx, event)`）。
- `Subscriber`：注册特定的 Topic 处理器（`Subscribe(name, handler)`）。
- `EventBus`：组合了上述两接口，并带有 `Shutdown(ctx) error` 优雅退出方法。

---

## 3. 具体实现与适配

### 基于内存的事件总线：[InMemoryEventBus](../infra/event/memory.go)

- **后台工作协程池**：采用协程池模式（Worker Loop），在构建 `InMemoryEventBus` 时会通过 `workerNum` 参数指定启动若干个后台消费协程，持续从底层的 `taskChan` Channel 中读取任务并并发分发。
- **订阅并发控制**：在 `Subscribe` 和分发事件（`dispatchEvent`）时，采用 `sync.RWMutex` 读写锁来保证对订阅关系字典 `subscribers` 的读写并发安全。为了减少锁竞争，分发时会先将处理器拷贝一份副本（Deep Copy Slice），随后在无锁状态下遍历执行具体处理器。
- **优雅关闭（Shutdown）**：
  1. 将内部关闭状态置为 `closed = true`，后续的 `Publish` 与 `Subscribe` 均直接返回 `ErrEventBusClosed` 报错。
  2. 关闭底层的 `taskChan`，使所有 Worker 在消费完 Channel 积压的消息后自然退出循环。
  3. 通过 `sync.WaitGroup` 等待所有 Worker 完全退出，并结合 `context.Context` 提供超时强制退出控制。
- **防御性 Panic 拦截与错误隔离**：分发执行处理器时包裹了 `recover`。当单个订阅处理器的执行发生 Panic，或者返回 `error` 时，会被事件总线自动捕获并记录日志，从而确保事件总线内部协程池的安全，且不同的事件处理器之间互不干扰。
- **背压控制（Queue Limitation）**：底层 Channel 具备有限容量 `bufferSize`。若瞬时写入速度过快导致积压达到上限，`Publish` 会采用 `select` 语法防阻塞，并立即向调用方返回错误 `ErrQueueFull`，提供快速失败能力。

---

## 4. 测试与 Mock

### 4.1 Mock 生成与使用

在接口定义文件 [interface.go](../domain/event/interface.go) 头部声明了：
```go
//go:generate go run github.com/golang/mock/mockgen -source=interface.go -destination=./mock/event_mock.go -package=mock
```
- **生成方式**：在根目录下执行 `go generate ./...` 将会在 [mock](../domain/event/mock/) 子包下自动产出 Mock 结构。

### 4.2 单元测试

- **测试文件**：[memory_test.go](../infra/event/memory_test.go)
- **规避循环依赖**：测试包采用外部测试声明 `package event_test`，避免了对 `event` 的循环导入。
- **覆盖特性**：
  - `TestInMemoryEventBus_PublishSubscribe` 验证常规的订阅发布以及 Payload 的正确传输。
  - `TestInMemoryEventBus_PanicRecovery` 验证处理器发生 Panic 时总线是否能防崩溃，且不影响其他处理器正常分发。
  - `TestInMemoryEventBus_Shutdown` 验证调用优雅关闭后，积压任务是否可以被正常处理完，以及关闭状态下的并发拦截机制。
  - `TestInMemoryEventBus_QueueFull` 验证缓冲区满时能正确且迅速地返回 `ErrQueueFull` 错误，避免上游协程被无限阻塞。
