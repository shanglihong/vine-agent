# Memory 模块设计文档

Memory 模块是系统的记忆管理中心，负责管理 AI 对话会话（Session）的生命周期、状态、缓存与持久化，以及用户的长期偏好/事实画像（Profile）演化和历史对话消息的全文检索（Retrieval）。

---

## 1. 架构定位

在系统的三层 DDD 架构中，Memory 模块由三个子模块共同组成，定位如下：
- **领域层（Domain Layer）**：
  - **Session 子模块**：定义了会话实体 `Session`、仓储契约 `SessionRepository` 接口、领域异常以及提供缓存加速与并发安全控制的领域服务 `SessionService` 接口。
  - **Profile 子模块**：定义了用户长期记忆画像实体 `Profile`、仓储契约 `ProfileRepository` 接口、偏好与事实提取器契约 `Extractor` 接口以及进行画像演化管理的领域服务 `EvolutionService` 接口。
  - **Retrieval 子模块**：定义了全文检索模型 `SearchResult`、仓储契约 `RetrievalRepository` 接口以及消息检索领域服务 `RetrievalService` 接口。
- **基础设施层（Infrastructure Layer）**：
  - **Session & Retrieval 适配器**：提供基于 SQLite 的仓储实现，分别使用普通表持久化 Session 以及使用 FTS5 虚拟表（`messages_fts`）进行高效消息检索。
  - **Profile 适配器**：提供基于本地 Markdown 文件的持久化实现，将偏好和事实以 Markdown 列表格式分别保存。
- **职责划分**：领域服务及实体主导业务决策（如缓存失效、状态流转、画像演化合并、数据深拷贝等）；基础设施层则保持无状态，只负责物理存储 and 事务保障，严禁做业务决策。

---

## 2. 核心组件与实体

### 2.1 会话管理（Session）

#### 实体/聚合根：[Session](../domain/memory/session/session.go)
`Session` 代表一个 AI 对话会话领域对象。它是会话子模块的聚合根，包含以下核心字段：
- `ID` (string): 会话唯一标识。
- `UserID` (string): 所属用户标识。
- `CreatedAt`/`UpdatedAt` (time.Time): 创建与更新时间。
- `Metadata` (map[string]string): 会话的元数据字典，用于保存状态和特定标记。
- `Messages` ([]message.Message): 会话内历史对话消息列表。

##### 内聚行为与状态管理
- **状态流转**：通过在 `Metadata` 中设置 `status` 键实现会话的状态流转：
  - `MarkPendingConfirmation()`: 将状态置为 `pending_confirmation`（等待敏感工具执行确认）。
  - `MarkInterruptedText()`: 将状态置为 `interrupted_text`（流式文本生成被中断）。
  - `ClearStatus()`: 清除状态。
  - `ApplyInterrupt(err)`: 解析领域异常 `InterruptError`，并根据其中的状态信息自动更新会话状态。
- **演进进度记录**：
  - `GetLastEvolvedMsgCount()`: 获取上次完成偏好演进时的消息总数。
  - `UpdateLastEvolvedMsgCount()`: 更新已演进的消息数记录为当前消息列表长度。

#### 仓储契约：[SessionRepository](../domain/memory/session/interface.go)
定义了 Session 物理持久化的多态契约接口，使领域服务能与具体数据库技术解耦：
```go
type SessionRepository interface {
	Save(ctx context.Context, sess *Session) error
	Get(ctx context.Context, id string) (*Session, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, userID string) ([]*Session, error)
}
```

#### 领域服务接口：[SessionService](../domain/memory/session/interface.go)
领域服务 `SessionService` 接口定义了统一的业务操作，其具体实现由包内私有的 `sessionService`（定义在 [session_service.go](../domain/memory/session/session_service.go)）来承担，主要职责包括：
1. **二级缓存加速**：内置并发安全的 `utils.TTLCache` 缓存，生存时间默认为 `5` 分钟，由服务内部管理。
2. **读写策略**：
   - `Get`：首先查询缓存，若未命中则穿透至底层 Repository 并写入缓存。
   - `Save` / `Delete`：在物理存储操作成功后，使缓存中的对应记录失效，确保数据一致性。
3. **并发安全与深拷贝**：`Get` 方法在返回 Session 时会调用内部的 `cloneSession` 进行深拷贝，防止外部直接修改缓存中的对象而引发竞态（Data Race）问题。

---

### 2.2 长期记忆画像（Profile）

#### 实体/聚合根：[Profile](../domain/memory/profile/profile.go)
`Profile` 长期记忆画像聚合根用于整合并持有用户的长期偏好与事实。
- `UserID` (string): 用户唯一标识。
- `Preferences` ([]string): 偏好纯文本列表。
- `Facts` ([]string): 事实纯文本列表。
- `UpdatedAt` (time.Time): 更新时间。

##### 内聚行为
- `Update(newPrefs, newFacts)`: 接收新提取的偏好和事实列表。在赋值前，会自动通过辅助函数 `cleanItem` 对列表项进行防御性清洗（剥离前导/尾随空格及可能多余的 Markdown 列表前缀符号 `-` 或 `*`），最后更新 `UpdatedAt`。

#### 仓储契约：[ProfileRepository](../domain/memory/profile/interface.go)
定义了用户长期记忆画像持久化的契约：
```go
type ProfileRepository interface {
	GetByUserID(ctx context.Context, userID string) (*Profile, error)
	Save(ctx context.Context, prof *Profile) error
}
```

#### 提取器契约：[Extractor](../domain/memory/profile/interface.go)
定义了从对话中提取新画像项的通用契约，通常在大模型基础设施（如 DeepSeek、Gemini）中实现：
```go
type Extractor interface {
	Extract(ctx context.Context, messages []message.Message, existingPrefs []string, existingFacts []string) ([]string, []string, error)
}
```

#### 领域服务接口：[EvolutionService](../domain/memory/profile/interface.go)
领域服务 `EvolutionService` 负责编排记忆进化流。具体私有实现 `evolutionService` 定义在 [evolution_service.go](../domain/memory/profile/evolution_service.go)。其核心方法 `Evolve` 传入当前 `Profile` 实体和新消息：
1. 调用 `Extractor` 根据新消息及现有偏好/事实，提炼合成出最新的列表。
2. 触发 `Profile` 聚合根执行 `Update` 操作，完成实体层面的偏好事实更新。

---

### 2.3 消息全文检索（Retrieval）

#### 数据模型：[SearchResult](../domain/memory/retrieval/model.go)
跨会话检索时的结果封装，包含命中的 `Message` 实体以及所处的会话与用户上下文：
- `SessionID` (string): 消息所属会话 ID。
- `UserID` (string): 消息所属用户 ID.
- `Message` (message.Message): 具体的对话消息实体。

#### 仓储契约：[RetrievalRepository](../domain/memory/retrieval/interface.go)
定义了消息全文检索物理持久化的多态契约：
```go
type RetrievalRepository interface {
	Save(ctx context.Context, sessionID string, userID string, msg message.Message) error
	SearchSession(ctx context.Context, sessionID string, query string, limit int) ([]message.Message, error)
	SearchUser(ctx context.Context, userID string, query string, limit int) ([]SearchResult, error)
	DeleteBySession(ctx context.Context, sessionID string) error
}
```

#### 领域服务接口：[RetrievalService](../domain/memory/retrieval/interface.go)
领域服务 `RetrievalService` 接口定义了全文检索的业务操作契约，由私有的 `retrievalService`（定义在 [retrieval_service.go](../domain/memory/retrieval/retrieval_service.go)）实现。主要功能包括：
1. **入参验证**：在保存及查询前，对 SessionID、UserID 和 Query 进行非空验证，过滤空内容的记录。
2. **检索限制**：对最大返回数量（Limit）提供防御性保护（默认为 20）。

---

## 3. 具体实现与适配

### 3.1 会话的 SQLite 仓储：[SessionStore](../infra/persistence/sqlite/session.go)

- **连接管理**：采用 `sync.Once` 进行惰性初始化（通过 `getSessionDB()`）。在首次操作数据库时才建立物理连接并定位项目根目录下的 `data/memory` 数据库文件，避免进程启动时发生阻塞。
- **数据序列化**：由于 SQLite 无法直接存储 Go 复杂结构，在 `Save` 与 `Get` 时，历史消息列表 `Messages` 和元数据 `Metadata` 会自动序列化/反序列化为 JSON 文本存储在数据库中。
- **事务支持**：在 `Save` 操作中，采用 SQLite 独占事务（`BeginTx`、`Commit`、`Rollback`）保障持久化一致性，并使用 `ON CONFLICT(id) DO UPDATE` 机制以支持 Upsert 语义。
- **按需加载优化**：`List` 方法查询用户会话列表时采用精简 SQL（不查询 `messages` 字段），返回的 `Session` 列表中历史消息为 `nil`，从而避免拉取列表时产生不必要的 I/O 损耗。

### 3.2 长期画像的本地文件仓储：[fileProfileRepository](../infra/persistence/file/profile.go)

- **物理映射**：以本地文件夹作为存储目录，将用户的偏好和事实列表分别序列化为以 `- ` 符号开头的 Markdown 列表文件进行保存，对应路径为 `baseDir/<UserID>/preferences.md` 和 `baseDir/<UserID>/facts.md`。
- **读写逻辑**：
  - **Save**：在指定用户目录下生成或覆盖 `preferences.md` 与 `facts.md`。
  - **GetByUserID**：如果两个文件均不存在，则返回 `(nil, nil)`。如果存在，则利用 `bufio.Scanner` 按行读取，并对每一行去除前导的 `-`、`*` 或 `+` 列表符号，将其还原为纯文本切片。
- **并发控制**：内置 `sync.RWMutex` 读写锁，保护多协程同时并发读写用户画像文件的安全。

### 3.3 全文检索的 SQLite 仓储：[RetrievalStore](../infra/persistence/sqlite/retrieval.go)

- **全文索引虚拟表**：底层的 SQLite 存储使用全文检索虚拟表（`messages_fts`）存储各条消息及其对应的 Role，提供了强大的全文检索查询引擎。
- **SQL 匹配**：
  - 会话内检索（`SearchSession`）：通过 SQL `MATCH` 语法指定 FTS 匹配，在特定 `session_id` 范围内查找并过滤结果。
  - 跨会话检索（`SearchUser`）：在特定 `user_id` 下使用 `MATCH` 检索，并将匹配到的 `session_id` 组合成 `SearchResult` 返回，支持多会话上下文召回。

---

## 4. 测试与 Mock

### 4.1 Mock 生成与使用

在每个子领域包的接口定义文件（`interface.go`）上方声明了 `go:generate` 指令，如：
```go
//go:generate go run github.com/golang/mock/mockgen -source=interface.go -destination=./mock/session_mock.go -package=mock
```
- **生成命令**：在项目根目录运行 `go generate ./...` 即可一键生成所有的领域及仓储 Mock。
- **生成位置**：Mock 产物均存放在各自包内的 `./mock` 子目录中，从而避免污染主业务代码包。

### 4.2 单元测试与脚手架

#### 领域服务单元测试
- **会话管理测试**：位于 [session_service_test.go](../domain/memory/session/session_service_test.go)。使用 MockRepository 验证缓存是否按读写穿透/失效逻辑工作，验证 Get 深拷贝的并发安全。
- **画像进化测试**：位于 [evolution_service_test.go](../domain/memory/profile/evolution_service_test.go)。模拟 `Extractor` 和 `ProfileRepository` 的调用，测试进化流程在出错和正常状况下的状态机合并与数据清洗规则。
- **全文检索测试**：位于 [retrieval_service_test.go](../domain/memory/retrieval/retrieval_service_test.go)。验证空参防御行为以及正常条件下的持久化转发行为。
- **规避循环导入**：测试文件被声明为外部测试包（例如 `package session_test`），从而使测试代码不会产生循环依赖。

#### 基础设施单元测试
- **SQLite 仓储单元测试**：位于 [session_test.go](../infra/persistence/sqlite/session_test.go) 和 [retrieval_test.go](../infra/persistence/sqlite/retrieval_test.go)。
  - **全局测试脚手架**：在 [main_test.go](../infra/persistence/sqlite/main_test.go) 中采用 `TestMain` 管理测试生命周期。测试执行时会自动连接 SQLite 内存数据库（`:memory:`）并通过加载 `scripts/sqlite_memory.sql` 文件进行 Schema 结构的模拟初始化。
  - **测试隔离**：每次子测试开始前均清空相关表数据，保证测试之间的彻底隔离。
- **文件仓储单元测试**：位于 [profile_test.go](../infra/persistence/file/profile_test.go)。
  - 验证在临时目录下偏好/事实文件的创建、读取、转换以及解析带有各种非标准 Markdown 标识行时的稳定性。
