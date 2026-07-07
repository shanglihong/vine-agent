package user

import (
	"context"
	"vine-agent/app/agent"
	"vine-agent/domain/user"
)

// UserAppService 用户应用层服务，负责编排用户相关的业务用例
type UserAppService struct {
	userSvc user.UserService
}

// NewUserAppService 构造一个 UserAppService 实例
func NewUserAppService(userSvc user.UserService) *UserAppService {
	return &UserAppService{userSvc: userSvc}
}

// GetUser 从 context 获取 userID 进而读取用户信息
func (a *UserAppService) GetUser(ctx context.Context) (*user.User, error) {
	userID := agent.GetUserID(ctx)
	return a.userSvc.GetUser(ctx, userID)
}
