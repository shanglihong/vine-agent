package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"vine-agent/domain/memory/profile"
	"vine-agent/domain/memory/session"
	"vine-agent/domain/message"
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

// TriggerEvolution 触发记忆演进过程。支持传入批量 sessionIDs，并按照 userID 进行聚合请求，完成合并后持久化更新。
func (a *EvolutionAppService) TriggerEvolution(ctx context.Context, sessionIDs []string) error {
	// 1. 批量获取会话
	sessionsMap, getErr := a.sessionSvc.GetBatch(ctx, sessionIDs)

	// 2. 按 userID 聚合会话
	userSessions := make(map[string][]*session.Session)
	for _, sess := range sessionsMap {
		userSessions[sess.UserID] = append(userSessions[sess.UserID], sess)
	}

	// 3. 对每个 userID 并行执行记忆聚合与演进
	var evolveErrs []error
	var evolveErrsMu sync.Mutex
	var evolveWg sync.WaitGroup

	for uID, sessList := range userSessions {
		evolveWg.Add(1)
		go func(userID string, sList []*session.Session) {
			defer evolveWg.Done()

			// 合并该用户所有会话中的未进化增量消息，过滤非对话消息并清除思考内容
			var combinedMessages []message.Message
			for _, sess := range sList {
				lastCount := sess.GetLastEvolvedMsgCount()
				currentCount := len(sess.Messages)
				if currentCount > lastCount {
					for _, msg := range sess.Messages[lastCount:currentCount] {
						// 仅保留对话角色消息（user 和 assistant）
						if msg.Role != message.RoleUser && msg.Role != message.RoleAssistant {
							continue
						}

						// 过滤思考推导消息，将其 ReasoningContent 置空，只保留 Content
						cleanMsg := msg
						cleanMsg.ReasoningContent = ""

						combinedMessages = append(combinedMessages, cleanMsg)
					}
				}
			}

			// 无增量消息，无需触发演进
			if len(combinedMessages) == 0 {
				return
			}

			// 获取用户长期记忆画像
			prof, err := a.profileRepo.GetByUserID(ctx, userID)
			if err != nil {
				evolveErrsMu.Lock()
				evolveErrs = append(evolveErrs, fmt.Errorf("failed to retrieve profile for user %s: %w", userID, err))
				evolveErrsMu.Unlock()
				return
			}
			if prof == nil {
				// 新用户，初始化空白 Profile
				prof = profile.NewProfile(userID)
			}

			// 编排领域服务执行演进（对该用户的全部增量消息做单次 LLM 提炼聚合）
			err = a.evolutionSvc.Evolve(ctx, prof, combinedMessages)
			if err != nil {
				evolveErrsMu.Lock()
				evolveErrs = append(evolveErrs, fmt.Errorf("failed to evolve profile for user %s: %w", userID, err))
				evolveErrsMu.Unlock()
				return
			}

			// 持久化画像
			err = a.profileRepo.Save(ctx, prof)
			if err != nil {
				evolveErrsMu.Lock()
				evolveErrs = append(evolveErrs, fmt.Errorf("failed to save profile for user %s: %w", userID, err))
				evolveErrsMu.Unlock()
				return
			}

			// 更新涉及的所有 Session 的进度计数，并写回存储
			for _, sess := range sList {
				lastCount := sess.GetLastEvolvedMsgCount()
				currentCount := len(sess.Messages)
				if currentCount > lastCount {
					sess.UpdateLastEvolvedMsgCount()
					err = a.sessionSvc.Save(ctx, sess)
					if err != nil {
						evolveErrsMu.Lock()
						evolveErrs = append(evolveErrs, fmt.Errorf("failed to update and save session %s: %w", sess.ID, err))
						evolveErrsMu.Unlock()
					}
				}
			}
		}(uID, sessList)
	}

	evolveWg.Wait()

	// 4. 汇总所有加载和演化阶段的错误
	var allErrs []error
	if getErr != nil {
		allErrs = append(allErrs, getErr)
	}
	allErrs = append(allErrs, evolveErrs...)

	if len(allErrs) > 0 {
		return errors.Join(allErrs...)
	}

	return nil
}
