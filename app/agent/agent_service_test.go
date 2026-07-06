package agent_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appagent "vine-agent/app/agent"
	"vine-agent/domain/chat"
	chatmock "vine-agent/domain/chat/mock"
	"vine-agent/domain/memory/session"
	sessionmock "vine-agent/domain/memory/session/mock"
	"vine-agent/domain/message"
	"vine-agent/domain/tool"
	infraevent "vine-agent/infra/event"
)

type testStreamReader struct {
	msgs  []*message.StreamMessage
	index int
	err   error
}

func (r *testStreamReader) Recv() (*message.StreamMessage, error) {
	if r.index >= len(r.msgs) {
		if r.err != nil {
			return nil, r.err
		}
		return nil, io.EOF
	}
	msg := r.msgs[r.index]
	r.index++
	return msg, nil
}

func (r *testStreamReader) Close() error {
	return nil
}

func TestAgentAppService_StreamEventDecoupling(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatModel := chatmock.NewMockChatModel(ctrl)
	mockSessionSvc := sessionmock.NewMockSessionService(ctrl)

	// 使用真实的并发安全内存事件总线作为 Publisher 和 Subscriber（单 worker 保证强保序分发）
	eventBus := infraevent.NewInMemoryEventBus(10, 1, nil)
	defer eventBus.Shutdown(context.Background())

	// 实例化事件化改造后的 Service
	svc := appagent.NewService(mockChatModel, mockSessionSvc, eventBus, eventBus)

	ctx := context.Background()
	sessionID := "test-session-123"
	userID := "user-123"

	// 将会话元数据注入 context 中
	ctx = appagent.WithSessionID(ctx, sessionID)
	ctx = appagent.WithUserID(ctx, userID)

	// 1. Mock Session 的读取与保存行为
	sess := session.NewSession(sessionID, userID, nil)
	mockSessionSvc.EXPECT().Get(gomock.Any(), sessionID).Return(sess, nil).Times(1)
	mockSessionSvc.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// 2. 模拟 ChatModel Stream 方法返回的增量生成流
	streamMsgs := []*message.StreamMessage{
		{Type: message.StreamMessageTextDelta, Content: "Hello"},
		{Type: message.StreamMessageTextDelta, Content: " world!"},
	}
	mockReader := &testStreamReader{msgs: streamMsgs}

	// 期待 ChatModel 的 Stream 方法被调用并返回 mockReader
	mockChatModel.EXPECT().Stream(gomock.Any(), gomock.Any(), gomock.Any()).Return(mockReader, nil).Times(1)

	// 3. 执行应用服务的 Stream 方法
	userMsgs := []message.Message{
		{Role: message.RoleUser, Content: "Hi"},
	}

	// 设置最大迭代次数为 1 快速完成生成测试
	opts := []chat.OptionFunc{
		chat.WithMaxIterations(1),
	}

	reader, err := svc.Stream(ctx, userMsgs, opts...)
	require.NoError(t, err)
	defer reader.Close()

	// 4. 从返回的 reader 中读取数据，验证是否通过事件总线成功分发和接收
	var results []*message.StreamMessage
	for {
		msg, err := reader.Recv()
		if err != nil {
			if err == io.EOF || (err != nil && err.Error() == "agent reached max iterations (1) without resolving") {
				break
			}
			t.Fatalf("unexpected error: %v", err)
		}
		results = append(results, msg)
	}

	require.Len(t, results, 2)
	assert.Equal(t, message.StreamMessageTextDelta, results[0].Type)
	assert.Equal(t, "Hello", results[0].Content)
	assert.Equal(t, message.StreamMessageTextDelta, results[1].Type)
	assert.Equal(t, " world!", results[1].Content)
}

func TestAgentAppService_ToolExecutionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatModel := chatmock.NewMockChatModel(ctrl)
	mockSessionSvc := sessionmock.NewMockSessionService(ctrl)

	eventBus := infraevent.NewInMemoryEventBus(10, 1, nil)
	defer eventBus.Shutdown(context.Background())

	svc := appagent.NewService(mockChatModel, mockSessionSvc, eventBus, eventBus)

	ctx := context.Background()
	sessionID := "test-session-456"
	userID := "user-456"

	ctx = appagent.WithSessionID(ctx, sessionID)
	ctx = appagent.WithUserID(ctx, userID)

	sess := session.NewSession(sessionID, userID, nil)
	mockSessionSvc.EXPECT().Get(gomock.Any(), sessionID).Return(sess, nil).Times(1)
	mockSessionSvc.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	tc := message.ToolCall{
		Index: 0,
		ID:    "call_error",
		Type:  "function",
		Function: message.FunctionCall{
			Name:      "non_existent_tool",
			Arguments: `{}`,
		},
	}
	reader1 := &testStreamReader{
		msgs: []*message.StreamMessage{
			{Type: message.StreamMessageToolCall, ToolCall: &tc},
		},
	}

	mockChatModel.EXPECT().Stream(gomock.Any(), gomock.Any(), gomock.Any()).Return(reader1, nil).Times(1)

	userMsgs := []message.Message{
		{Role: message.RoleUser, Content: "Run tool please"},
	}

	opts := []chat.OptionFunc{
		chat.WithMaxIterations(1),
	}

	reader, err := svc.Stream(ctx, userMsgs, opts...)
	require.NoError(t, err)
	defer reader.Close()

	var results []*message.StreamMessage
	var recvErr error
	for {
		msg, err := reader.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			recvErr = err
			break
		}
		results = append(results, msg)
	}

	require.Error(t, recvErr)
	assert.Contains(t, recvErr.Error(), "agent reached max iterations (1)")

	var hasToolCall, hasToolResult bool
	for _, r := range results {
		if r.Type == message.StreamMessageToolCall {
			hasToolCall = true
			assert.Equal(t, "non_existent_tool", r.ToolCall.Function.Name)
		}
		if r.Type == message.StreamMessageToolResult {
			hasToolResult = true
			assert.NotNil(t, r.ToolResult.Error)
			assert.Contains(t, r.ToolResult.Error.Error(), "not found in options")
		}
	}

	assert.True(t, hasToolCall, "should contain tool call message")
	assert.True(t, hasToolResult, "should contain tool result message with error output")

	assert.Len(t, sess.Messages, 3)
	assert.Equal(t, message.RoleUser, sess.Messages[0].Role)
	assert.Equal(t, message.RoleAssistant, sess.Messages[1].Role)
	assert.Equal(t, message.RoleTool, sess.Messages[2].Role)
	assert.Contains(t, sess.Messages[2].Content, "not found in options")
}

func TestAgentAppService_PendingConfirmation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatModel := chatmock.NewMockChatModel(ctrl)
	mockSessionSvc := sessionmock.NewMockSessionService(ctrl)

	eventBus := infraevent.NewInMemoryEventBus(10, 1, nil)
	defer eventBus.Shutdown(context.Background())

	svc := appagent.NewService(mockChatModel, mockSessionSvc, eventBus, eventBus)

	ctx := context.Background()
	sessionID := "test-session-confirm"
	userID := "user-confirm"

	ctx = appagent.WithSessionID(ctx, sessionID)
	ctx = appagent.WithUserID(ctx, userID)

	sess := session.NewSession(sessionID, userID, nil)
	mockSessionSvc.EXPECT().Get(gomock.Any(), sessionID).Return(sess, nil).Times(1)
	mockSessionSvc.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// 敏感工具
	sensitiveTool := &dummyTool{
		info: tool.Definition{
			Name:                 "sensitive_tool",
			RequiresConfirmation: true,
		},
		execute: func(ctx context.Context, args string) (string, error) {
			return "sensitive_result", nil
		},
	}

	tc := message.ToolCall{
		Index: 0,
		ID:    "call_sensitive_3",
		Type:  "function",
		Function: message.FunctionCall{
			Name:      "sensitive_tool",
			Arguments: `{}`,
		},
	}
	reader1 := &testStreamReader{
		msgs: []*message.StreamMessage{
			{Type: message.StreamMessageToolCall, ToolCall: &tc},
		},
	}

	mockChatModel.EXPECT().Stream(gomock.Any(), gomock.Any(), gomock.Any()).Return(reader1, nil).Times(1)

	userMsgs := []message.Message{
		{Role: message.RoleUser, Content: "Run sensitive tool please"},
	}

	opts := []chat.OptionFunc{
		chat.WithMaxIterations(2),
		chat.WithTools([]tool.Tool{
			sensitiveTool,
		}),
	}

	reader, err := svc.Stream(ctx, userMsgs, opts...)
	require.NoError(t, err)
	defer reader.Close()

	var results []*message.StreamMessage
	var recvErr error
	for {
		msg, err := reader.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			recvErr = err
			break
		}
		results = append(results, msg)
	}

	require.Error(t, recvErr)
	var interruptErr *session.InterruptError
	assert.True(t, errors.As(recvErr, &interruptErr))
	assert.Equal(t, session.SessionStatusPendingConfirmation, interruptErr.Status)
	assert.Equal(t, sessionID, interruptErr.SessionID)
	assert.Equal(t, session.SessionStatusPendingConfirmation, sess.GetStatus())

	// 验证挂起工具明细
	assert.Len(t, interruptErr.ToolCalls, 1)
	assert.Equal(t, "call_sensitive_3", interruptErr.ToolCalls[0].ID)
	assert.Equal(t, "sensitive_tool", interruptErr.ToolCalls[0].Function.Name)

	assert.Equal(t, "call_sensitive_3", sess.Metadata[session.MetadataPendingConfirmToolCallIDs])
	assert.Equal(t, "sensitive_tool", sess.Metadata[session.MetadataPendingConfirmToolNames])
}

func TestAgentAppService_MultiplePendingConfirmations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatModel := chatmock.NewMockChatModel(ctrl)
	mockSessionSvc := sessionmock.NewMockSessionService(ctrl)

	eventBus := infraevent.NewInMemoryEventBus(10, 1, nil)
	defer eventBus.Shutdown(context.Background())

	svc := appagent.NewService(mockChatModel, mockSessionSvc, eventBus, eventBus)

	ctx := context.Background()
	sessionID := "test-session-multi-confirm"
	userID := "user-confirm"

	ctx = appagent.WithSessionID(ctx, sessionID)
	ctx = appagent.WithUserID(ctx, userID)

	sess := session.NewSession(sessionID, userID, nil)
	mockSessionSvc.EXPECT().Get(gomock.Any(), sessionID).Return(sess, nil).Times(1)
	mockSessionSvc.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// 敏感工具 A
	sensitiveToolA := &dummyTool{
		info: tool.Definition{
			Name:                 "sensitive_tool_a",
			RequiresConfirmation: true,
		},
		execute: func(ctx context.Context, args string) (string, error) {
			return "sensitive_result_a", nil
		},
	}
	// 敏感工具 B
	sensitiveToolB := &dummyTool{
		info: tool.Definition{
			Name:                 "sensitive_tool_b",
			RequiresConfirmation: true,
		},
		execute: func(ctx context.Context, args string) (string, error) {
			return "sensitive_result_b", nil
		},
	}

	tcA := message.ToolCall{
		Index: 0,
		ID:    "call_sensitive_a",
		Type:  "function",
		Function: message.FunctionCall{
			Name:      "sensitive_tool_a",
			Arguments: `{}`,
		},
	}
	tcB := message.ToolCall{
		Index: 1,
		ID:    "call_sensitive_b",
		Type:  "function",
		Function: message.FunctionCall{
			Name:      "sensitive_tool_b",
			Arguments: `{}`,
		},
	}
	reader1 := &testStreamReader{
		msgs: []*message.StreamMessage{
			{Type: message.StreamMessageToolCall, ToolCall: &tcA},
			{Type: message.StreamMessageToolCall, ToolCall: &tcB},
		},
	}

	mockChatModel.EXPECT().Stream(gomock.Any(), gomock.Any(), gomock.Any()).Return(reader1, nil).Times(1)

	userMsgs := []message.Message{
		{Role: message.RoleUser, Content: "Run both sensitive tools"},
	}

	opts := []chat.OptionFunc{
		chat.WithMaxIterations(2),
		chat.WithTools([]tool.Tool{
			sensitiveToolA,
			sensitiveToolB,
		}),
	}

	reader, err := svc.Stream(ctx, userMsgs, opts...)
	require.NoError(t, err)
	defer reader.Close()

	var recvErr error
	for {
		_, err := reader.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			recvErr = err
			break
		}
	}

	require.Error(t, recvErr)
	var interruptErr *session.InterruptError
	assert.True(t, errors.As(recvErr, &interruptErr))
	assert.Equal(t, session.SessionStatusPendingConfirmation, interruptErr.Status)
	assert.Equal(t, session.SessionStatusPendingConfirmation, sess.GetStatus())

	// 核心断言：应该一次性收集两个敏感工具，且 session metadata 包含它们以逗号分割的 ID 与名称列表
	assert.Len(t, interruptErr.ToolCalls, 2)
	assert.Equal(t, "call_sensitive_a", interruptErr.ToolCalls[0].ID)
	assert.Equal(t, "call_sensitive_b", interruptErr.ToolCalls[1].ID)

	assert.Equal(t, "call_sensitive_a,call_sensitive_b", sess.Metadata[session.MetadataPendingConfirmToolCallIDs])
	assert.Equal(t, "sensitive_tool_a,sensitive_tool_b", sess.Metadata[session.MetadataPendingConfirmToolNames])
}

func TestAgentAppService_PartialSuccessPersistenceOnConfirmation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatModel := chatmock.NewMockChatModel(ctrl)
	mockSessionSvc := sessionmock.NewMockSessionService(ctrl)

	eventBus := infraevent.NewInMemoryEventBus(10, 1, nil)
	defer eventBus.Shutdown(context.Background())

	svc := appagent.NewService(mockChatModel, mockSessionSvc, eventBus, eventBus)

	ctx := context.Background()
	sessionID := "test-session-partial-success"
	userID := "user-confirm"

	ctx = appagent.WithSessionID(ctx, sessionID)
	ctx = appagent.WithUserID(ctx, userID)

	sess := session.NewSession(sessionID, userID, nil)
	mockSessionSvc.EXPECT().Get(gomock.Any(), sessionID).Return(sess, nil).Times(1)
	mockSessionSvc.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// 敏感工具
	sensitiveTool := &dummyTool{
		info: tool.Definition{
			Name:                 "sensitive_tool",
			RequiresConfirmation: true,
		},
		execute: func(ctx context.Context, args string) (string, error) {
			return "sensitive_result", nil
		},
	}
	// 普通工具
	normalTool := &dummyTool{
		info: tool.Definition{
			Name: "normal_tool",
		},
		execute: func(ctx context.Context, args string) (string, error) {
			return "normal_success_content", nil
		},
	}

	tcSensitive := message.ToolCall{
		Index: 0,
		ID:    "call_sensitive_3",
		Type:  "function",
		Function: message.FunctionCall{
			Name:      "sensitive_tool",
			Arguments: `{}`,
		},
	}
	tcNormal := message.ToolCall{
		Index: 1,
		ID:    "call_normal_1",
		Type:  "function",
		Function: message.FunctionCall{
			Name:      "normal_tool",
			Arguments: `{}`,
		},
	}
	reader1 := &testStreamReader{
		msgs: []*message.StreamMessage{
			{Type: message.StreamMessageToolCall, ToolCall: &tcSensitive},
			{Type: message.StreamMessageToolCall, ToolCall: &tcNormal},
		},
	}

	mockChatModel.EXPECT().Stream(gomock.Any(), gomock.Any(), gomock.Any()).Return(reader1, nil).Times(1)

	userMsgs := []message.Message{
		{Role: message.RoleUser, Content: "Run both sensitive and normal tools"},
	}

	opts := []chat.OptionFunc{
		chat.WithMaxIterations(2),
		chat.WithTools([]tool.Tool{
			sensitiveTool,
			normalTool,
		}),
	}

	reader, err := svc.Stream(ctx, userMsgs, opts...)
	require.NoError(t, err)
	defer reader.Close()

	var recvErr error
	for {
		_, err := reader.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			recvErr = err
			break
		}
	}

	require.Error(t, recvErr)
	var interruptErr *session.InterruptError
	assert.True(t, errors.As(recvErr, &interruptErr))
	assert.Equal(t, session.SessionStatusPendingConfirmation, interruptErr.Status)
	assert.Equal(t, session.SessionStatusPendingConfirmation, sess.GetStatus())

	// 核心断言：验证因为禁用了 failFast，正常并发执行的工具的消息结果也成功追加并持久化到了消息列表中
	assert.Len(t, sess.Messages, 3)
	assert.Equal(t, message.RoleUser, sess.Messages[0].Role)
	assert.Equal(t, message.RoleAssistant, sess.Messages[1].Role)
	assert.Equal(t, message.RoleTool, sess.Messages[2].Role)
	assert.Equal(t, "call_normal_1", sess.Messages[2].ToolCallID)
	assert.Equal(t, "normal_success_content", sess.Messages[2].Content)
}

func TestAgentAppService_Stream_AutoCancelPendingConfirmation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatModel := chatmock.NewMockChatModel(ctrl)
	mockSessionSvc := sessionmock.NewMockSessionService(ctrl)

	eventBus := infraevent.NewInMemoryEventBus(10, 1, nil)
	defer eventBus.Shutdown(context.Background())

	svc := appagent.NewService(mockChatModel, mockSessionSvc, eventBus, eventBus)

	ctx := context.Background()
	sessionID := "test-session-auto-cancel"
	userID := "user-123"

	ctx = appagent.WithSessionID(ctx, sessionID)
	ctx = appagent.WithUserID(ctx, userID)

	// 1. 初始化一个处于 pending_confirmation 状态的 Session
	sess := session.NewSession(sessionID, userID, nil)
	sess.MarkPendingConfirmation()
	sess.Metadata[session.MetadataPendingConfirmToolCallIDs] = "call_1,call_2"
	sess.Metadata[session.MetadataPendingConfirmToolNames] = "sensitive_tool_1,sensitive_tool_2"

	// 2. 模拟 Session 的读取与保存行为
	mockSessionSvc.EXPECT().Get(gomock.Any(), sessionID).Return(sess, nil).Times(1)
	mockSessionSvc.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// 3. 模拟 ChatModel 的 Stream 方法被调用
	streamMsgs := []*message.StreamMessage{
		{Type: message.StreamMessageTextDelta, Content: "Continuing conversation..."},
	}
	mockReader := &testStreamReader{msgs: streamMsgs}
	mockChatModel.EXPECT().Stream(gomock.Any(), gomock.Any(), gomock.Any()).Return(mockReader, nil).Times(1)

	// 4. 发送新消息
	userMsgs := []message.Message{
		{Role: message.RoleUser, Content: "Never mind, do something else"},
	}

	opts := []chat.OptionFunc{
		chat.WithMaxIterations(1),
	}

	reader, err := svc.Stream(ctx, userMsgs, opts...)
	require.NoError(t, err)
	defer reader.Close()

	// 消费 stream 以让循环走完并更新/保存 Session
	for {
		_, err := reader.Recv()
		if err != nil {
			break
		}
	}

	// 5. 校验自动取消的行为
	// 期望追加了两条取消的 Tool 消息以及用户的新消息
	// 顺序应该是：
	// - ToolMessage(call_1, cancelled)
	// - ToolMessage(call_2, cancelled)
	// - UserMessage("Never mind, do something else")
	// - AssistantMessage("Continuing conversation...")
	require.Len(t, sess.Messages, 4)

	assert.Equal(t, message.RoleTool, sess.Messages[0].Role)
	assert.Equal(t, "call_1", sess.Messages[0].ToolCallID)
	assert.Contains(t, sess.Messages[0].Content, "cancelled")

	assert.Equal(t, message.RoleTool, sess.Messages[1].Role)
	assert.Equal(t, "call_2", sess.Messages[1].ToolCallID)
	assert.Contains(t, sess.Messages[1].Content, "cancelled")

	assert.Equal(t, message.RoleUser, sess.Messages[2].Role)
	assert.Equal(t, "Never mind, do something else", sess.Messages[2].Content)

	assert.Equal(t, message.RoleAssistant, sess.Messages[3].Role)

	// 状态和元数据应该被清空
	assert.Empty(t, sess.GetStatus())
	assert.NotContains(t, sess.Metadata, session.MetadataPendingConfirmToolCallIDs)
	assert.NotContains(t, sess.Metadata, session.MetadataPendingConfirmToolNames)
}


