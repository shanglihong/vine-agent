package user

import (
	"context"
	"strings"

	"vine-agent/app/agent"
	"vine-agent/app/memory"
	"vine-agent/domain/memory/profile"
	"vine-agent/domain/memory/session"
	"vine-agent/domain/user"
)

// UserAppService 用户应用层服务，负责编排用户相关的业务用例
type UserAppService struct {
	userSvc         user.UserService
	sessionSvc      session.SessionService
	profileRepo     profile.ProfileRepository
	evolutionAppSvc *memory.EvolutionAppService
}

// NewUserAppService 构造一个 UserAppService 实例
func NewUserAppService(
	userSvc user.UserService,
	sessionSvc session.SessionService,
	profileRepo profile.ProfileRepository,
	evolutionAppSvc *memory.EvolutionAppService,
) *UserAppService {
	return &UserAppService{
		userSvc:         userSvc,
		sessionSvc:      sessionSvc,
		profileRepo:     profileRepo,
		evolutionAppSvc: evolutionAppSvc,
	}
}

// GetUser 从 context 获取 userID 进而读取用户信息
func (a *UserAppService) GetUser(ctx context.Context) (*user.User, error) {
	userID := agent.GetUserID(ctx)
	return a.userSvc.GetUser(ctx, userID)
}

// GetUserProfile 获取用户偏好与事实画像，支持如果 userIDOrSessionID 为会话 ID 则执行反推 userID 的决策
func (a *UserAppService) GetUserProfile(ctx context.Context, userIDOrSessionID string) (*profile.Profile, error) {
	userID := userIDOrSessionID
	if strings.HasPrefix(userID, "sess_") {
		sess, err := a.sessionSvc.Get(ctx, userID)
		if err == nil && sess != nil {
			userID = sess.UserID
		}
	}

	prof, err := a.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if prof == nil {
		// 返回空画像
		return &profile.Profile{
			UserID:      userID,
			Preferences: []string{},
			Facts:       []string{},
		}, nil
	}

	return prof, nil
}

// EvolveAndGetProfile 编排长期记忆画像演进并获取演进后的最新画像
func (a *UserAppService) EvolveAndGetProfile(ctx context.Context, userIDOrSessionID, sessionID string) (*profile.Profile, error) {
	userID := userIDOrSessionID
	if userID == "" || strings.HasPrefix(userID, "sess_") {
		sess, err := a.sessionSvc.Get(ctx, sessionID)
		if err == nil && sess != nil {
			userID = sess.UserID
		}
	}

	// 1. 触发进化编排
	if err := a.evolutionAppSvc.TriggerEvolution(ctx, []string{sessionID}); err != nil {
		return nil, err
	}

	// 2. 捞取最新画像并返回
	return a.GetUserProfile(ctx, userID)
}
