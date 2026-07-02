package memory

import (
	"context"
	"fmt"

	"vine-agent/domain/memory/profile"
	"vine-agent/domain/memory/session"
)

// EvolutionAppService 记忆进化应用层服务，负责编排短期会话与长期记忆的流转与演进
type EvolutionAppService struct {
	sessionSvc   session.SessionService
	profileRepo  profile.ProfileRepository
	evolutionSvc profile.EvolutionService
}

// NewEvolutionAppService 构造一个 EvolutionAppService 实例
func NewEvolutionAppService(
	sessionSvc session.SessionService,
	profileRepo profile.ProfileRepository,
	evolutionSvc profile.EvolutionService,
) *EvolutionAppService {
	return &EvolutionAppService{
		sessionSvc:   sessionSvc,
		profileRepo:  profileRepo,
		evolutionSvc: evolutionSvc,
	}
}

// TriggerEvolution 触发记忆演进过程。将提取未进化的增量消息，完成合并，并持久化更新
func (a *EvolutionAppService) TriggerEvolution(ctx context.Context, sessionID string) error {
	// 1. 获取会话（短期记忆）
	sess, err := a.sessionSvc.Get(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to retrieve session %s: %w", sessionID, err)
	}
	if sess == nil {
		return fmt.Errorf("session %s not found", sessionID)
	}

	// 2. 检查是否有未进化的新消息
	lastCount := sess.GetLastEvolvedMsgCount()
	currentCount := len(sess.Messages)
	if currentCount <= lastCount {
		return nil // 无增量消息，无需触发进化
	}

	// 3. 截取未演进的增量消息列表
	deltaMessages := sess.Messages[lastCount:currentCount]

	// 4. 获取用户长期记忆画像
	prof, err := a.profileRepo.GetByUserID(ctx, sess.UserID)
	if err != nil {
		return fmt.Errorf("failed to retrieve profile for user %s: %w", sess.UserID, err)
	}
	if prof == nil {
		// 新用户，初始化空白 Profile
		prof = profile.NewProfile(sess.UserID)
	}

	// 5. 编排领域服务执行演进
	err = a.evolutionSvc.Evolve(ctx, prof, deltaMessages)
	if err != nil {
		return fmt.Errorf("failed to evolve profile: %w", err)
	}

	// 6. 持久化画像
	err = a.profileRepo.Save(ctx, prof)
	if err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	// 7. 更新 Session 中的演进消息进度计数，并持久化更新后的会话状态
	sess.UpdateLastEvolvedMsgCount()
	err = a.sessionSvc.Save(ctx, sess)
	if err != nil {
		return fmt.Errorf("failed to update and save session evolution count: %w", err)
	}

	return nil
}
