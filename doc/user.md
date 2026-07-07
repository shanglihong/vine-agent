# User 模块设计文档

User 模块负责系统的用户领域信息管理，为智能体会话隔离、长期画像归档以及接口层的权限隔离提供底层的用户数据支撑。

---

## 1. 架构定位

在系统的三层 DDD 架构中，User 模块跨越领域层与基础设施层：
- **领域层（Domain Layer）**：
  - **接口定义**：[interface.go](../domain/user/interface.go)，定义了 `UserRepository` 与 `UserService` 接口契约，以及 `ErrUserNotFound` 领域异常。
  - **核心实体**：[user.go](../domain/user/user.go)，定义了 `User` 聚合根。
  - **领域服务**：[user_service.go](../domain/user/user_service.go)，提供了具体业务逻辑实现的私有 `userService` 结构体。
- **基础设施层（Infrastructure Layer）**：
  - **仓储实现**：[user.go](../infra/persistence/sqlite/user.go)，提供了基于 SQLite 的具体仓储适配实现 `UserStore`。
- **职责划分**：领域服务及实体只关注业务决策（如入参校验、构造新实体）；基础设施层维持无状态，专职于物理存储交互，严禁在基础设施层中做任何业务逻辑决断。

---

## 2. 核心组件与实体

### 2.1 用户聚合根：[User](../domain/user/user.go)
代表系统中的用户主体。它是一个实体对象，主要字段包括：
- `ID` (string)：用户唯一标识 ID（前端与后端交互和仓储索引的关键键）。
- `Username` (string)：用户登录/展示名。
- `Email` (string)：用户绑定的邮箱。
- `CreatedAt` / `UpdatedAt` (time.Time)：实体的创建与更新时间。

##### 构造行为
- 提供构造函数 `NewUser(id, username, email)`，用于安全地初始化一个新的用户实体，并默认打上当前时间的物理时间戳。

### 2.2 仓储契约：[UserRepository](../domain/user/interface.go)
解耦领域层与存储介质的物理适配接口：
```go
type UserRepository interface {
	Get(ctx context.Context, id string) (*User, error)
}
```

### 2.3 领域服务接口：[UserService](../domain/user/interface.go)
提供用户管理核心操作：
```go
type UserService interface {
	GetUser(ctx context.Context, id string) (*User, error)
}
```
其具体私有实现 `userService`（[user_service.go](../domain/user/user_service.go)）具有以下特性：
- **依赖倒置**：注入 `UserRepository` 契约，消除了对基础设施底层技术的直接依赖。
- **业务校验**：在 `GetUser` 阶段执行强防卫性校验（如用户 ID 不能为空），提前拦截非法请求，成功后再调用 Repository 执行读取。

---

## 3. 具体实现与适配

### SQLite 仓储适配器：[UserStore](../infra/persistence/sqlite/user.go)

- **实现说明**：`UserStore` 结构体实现了领域层的 `UserRepository` 契约。
- **惰性连接**：与其它仓储类似，实例化时传入数据库路径并缓存，在具体操作时才会初始化底层物理连接，避免阻塞主线程的启动。
- **只读模拟逻辑**：由于当前业务侧只要求根据请求中的 User ID 静态读取数据返回，不需要支持用户在线注册或后台修改等写流程，因此目前的物理 `Get` 方法采用的是**硬编码静态返回**的机制。在传入任意非空 `id` 时返回用户名 `TestUser` 且创建时间固定为 `2026-07-07` 的数据对象；如果 `id` 为空则抛出领域级的 `ErrUserNotFound` 错误。

---

## 4. 测试与 Mock

### 4.1 Mock 生成与使用

在接口契约定义文件 [interface.go](../domain/user/interface.go) 头部声明了：
```go
//go:generate go run github.com/golang/mock/mockgen -source=interface.go -destination=./mock/user_mock.go -package=mock
```
- **生成命令**：在项目根目录运行 `go generate ./...` 一键生成 Mock 类。
- **生成产物**：Mock 类文件将保存在包内的 `./mock` 子目录中，从而保持主包结构洁净。

### 4.2 单元测试

- **测试文件**：[user_service_test.go](../domain/user/user_service_test.go)
- **规避测试循环导入**：测试文件被声明在独立的测试包 `package user_test` 中，从机制上避免了对主体 `user` 包引入循环导包的风险。
- **测试逻辑**：
  - 使用 `gomock` 框架自动生成 `MockUserRepository` 实例。
  - `t.Run` 细分三个分支测试：
    1. 传入空 ID 时，领域服务层是否成功进行了防卫性校验并报错；
    2. 当 Repository 模拟返回 `ErrUserNotFound` 时，服务层是否正确向上传递领域错误；
    3. 模拟成功获取用户时，返回值与预期 User 实体对象完全契合。
