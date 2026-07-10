package memory_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"vine-agent/app/memory"
	profile_mock "vine-agent/domain/memory/profile/mock"
	"vine-agent/domain/memory/session"
	session_mock "vine-agent/domain/memory/session/mock"
	"vine-agent/domain/message"
)

func TestEvolutionAppService_TriggerEvolution(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSessionSvc := session_mock.NewMockSessionService(ctrl)
	mockProfileRepo := profile_mock.NewMockProfileRepository(ctrl)
	mockEvolutionSvc := profile_mock.NewMockEvolutionService(ctrl)

	appSvc := memory.NewEvolutionAppService(mockSessionSvc, mockProfileRepo, mockEvolutionSvc)
	ctx := context.Background()

	t.Run("empty sessionIDs should do nothing", func(t *testing.T) {
		mockSessionSvc.EXPECT().GetBatch(gomock.Any(), nil).Return(nil, nil).Times(1)
		err := appSvc.TriggerEvolution(ctx, nil)
		assert.NoError(t, err)
	})

	t.Run("sessions of same user should aggregate", func(t *testing.T) {
		userID := "user_1"
		sessID1 := "sess_1"
		sessID2 := "sess_2"

		// 模拟会话数据
		sess1 := session.NewSession(sessID1, userID, nil)
		sess1.Messages = []message.Message{
			{Role: message.RoleUser, Content: "Msg 1"},
		}
		// LastEvolvedMsgCount 为 0，有 1 条增量消息

		sess2 := session.NewSession(sessID2, userID, nil)
		sess2.Messages = []message.Message{
			{Role: message.RoleUser, Content: "Msg 2"},
		}
		// LastEvolvedMsgCount 为 0，有 1 条增量消息

		// 预期调用
		mockSessionSvc.EXPECT().GetBatch(gomock.Any(), []string{sessID1, sessID2}).Return(map[string]*session.Session{
			sessID1: sess1,
			sessID2: sess2,
		}, nil).Times(1)

		mockProfileRepo.EXPECT().GetByUserID(gomock.Any(), userID).Return(nil, nil).Times(1)

		// 验证演化时合并了两条消息
		mockEvolutionSvc.EXPECT().Evolve(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, prof any, msgs []message.Message) error {
				assert.Len(t, msgs, 2)
				// 顺序可能是并发的不确定，但内容应该存在
				contents := map[string]bool{}
				for _, m := range msgs {
					contents[m.Content] = true
				}
				assert.True(t, contents["Msg 1"])
				assert.True(t, contents["Msg 2"])
				return nil
			},
		).Times(1)

		mockProfileRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		// 验证两个 session 都更新了进度并写回
		mockSessionSvc.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, s *session.Session) error {
				assert.Equal(t, 1, s.GetLastEvolvedMsgCount())
				return nil
			},
		).Times(2)

		err := appSvc.TriggerEvolution(ctx, []string{sessID1, sessID2})
		assert.NoError(t, err)
	})

	t.Run("sessions of different users should run in parallel", func(t *testing.T) {
		userID1 := "user_a"
		userID2 := "user_b"
		sessID1 := "sess_a"
		sessID2 := "sess_b"

		sess1 := session.NewSession(sessID1, userID1, nil)
		sess1.Messages = []message.Message{{Role: message.RoleUser, Content: "Hello User A"}}

		sess2 := session.NewSession(sessID2, userID2, nil)
		sess2.Messages = []message.Message{{Role: message.RoleUser, Content: "Hello User B"}}

		mockSessionSvc.EXPECT().GetBatch(gomock.Any(), []string{sessID1, sessID2}).Return(map[string]*session.Session{
			sessID1: sess1,
			sessID2: sess2,
		}, nil).Times(1)

		// 两个用户的 Profile 获取、演进、保存，以及会话更新
		mockProfileRepo.EXPECT().GetByUserID(gomock.Any(), userID1).Return(nil, nil).Times(1)
		mockProfileRepo.EXPECT().GetByUserID(gomock.Any(), userID2).Return(nil, nil).Times(1)

		mockEvolutionSvc.EXPECT().Evolve(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(2)
		mockProfileRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(2)

		mockSessionSvc.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(2)

		err := appSvc.TriggerEvolution(ctx, []string{sessID1, sessID2})
		assert.NoError(t, err)
	})

	t.Run("one session fails to load, others proceed and return joined error", func(t *testing.T) {
		sessIDErr := "sess_error"
		sessIDOk := "sess_ok"
		userIDOk := "user_ok"

		sessOk := session.NewSession(sessIDOk, userIDOk, nil)
		sessOk.Messages = []message.Message{{Role: message.RoleUser, Content: "Hello"}}

		// 批量获取，返回一个错误以及加载成功的 map
		mockSessionSvc.EXPECT().GetBatch(gomock.Any(), []string{sessIDErr, sessIDOk}).Return(map[string]*session.Session{
			sessIDOk: sessOk,
		}, errors.New("db error")).Times(1)

		// 正常的会话继续执行演进
		mockProfileRepo.EXPECT().GetByUserID(gomock.Any(), userIDOk).Return(nil, nil).Times(1)
		mockEvolutionSvc.EXPECT().Evolve(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockProfileRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockSessionSvc.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		err := appSvc.TriggerEvolution(ctx, []string{sessIDErr, sessIDOk})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("no new messages should do nothing", func(t *testing.T) {
		userID := "user_x"
		sessID := "sess_x"

		sess := session.NewSession(sessID, userID, nil)
		sess.Messages = []message.Message{{Role: message.RoleUser, Content: "Msg"}}
		sess.UpdateLastEvolvedMsgCount() // count = 1，无增量

		mockSessionSvc.EXPECT().GetBatch(gomock.Any(), []string{sessID}).Return(map[string]*session.Session{
			sessID: sess,
		}, nil).Times(1)
		// 不应调用 profileRepo, evolutionSvc 等

		err := appSvc.TriggerEvolution(ctx, []string{sessID})
		assert.NoError(t, err)
	})

	t.Run("filter non-dialogue roles and clear reasoning content", func(t *testing.T) {
		userID := "user_filter"
		sessID := "sess_filter"

		sess := session.NewSession(sessID, userID, nil)
		sess.Messages = []message.Message{
			{Role: message.RoleSystem, Content: "system instruction"}, // 应过滤
			{Role: message.RoleUser, Content: "user chat msg"},       // 应保留
			{Role: message.RoleAssistant, Content: "assistant reply msg", ReasoningContent: "some deep thinking"}, // 应保留，且清空 ReasoningContent
			{Role: message.RoleTool, Content: "tool run output"}, // 应过滤
			{
				Role:    message.RoleAssistant,
				Content: "assistant calling tool",
				ToolCalls: []message.ToolCall{
					{ID: "tc_1"},
				},
			}, // 应保留（因为是 assistant），且清空 ReasoningContent
		}

		mockSessionSvc.EXPECT().GetBatch(gomock.Any(), []string{sessID}).Return(map[string]*session.Session{
			sessID: sess,
		}, nil).Times(1)

		mockProfileRepo.EXPECT().GetByUserID(gomock.Any(), userID).Return(nil, nil).Times(1)

		mockEvolutionSvc.EXPECT().Evolve(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, prof any, msgs []message.Message) error {
				assert.Len(t, msgs, 3)

				// 第一条保留的应该是用户消息
				assert.Equal(t, message.RoleUser, msgs[0].Role)
				assert.Equal(t, "user chat msg", msgs[0].Content)

				// 第二条保留的应该是助手消息，且 ReasoningContent 已被清空
				assert.Equal(t, message.RoleAssistant, msgs[1].Role)
				assert.Equal(t, "assistant reply msg", msgs[1].Content)
				assert.Empty(t, msgs[1].ReasoningContent)

				// 第三条保留的应该是助手调用工具消息
				assert.Equal(t, message.RoleAssistant, msgs[2].Role)
				assert.Equal(t, "assistant calling tool", msgs[2].Content)
				assert.Len(t, msgs[2].ToolCalls, 1)

				return nil
			},
		).Times(1)

		mockProfileRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockSessionSvc.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		err := appSvc.TriggerEvolution(ctx, []string{sessID})
		assert.NoError(t, err)
	})

	t.Run("parallel conflict safety in session retrieving", func(t *testing.T) {
		// 并发多 session 压力测试，确保没有并发冲突的 Data Race
		sessCount := 50
		sessionIDs := make([]string, sessCount)

		for i := 0; i < sessCount; i++ {
			sessionIDs[i] = fmt.Sprintf("sess_%d", i)
		}

		mockSessionSvc.EXPECT().GetBatch(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, ids []string) (map[string]*session.Session, error) {
				res := make(map[string]*session.Session)
				for _, id := range ids {
					sess := session.NewSession(id, "user_stress", nil)
					sess.Messages = []message.Message{{Role: message.RoleUser, Content: "Stress Message"}}
					res[id] = sess
				}
				return res, nil
			},
		).AnyTimes()

		mockProfileRepo.EXPECT().GetByUserID(gomock.Any(), "user_stress").Return(nil, nil).AnyTimes()
		mockEvolutionSvc.EXPECT().Evolve(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockProfileRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockSessionSvc.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		var getWg sync.WaitGroup
		getWg.Add(1)
		var err error
		go func() {
			defer getWg.Done()
			err = appSvc.TriggerEvolution(ctx, sessionIDs)
		}()

		getWg.Wait()
		assert.NoError(t, err)
	})
}

// 辅助打印防止 go 提示 fmt 未使用
var _ = time.Second
