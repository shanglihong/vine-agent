package session

import (
	"context"
)

//go:generate mockgen -source=repository_interface.go -destination=./mock/session_repository_mock.go -package=mock

// Repository 定义了 Session 的物理持久化（仓储）接口契约，使用领域实体以避免污染领域层
type Repository interface {
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

