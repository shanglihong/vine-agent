package agent_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appagent "vine-agent/app/agent"
	agentmock "vine-agent/app/agent/mock"
	"vine-agent/domain/chat"
	"vine-agent/domain/memory/session"
	sessionmock "vine-agent/domain/memory/session/mock"
	"vine-agent/domain/message"
	"vine-agent/domain/tool"
)

func TestInteractionService_ReadStream(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAgentSvc := agentmock.NewMockService(ctrl)
	mockSessionSvc := sessionmock.NewMockSessionService(ctrl)
	interactionSvc := appagent.NewInteractionService(mockAgentSvc, mockSessionSvc)

	t.Run("normal stream aggregation", func(t *testing.T) {
		msgs := []*message.StreamMessage{
			{Type: message.StreamMessageTextDelta, Content: "Hello"},
			{Type: message.StreamMessageTextDelta, Content: " world!"},
		}
		reader := &testStreamReader{msgs: msgs}
		res, err := interactionSvc.ReadStream(context.Background(), reader)
		require.NoError(t, err)
		assert.Equal(t, "Hello world!", res.Content)
		assert.Nil(t, res.Interrupt)
	})

	t.Run("stream interrupted", func(t *testing.T) {
		msgs := []*message.StreamMessage{
			{Type: message.StreamMessageTextDelta, Content: "Thinking..."},
		}
		interruptErr := session.NewPendingConfirmationError("sess_123", &message.Message{Role: message.RoleAssistant}, nil)
		reader := &testStreamReader{msgs: msgs, err: interruptErr}
		res, err := interactionSvc.ReadStream(context.Background(), reader)
		require.NoError(t, err)
		assert.Equal(t, "Thinking...", res.Content)
		assert.NotNil(t, res.Interrupt)
		assert.Equal(t, "sess_123", res.Interrupt.SessionID)
	})

	t.Run("stream other error", func(t *testing.T) {
		expectedErr := errors.New("some system error")
		reader := &testStreamReader{err: expectedErr}
		res, err := interactionSvc.ReadStream(context.Background(), reader)
		require.Error(t, err)
		assert.True(t, errors.Is(err, expectedErr))
		assert.Nil(t, res)
	})
}

func TestInteractionService_ResumeStream(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAgentSvc := agentmock.NewMockService(ctrl)
	mockSessionSvc := sessionmock.NewMockSessionService(ctrl)
	interactionSvc := appagent.NewInteractionService(mockAgentSvc, mockSessionSvc)

	sessionID := "sess-resume-test"
	userID := "user-123"

	// 敏感工具 A (需要确认)
	toolA := &dummyTool{
		info: tool.Definition{
			Name:                 "sensitive_tool_a",
			RequiresConfirmation: true,
		},
		execute: func(ctx context.Context, args string) (string, error) {
			return "result_a", nil
		},
	}
	// 敏感工具 B (需要确认)
	toolB := &dummyTool{
		info: tool.Definition{
			Name:                 "sensitive_tool_b",
			RequiresConfirmation: true,
		},
		execute: func(ctx context.Context, args string) (string, error) {
			return "result_b", nil
		},
	}

	opts := []chat.OptionFunc{
		chat.WithTools([]tool.Tool{toolA, toolB}),
	}

	t.Run("session not found in context", func(t *testing.T) {
		reader, err := interactionSvc.ResumeStream(context.Background(), []string{"call_a"}, opts...)
		require.Error(t, err)
		assert.Nil(t, reader)
		assert.Contains(t, err.Error(), "session id not found")
	})

	t.Run("session not found from svc", func(t *testing.T) {
		ctx := appagent.WithSessionID(context.Background(), sessionID)
		ctx = appagent.WithUserID(ctx, userID)

		mockSessionSvc.EXPECT().Get(gomock.Any(), sessionID).Return(nil, session.ErrSessionNotFound).Times(1)

		reader, err := interactionSvc.ResumeStream(ctx, []string{"call_a"}, opts...)
		require.Error(t, err)
		assert.True(t, errors.Is(err, session.ErrSessionNotFound))
		assert.Nil(t, reader)
	})

	t.Run("session not in pending_confirmation status", func(t *testing.T) {
		ctx := appagent.WithSessionID(context.Background(), sessionID)
		ctx = appagent.WithUserID(ctx, userID)

		sess := session.NewSession(sessionID, userID, nil)
		mockSessionSvc.EXPECT().Get(gomock.Any(), sessionID).Return(sess, nil).Times(1)

		reader, err := interactionSvc.ResumeStream(ctx, []string{"call_a"}, opts...)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not in pending_confirmation status")
		assert.Nil(t, reader)
	})

	t.Run("all tools confirmed and resume success", func(t *testing.T) {
		ctx := appagent.WithSessionID(context.Background(), sessionID)
		ctx = appagent.WithUserID(ctx, userID)

		sess := session.NewSession(sessionID, userID, nil)
		sess.MarkPendingConfirmation()
		sess.Metadata[session.MetadataPendingConfirmToolCallIDs] = "call_a,call_b"
		sess.Metadata[session.MetadataPendingConfirmToolNames] = "sensitive_tool_a,sensitive_tool_b"

		assistantMsg := message.Message{
			Role: message.RoleAssistant,
			ToolCalls: []message.ToolCall{
				{
					ID: "call_a",
					Function: message.FunctionCall{
						Name:      "sensitive_tool_a",
						Arguments: "{}",
					},
				},
				{
					ID: "call_b",
					Function: message.FunctionCall{
						Name:      "sensitive_tool_b",
						Arguments: "{}",
					},
				},
			},
		}
		sess.Messages = append(sess.Messages, assistantMsg)

		mockSessionSvc.EXPECT().Get(gomock.Any(), sessionID).Return(sess, nil).Times(1)
		mockSessionSvc.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, s *session.Session) error {
			assert.Equal(t, "", s.GetStatus())
			assert.NotContains(t, s.Metadata, session.MetadataPendingConfirmToolCallIDs)
			// 此时 Messages 应该增加了两个 ToolMessage
			require.Len(t, s.Messages, 3)
			assert.Equal(t, message.RoleTool, s.Messages[1].Role)
			assert.Equal(t, "call_a", s.Messages[1].ToolCallID)
			assert.Equal(t, "result_a", s.Messages[1].Content)

			assert.Equal(t, message.RoleTool, s.Messages[2].Role)
			assert.Equal(t, "call_b", s.Messages[2].ToolCallID)
			assert.Equal(t, "result_b", s.Messages[2].Content)
			return nil
		}).Times(1)

		mockReader := &testStreamReader{}
		mockAgentSvc.EXPECT().Stream(gomock.Any(), nil, gomock.Any()).DoAndReturn(func(ctx context.Context, msgs []message.Message, o ...chat.OptionFunc) (message.StreamMessageReader, error) {
			// 校验 ctx 里是否包含了已确认的 tool call IDs
			assert.True(t, appagent.IsToolCallConfirmed(ctx, "call_a"))
			assert.True(t, appagent.IsToolCallConfirmed(ctx, "call_b"))
			return mockReader, nil
		}).Times(1)

		reader, err := interactionSvc.ResumeStream(ctx, []string{"call_a", "call_b"}, opts...)
		require.NoError(t, err)
		assert.Equal(t, mockReader, reader)
	})

	t.Run("partial tools confirmed and remains interrupted", func(t *testing.T) {
		ctx := appagent.WithSessionID(context.Background(), sessionID)
		ctx = appagent.WithUserID(ctx, userID)

		sess := session.NewSession(sessionID, userID, nil)
		sess.MarkPendingConfirmation()
		sess.Metadata[session.MetadataPendingConfirmToolCallIDs] = "call_a,call_b"
		sess.Metadata[session.MetadataPendingConfirmToolNames] = "sensitive_tool_a,sensitive_tool_b"

		assistantMsg := message.Message{
			Role: message.RoleAssistant,
			ToolCalls: []message.ToolCall{
				{
					ID: "call_a",
					Function: message.FunctionCall{
						Name:      "sensitive_tool_a",
						Arguments: "{}",
					},
				},
				{
					ID: "call_b",
					Function: message.FunctionCall{
						Name:      "sensitive_tool_b",
						Arguments: "{}",
					},
				},
			},
		}
		sess.Messages = append(sess.Messages, assistantMsg)

		mockSessionSvc.EXPECT().Get(gomock.Any(), sessionID).Return(sess, nil).Times(1)
		mockSessionSvc.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, s *session.Session) error {
			// 依然是 pending_confirmation 状态
			assert.Equal(t, session.SessionStatusPendingConfirmation, s.GetStatus())
			// 只执行了 call_a，剩下 call_b 依然待确认
			assert.Equal(t, "call_b", s.Metadata[session.MetadataPendingConfirmToolCallIDs])
			assert.Equal(t, "sensitive_tool_b", s.Metadata[session.MetadataPendingConfirmToolNames])

			// Messages 里增加了已确认 the call_a的结果
			require.Len(t, s.Messages, 2)
			assert.Equal(t, message.RoleTool, s.Messages[1].Role)
			assert.Equal(t, "call_a", s.Messages[1].ToolCallID)
			assert.Equal(t, "result_a", s.Messages[1].Content)
			return nil
		}).Times(1)

		// 不应该调用 agentSvc.Stream
		mockAgentSvc.EXPECT().Stream(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		reader, err := interactionSvc.ResumeStream(ctx, []string{"call_a"}, opts...)
		require.Error(t, err)
		assert.Nil(t, reader)

		// 应该返回一个新的 InterruptError，里面仅包含 call_b
		var interruptErr *session.InterruptError
		assert.True(t, errors.As(err, &interruptErr))
		assert.Equal(t, session.SessionStatusPendingConfirmation, interruptErr.Status)
		require.Len(t, interruptErr.ToolCalls, 1)
		assert.Equal(t, "call_b", interruptErr.ToolCalls[0].ID)
	})
}
