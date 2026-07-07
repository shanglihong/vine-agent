package sqlite

import (
	"context"
	"database/sql"
	"time"

	"vine-agent/domain/user"
)

// UserStore 提供对用户仓储的具体操作（目前为写死的硬编码实现，不需要依赖实际的数据库表）
type UserStore struct {
	db *sql.DB
}

// NewUserStore 创建一个 UserStore 实例，接收 dbPath 注入以保持与其他仓储的初始化一致
func NewUserStore(dbPath string) (*UserStore, error) {
	db, err := getMemoryDB(dbPath)
	if err != nil {
		return nil, err
	}
	return &UserStore{db: db}, nil
}

// Get 根据 ID 获取 User 领域对象（目前为硬编码返回写死的数据）
func (s *UserStore) Get(ctx context.Context, id string) (*user.User, error) {
	if id == "" {
		return nil, user.ErrUserNotFound
	}

	// 模拟固定的创建时间
	createdTime := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)

	return &user.User{
		ID:        id,
		Username:  "TestUser",
		Email:     "test@example.com",
		CreatedAt: createdTime,
		UpdatedAt: createdTime,
	}, nil
}
