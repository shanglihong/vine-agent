# Memory 模块设计文档

Memory 模块是系统的短期记忆管理中心，主要负责 AI 对话会话（Session）的生命周期、状态、缓存与物理持久化管理，为智能体交互提供稳定、高效的记忆存取支持。

---

## 1. 架构定位

在系统的三层 DDD 架构中，Memory 模块的定位如下：
- **领域层（Domain Layer）**：定义了会话实体 `Session`、仓储契约 `SessionRepository` 接口、领域异常以及提供缓存加速与并发安全控制的领域服务 `SessionService` 接口。
- **基础设施层（Infrastructure Layer）**：实现具体的物理持久化适配器（如 SQLite 持久化），以及相关的并发安全连接管理。
- **职责划分**：领域服务及实体主导业务决策（如缓存失效策略、会话状态流转、数据深拷贝等）；基础设施层则保持无状态，只负责物理存储和事务保障，严禁做业务决策。

---

## 2. 核心组件与实体

### 2.1 实体/聚合根：[Session](../domain/memory/session/session.go)

`Session` 代表一个 AI 对话会话领域对象。它是 Memory 模块的聚合根，包含以下核心字段：
- `ID` (string): 会话唯一标识。
- `UserID` (string): 所属用户标识。
- `CreatedAt`/`UpdatedAt` (time.Time): 创建与更新时间。
- `Metadata` (map[string]string): 会话的元数据字典，用于保存状态和特定标记。
- `Messages` ([]message.Message): 会话内历史对话消息列表。

#### 内聚行为与状态管理
- **状态流转**：通过在 `Metadata` 中设置 `status` 键实现会话的状态流转：
  - `MarkPendingConfirmation()`: 将状态置为 `pending_confirmation`（等待敏感工具执行确认）。
  - `MarkInterruptedText()`: 将状态置为 `interrupted_text`（流式文本生成被中断）。
  - `ClearStatus()`: 清除状态。
  - `ApplyInterrupt(err)`: 解析领域异常 `InterruptError`，并根据其中的状态信息自动更新会话状态。
- **演进进度记录**：
  - `GetLastEvolvedMsgCount()`: 获取上次完成偏好演进时的消息总数。
  - `UpdateLastEvolvedMsgCount()`: 更新已演进的消息数记录为当前消息列表长度。

### 2.2 仓储契约：[SessionRepository](../domain/memory/session/interface.go)

定义了 Session 物理持久化的多态契约接口，使领域服务能与具体数据库技术解耦：
```go
type SessionRepository interface {
	Save(ctx context.Context, sess *Session) error
	Get(ctx context.Context, id string) (*Session, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, userID string) ([]*Session, error)
}
```

### 2.3 领域服务接口：[SessionService](../domain/memory/session/interface.go)

领域服务 `SessionService` 接口定义了统一的业务操作，其具体实现由包内私有的 `sessionService`（定义在 [session_service.go](../domain/memory/session/session_service.go)）来承担，主要职责包括：
1. **二级缓存加速**：内置并发安全的 `utils.TTLCache` 缓存，生存时间默认为 `5` 分钟，由服务内部管理。
2. **读写策略**：
   - `Get`：首先查询缓存，若未命中则穿透至底层 Repository 并写入缓存。
   - `Save` / `Delete`：在物理存储操作成功后，使缓存中的对应记录失效，确保数据一致性。
3. **并发安全与深拷贝**：`Get` 方法在返回 Session 时会调用内部的 `cloneSession` 进行深拷贝，防止外部直接修改缓存中的对象而引发竞态（Data Race）问题。

---

## 3. 具体实现与适配

基础设施层提供了基于 SQLite 的具体仓储实现：

### 3.1 SQLite 仓储：[SessionStore](../infra/persistence/sqlite/session.go)

- **连接管理**：采用 `sync.Once` 进行惰性初始化（通过 `getSessionDB()`）。在首次操作数据库时才建立物理连接并定位项目根目录下的 `data/memory` 数据库文件，避免进程启动时发生阻塞。
- **数据序列化**：由于 SQLite 无法直接存储 Go 复杂结构，在 `Save` 与 `Get` 时，历史消息列表 `Messages` 和元数据 `Metadata` 会自动序列化/反序列化为 JSON 文本存储在数据库中。
- **事务支持**：在 `Save` 操作中，采用 SQLite 独占事务（`BeginTx`、`Commit`、`Rollback`）保障持久化一致性，并使用 `ON CONFLICT(id) DO UPDATE` 机制以支持 Upsert 语义。
- **按需加载优化**：`List` 方法查询用户会话列表时采用精简 SQL（不查询 `messages` 字段），返回的 `Session` 列表中历史消息为 `nil`，从而避免拉取列表时产生不必要的 I/O 损耗。

---

## 4. 测试与 Mock

为保证代码质量及架构层级的单向依赖，模块中采用了以下测试规约：

### 4.1 Mock 生成与使用
在仓储接口定义文件上方声明了 `go:generate` 指令：
```go
//go:generate go run github.com/golang/mock/mockgen -source=interface.go -destination=./mock/session_mock.go -package=mock
```
- **生成命令**：在项目根目录运行 `go generate ./...` 即可一键生成 Mock。
- **生成位置**：Mock 产物严格存放在接口所在目录的子包 `./mock` 中，避免污染上层代码。

### 4.2 单元测试与脚手架

1. **领域服务单元测试**：位于 [session_service_test.go](../domain/memory/session/session_service_test.go)。
   - **规避循环导入**：测试文件被声明为外部测试包（`package session_test`），从而使测试代码不会产生循环依赖。
   - **行为测试**：使用 `gomock` 模拟 `Repository` 的行为，专门验证缓存的失效策略、读写穿透以及深拷贝的有效性。
2. **SQLite 仓储单元测试**：位于 [session_test.go](../infra/persistence/sqlite/session_test.go)。
   - **全局测试脚手架**：在 [main_test.go](../infra/persistence/sqlite/main_test.go) 中抽取了 `TestMain` 管理测试生命周期。测试执行时会自动连接 SQLite 内存数据库（`:memory:`）并通过加载 `scripts/sqlite_memory.sql` 文件进行 Schema 结构的模拟初始化。
   - **测试隔离**：每次子测试开始前均通过 `DELETE FROM sessions` 清空数据，保证测试之间的彻底隔离。
