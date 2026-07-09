package session

import (
	"context"
)

//go:generate go run github.com/golang/mock/mockgen -source=interface.go -destination=./mock/session_mock.go -package=mock

// SessionRepository 定义了 Session 的物理持久化（仓储）接口契约，使用领域实体以避免污染领域层
type SessionRepository interface {
	// Save 保存或更新完整的 Session 领域对象
	Save(ctx context.Context, sess *Session) error

	// Get 根据 ID 获取 Session 领域对象
	// 如果不存在，应当返回 ErrSessionNotFound 错误
	Get(ctx context.Context, id string) (*Session, error)

	// Delete 根据 ID 删除 Session 领域对象
	Delete(ctx context.Context, id string) error

	// List 根据 UserID 列出该用户的所有会话，列表不携带历史消息详情
	List(ctx context.Context, userID string) ([]*Session, error)
}

// SessionService 定义了 Session 领域服务的核心操作契约
type SessionService interface {
	// Save 保存 Session 到物理存储中，并使内存缓存中的对应记录失效以保持数据一致性
	Save(ctx context.Context, sess *Session) error

	// Get 首先尝试从并发安全内存缓存中读取 Session，如果未命中则穿透到底层物理持久化中读取并缓存
	Get(ctx context.Context, id string) (*Session, error)

	// Delete 从物理持久化中删除 Session，并清理内存缓存中的记录
	Delete(ctx context.Context, id string) error

	// List 从物理持久化中拉取用户会话列表（不携带冗余的历史消息详情），此方法不走缓存
	List(ctx context.Context, userID string) ([]*Session, error)

	// Rename 重命名指定的会话并使缓存失效
	Rename(ctx context.Context, id string, name string) error
}
