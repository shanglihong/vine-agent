package user

import (
	"context"
	"fmt"
)

type userService struct {
	repo UserRepository
}

// NewUserService 构造一个具体的领域服务实现
func NewUserService(repo UserRepository) UserService {
	return &userService{repo: repo}
}

// GetUser 获取用户信息
func (s *userService) GetUser(ctx context.Context, id string) (*User, error) {
	if id == "" {
		return nil, fmt.Errorf("user id cannot be empty")
	}
	return s.repo.Get(ctx, id)
}
