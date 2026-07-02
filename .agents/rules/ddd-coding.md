---
trigger: always_on
---

# DDD 编程指南
## 代码目录层级
代码根据业务功能分为三层
- app：应用层，存放编排代码，编排业务逻辑
- domain：领域层，存放业务领域代码，一个领域一个子目录，例如`domain/memory`
- infra：基础设施层，存放外部依赖的代码，例如`infra/client`

其他
- uitls：通用工具代码存放层级，工具代码仅依赖标准库

## 层级依赖约束
- infra 层**禁止**做业务决策（默认值填充、重试策略、状态流转等），这些属于 Domain 层
- domain 层**严禁**出现 ORM tag、数据库驱动引用或 infra 包的 import
- domain 层通过**仓储接口**与 infra 层交互，领域服务只注入仓储接口，不注入其他服务
- **允许** infra 层使用 domain 层定义的领域错误，契约接口，契约实体对象

**注意**：根据六边形架构，infra属于外层

## 编程实践
### DDD规范
实体
- 实体包含数据和内聚行为，例如状态转换，状态判断等无外部依赖的行为
- 每个实体的行为应该尽可能内聚

领域服务
- 领域服务只包含业务逻辑，作为不同的实体的编排动作
- **接口与解耦规范**：领域服务对外应暴露接口契约（例如 `XxxService`），具体的实现类定义为私有的结构体（例如 `xxxService`），从而对调用方屏蔽内部细节。
- **依赖注入与传递防护**：上层（如 `app` 应用层）在注入时必须仅依赖领域服务接口，从而消除对具体类依赖的装配传递。
- **接口文件收拢**：领域的仓储接口（如 `XxxRepository`）与服务接口（如 `XxxService`）统一合并收拢在同包下的 `interface.go` 文件中管理，以提高内聚性。
- 非主要流程行为通过领域消息进行解耦

基础设施
- 连接使用 `sync.Once` **惰性初始化**，在首次操作时建立，避免进程启动时阻塞
- 持久化结构体保持**无状态**，支持并发安全调用
- 涉及多步骤操作，需要详细考虑事务与异常处理

### 编码规范
- 符合go最佳实践

## Mock 与单元测试
- 使用 `gomock` + `testify`，标准结构：
  ```go
  func TestXxx(t *testing.T) {
      ctrl := gomock.NewController(t)
      defer ctrl.Finish()
      mockRepo := mock.NewMockXxxRepository(ctrl)
      t.Run("子测试名称", func(t *testing.T) {
          mockRepo.EXPECT().Get(ctx, id).Return(entity, nil).Times(1)
      })
  }
  ```
- **一键生成 Mock 与本地定位规范**：
  在项目根目录运行 `go generate ./...` 一键生成所有 Mock。为了避免外部模块依赖及路径解析报错，请在接口文件上方使用 `-source` 参数声明 `go:generate`，并将 Mock 产物存放在接口所在目录的子包 `./mock` 中。
  为了防止因执行环境未全局安装 `mockgen` 工具而报错，推荐使用 `go run` 替代裸调用以实现零依赖的一键自动生成：
  ```go
  //go:generate go run github.com/golang/mock/mockgen -source=interface.go -destination=./mock/xxx_mock.go -package=mock
  ```
- **规避测试循环导入 (Import Cycle)**：
  测试文件必须声明为**外部测试包**（如 `package xxx_test`）,私用方法测试放到一个独立的`_test.go`文件中。
- **全局测试脚手架抽取规范**：
  若测试中涉及全局资源初始化与销毁（例如Schema模拟初始化表），请使用 `TestMain` 进行统一逻辑管理，并且抽取到同包下一个独立的测试文件 `main_test.go` 中，保证具体业务测试专注且纯净。