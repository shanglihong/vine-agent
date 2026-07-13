package router

import (
	"context"
	"vine-agent/cmd/bootstrap"
	"vine-agent/cmd/http/dto"
	"vine-agent/domain/memory/profile"
	"vine-agent/domain/user"
)

var (
	userHandler = UserHandler{}
)

type UserHandler struct{}

func GetUserHandler() *UserHandler {
	return &userHandler
}

func (h *UserHandler) GetUser(ctx context.Context, _ dto.Null) (*user.User, error) {
	// TODO 后续考虑使用jwt处理用户登录
	u, err := bootstrap.GetAppContainer().UserAppService.GetUser(ctx)
	return u, err
}

func (h *UserHandler) GetUserProfile(ctx context.Context, req dto.UserIdReq) (*profile.Profile, error) {
	prof, err := bootstrap.GetAppContainer().UserAppService.GetUserProfile(ctx, req.UserID)
	return prof, err
}

func (h *UserHandler) Evolve(ctx context.Context, req dto.UserIdReq) (*profile.Profile, error) {
	prof, err := bootstrap.GetAppContainer().UserAppService.EvolveAndGetProfile(ctx, req.UserID, req.SessionID)
	return prof, err
}
