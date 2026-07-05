package agent_test

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vine-agent/app/agent"
	"vine-agent/domain/chat"
	"vine-agent/domain/memory/session"
	"vine-agent/domain/message"
	"vine-agent/domain/tool"
)

// === Mocks ===

type mockSessionService struct {
	sessions map[string]*session.Session
	mu       sync.Mutex
}

func newMockSessionService() *mockSessionService {
	return &mockSessionService{sessions: make(map[string]*session.Session)}
}

func (m *mockSessionService) Save(ctx context.Context, sess *session.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cloned := &session.Session{
		ID:        sess.ID,
		UserID:    sess.UserID,
		CreatedAt: sess.CreatedAt,
		UpdatedAt: sess.UpdatedAt,
		Metadata:  make(map[string]string),
		Messages:  append([]message.Message(nil), sess.Messages...),
	}
	for k, v := range sess.Metadata {
		cloned.Metadata[k] = v
	}
	m.sessions[sess.ID] = cloned
	return nil
}

func (m *mockSessionService) Get(ctx context.Context, id string) (*session.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sess, ok := m.sessions[id]
	if !ok {
		return nil, session.ErrSessionNotFound
	}
	cloned := &session.Session{
		ID:        sess.ID,
		UserID:    sess.UserID,
		CreatedAt: sess.CreatedAt,
		UpdatedAt: sess.UpdatedAt,
		Metadata:  make(map[string]string),
		Messages:  append([]message.Message(nil), sess.Messages...),
	}
	for k, v := range sess.Metadata {
		cloned.Metadata[k] = v
	}
	return cloned, nil
}

func (m *mockSessionService) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
	return nil
}

func (m *mockSessionService) List(ctx context.Context, userID string) ([]*session.Session, error) {
	return nil, nil
}

type mockChatModel struct {
	generateFn func(messages []message.Message) (*message.Message, error)
	streamFn   func(messages []message.Message) (message.StreamMessageReader, error)
}

func (m *mockChatModel) Generate(ctx context.Context, messages []message.Message, opts ...chat.OptionFunc) (*message.Message, error) {
	if m.generateFn != nil {
		return m.generateFn(messages)
	}
	return nil, errors.New("generate not implemented")
}

func (m *mockChatModel) Stream(ctx context.Context, messages []message.Message, opts ...chat.OptionFunc) (message.StreamMessageReader, error) {
	if m.streamFn != nil {
		return m.streamFn(messages)
	}
	return nil, errors.New("stream not implemented")
}

type mockStreamReader struct {
	chunks []*message.StreamMessage
	idx    int
}

func (r *mockStreamReader) Recv() (*message.StreamMessage, error) {
	if r.idx >= len(r.chunks) {
		return nil, io.EOF
	}
	chunk := r.chunks[r.idx]
	r.idx++
	return chunk, nil
}

func (r *mockStreamReader) Close() error {
	return nil
}

type dummyTool struct {
	name                 string
	requiresConfirmation bool
	executeFn            func(args string) (string, error)
}

func (d *dummyTool) Info() tool.Definition {
	return tool.Definition{
		Name:                 d.name,
		Description:          "dummy tool",
		RequiresConfirmation: d.requiresConfirmation,
	}
}

func (d *dummyTool) Execute(ctx context.Context, args string) (string, error) {
	if d.executeFn != nil {
		return d.executeFn(args)
	}
	return "dummy result", nil
}

// === Tests ===

func TestAgent_Run_NoTools(t *testing.T) {
	sessionSvc := newMockSessionService()
	chatModel := &mockChatModel{}
	svc := agent.NewAgentAppService(chatModel, sessionSvc)

	// 初始化 session
	sess := session.NewSession("sess_01", "user_01", nil)
	require.NoError(t, sessionSvc.Save(context.Background(), sess))

	// Mock 大模型只回复普通消息
	chatModel.generateFn = func(messages []message.Message) (*message.Message, error) {
		return &message.Message{
			Role:    message.RoleAssistant,
			Content: "Hello! How can I help you today?",
		}, nil
	}

	userMsg := &message.Message{Role: message.RoleUser, Content: "hi"}
	res, err := svc.Run(context.Background(), "sess_01", userMsg)
	require.NoError(t, err)
	assert.Equal(t, "Hello! How can I help you today?", res.Content)

	// 验证 session 持久化
	updatedSess, err := sessionSvc.Get(context.Background(), "sess_01")
	require.NoError(t, err)
	require.Len(t, updatedSess.Messages, 2)
	assert.Equal(t, "hi", updatedSess.Messages[0].Content)
	assert.Equal(t, "Hello! How can I help you today?", updatedSess.Messages[1].Content)
}

func TestAgent_Run_AutoToolCall(t *testing.T) {
	sessionSvc := newMockSessionService()
	chatModel := &mockChatModel{}
	svc := agent.NewAgentAppService(chatModel, sessionSvc)

	// 初始化 session
	sess := session.NewSession("sess_02", "user_01", nil)
	require.NoError(t, sessionSvc.Save(context.Background(), sess))

	var t1ExecuteCalled bool
	t1 := &dummyTool{
		name: "get_weather",
		executeFn: func(args string) (string, error) {
			t1ExecuteCalled = true
			assert.Contains(t, args, "Beijing")
			return "Sunny, 25C", nil
		},
	}

	// 流程：
	// 第一步调用 LLM：LLM 发起 ToolCall "get_weather"
	// 自动执行 Tool -> 写入结果
	// 第二步调用 LLM：LLM 返回最终文本
	callCount := 0
	chatModel.generateFn = func(messages []message.Message) (*message.Message, error) {
		callCount++
		if callCount == 1 {
			return &message.Message{
				Role: message.RoleAssistant,
				ToolCalls: []message.ToolCall{
					{
						ID:   "call_weather_01",
						Type: "function",
						Function: message.FunctionCall{
							Name:      "get_weather",
							Arguments: `{"city": "Beijing"}`,
						},
					},
				},
			}, nil
		}
		// 第二步，传入的消息应当包含 UserMsg -> Assistant(ToolCall) -> ToolResponse
		require.Len(t, messages, 3)
		assert.Equal(t, "call_weather_01", messages[2].ToolCallID)
		assert.Equal(t, "Sunny, 25C", messages[2].Content)

		return &message.Message{
			Role:    message.RoleAssistant,
			Content: "The weather in Beijing is Sunny, 25C.",
		}, nil
	}

	userMsg := &message.Message{Role: message.RoleUser, Content: "what is the weather in Beijing?"}
	res, err := svc.Run(context.Background(), "sess_02", userMsg, agent.WithTools([]tool.Tool{t1}))
	require.NoError(t, err)
	assert.Equal(t, "The weather in Beijing is Sunny, 25C.", res.Content)
	assert.True(t, t1ExecuteCalled)
	assert.Equal(t, 2, callCount)

	// 验证最终 Session messages: User -> Assistant(ToolCall) -> ToolResult -> Assistant(Final)
	updatedSess, err := sessionSvc.Get(context.Background(), "sess_02")
	require.NoError(t, err)
	require.Len(t, updatedSess.Messages, 4)
	assert.Equal(t, message.RoleUser, updatedSess.Messages[0].Role)
	assert.Equal(t, message.RoleAssistant, updatedSess.Messages[1].Role)
	assert.Len(t, updatedSess.Messages[1].ToolCalls, 1)
	assert.Equal(t, message.RoleTool, updatedSess.Messages[2].Role)
	assert.Equal(t, "Sunny, 25C", updatedSess.Messages[2].Content)
	assert.Equal(t, message.RoleAssistant, updatedSess.Messages[3].Role)
}

func TestAgent_Run_ConfirmationInterruptAndApprove(t *testing.T) {
	sessionSvc := newMockSessionService()
	chatModel := &mockChatModel{}
	svc := agent.NewAgentAppService(chatModel, sessionSvc)

	sess := session.NewSession("sess_03", "user_01", nil)
	require.NoError(t, sessionSvc.Save(context.Background(), sess))

	var tSecureCalled bool
	tSecure := &dummyTool{
		name:                 "transfer_money",
		requiresConfirmation: true,
		executeFn: func(args string) (string, error) {
			tSecureCalled = true
			return "success_transferred_100", nil
		},
	}

	// 第一次调用，LLM 返回敏感 ToolCall
	chatModel.generateFn = func(messages []message.Message) (*message.Message, error) {
		return &message.Message{
			Role: message.RoleAssistant,
			ToolCalls: []message.ToolCall{
				{
					ID:   "call_transfer_01",
					Type: "function",
					Function: message.FunctionCall{
						Name:      "transfer_money",
						Arguments: `{"amount": 100}`,
					},
				},
			},
		}, nil
	}

	userMsg := &message.Message{Role: message.RoleUser, Content: "transfer 100"}
	_, err := svc.Run(context.Background(), "sess_03", userMsg, agent.WithTools([]tool.Tool{tSecure}))

	// 应该返回 InterruptError 并且 session 状态是 pending_confirmation
	require.Error(t, err)
	var interruptErr *session.InterruptError
	require.True(t, errors.As(err, &interruptErr))
	assert.Equal(t, "sess_03", interruptErr.SessionID)
	assert.Equal(t, session.SessionStatusPendingConfirmation, interruptErr.Status)
	assert.NotNil(t, interruptErr.Message)
	assert.Equal(t, "call_transfer_01", interruptErr.Message.ToolCalls[0].ID)

	updatedSess, err := sessionSvc.Get(context.Background(), "sess_03")
	require.NoError(t, err)
	assert.Equal(t, session.SessionStatusPendingConfirmation, updatedSess.GetStatus())

	// 此时若再次调用 Run 应直接拒绝并返回同样的 InterruptError
	_, err = svc.Run(context.Background(), "sess_03", nil, agent.WithTools([]tool.Tool{tSecure}))
	require.Error(t, err)
	require.True(t, errors.As(err, &interruptErr))

	// 批准执行
	err = svc.ConfirmTool(context.Background(), "sess_03", "call_transfer_01", true, "", agent.WithTools([]tool.Tool{tSecure}))
	require.NoError(t, err)
	assert.True(t, tSecureCalled)

	// 状态应该已清除
	updatedSess, err = sessionSvc.Get(context.Background(), "sess_03")
	require.NoError(t, err)
	assert.Empty(t, updatedSess.GetStatus())
	require.Len(t, updatedSess.Messages, 3) // User -> Assistant(ToolCall) -> ToolResult(success)

	// 再次调用 Run，LLM 返回最终消息
	chatModel.generateFn = func(messages []message.Message) (*message.Message, error) {
		require.Len(t, messages, 3)
		assert.Equal(t, "success_transferred_100", messages[2].Content)
		return &message.Message{
			Role:    message.RoleAssistant,
			Content: "Money transferred successfully.",
		}, nil
	}

	res, err := svc.Run(context.Background(), "sess_03", nil, agent.WithTools([]tool.Tool{tSecure}))
	require.NoError(t, err)
	assert.Equal(t, "Money transferred successfully.", res.Content)
}

func TestAgent_Run_ConfirmationInterruptAndReject(t *testing.T) {
	sessionSvc := newMockSessionService()
	chatModel := &mockChatModel{}
	svc := agent.NewAgentAppService(chatModel, sessionSvc)

	sess := session.NewSession("sess_04", "user_01", nil)
	require.NoError(t, sessionSvc.Save(context.Background(), sess))

	tSecure := &dummyTool{
		name:                 "transfer_money",
		requiresConfirmation: true,
		executeFn: func(args string) (string, error) {
			t.Fatal("should not execute tool on reject")
			return "", nil
		},
	}

	chatModel.generateFn = func(messages []message.Message) (*message.Message, error) {
		return &message.Message{
			Role: message.RoleAssistant,
			ToolCalls: []message.ToolCall{
				{
					ID:   "call_transfer_02",
					Type: "function",
					Function: message.FunctionCall{
						Name:      "transfer_money",
						Arguments: `{"amount": 100}`,
					},
				},
			},
		}, nil
	}

	userMsg := &message.Message{Role: message.RoleUser, Content: "transfer 100"}
	_, err := svc.Run(context.Background(), "sess_04", userMsg, agent.WithTools([]tool.Tool{tSecure}))
	require.Error(t, err)

	// 拒绝执行
	err = svc.ConfirmTool(context.Background(), "sess_04", "call_transfer_02", false, "user cancelled the operation", agent.WithTools([]tool.Tool{tSecure}))
	require.NoError(t, err)

	updatedSess, err := sessionSvc.Get(context.Background(), "sess_04")
	require.NoError(t, err)
	assert.Empty(t, updatedSess.GetStatus())
	require.Len(t, updatedSess.Messages, 3) // User -> Assistant(ToolCall) -> ToolResult(rejected)
	assert.Contains(t, updatedSess.Messages[2].Content, "rejected: user cancelled the operation")

	// 再次调用 Run 恢复
	chatModel.generateFn = func(messages []message.Message) (*message.Message, error) {
		require.Len(t, messages, 3)
		assert.Contains(t, messages[2].Content, "rejected:")
		return &message.Message{
			Role:    message.RoleAssistant,
			Content: "Operation cancelled.",
		}, nil
	}

	res, err := svc.Run(context.Background(), "sess_04", nil, agent.WithTools([]tool.Tool{tSecure}))
	require.NoError(t, err)
	assert.Equal(t, "Operation cancelled.", res.Content)
}

func TestAgent_RunStream(t *testing.T) {
	sessionSvc := newMockSessionService()
	chatModel := &mockChatModel{}
	svc := agent.NewAgentAppService(chatModel, sessionSvc)

	sess := session.NewSession("sess_05", "user_01", nil)
	require.NoError(t, sessionSvc.Save(context.Background(), sess))

	t1 := &dummyTool{
		name: "echo",
		executeFn: func(args string) (string, error) {
			return "echo: " + args, nil
		},
	}

	// 模拟流式生成
	chatModel.streamFn = func(messages []message.Message) (message.StreamMessageReader, error) {
		if len(messages) > 0 && messages[len(messages)-1].Role == message.RoleTool {
			// 第二轮：返回最终普通回复
			return &mockStreamReader{
				chunks: []*message.StreamMessage{
					{
						Type:    message.StreamMessageTextDelta,
						Content: "Done!",
					},
				},
			}, nil
		}

		// 第一轮：返回三个片段：
		// 1. Content delta
		// 2. ToolCall 开始
		// 3. ToolCall 参数拼接
		return &mockStreamReader{
			chunks: []*message.StreamMessage{
				{
					Type:    message.StreamMessageTextDelta,
					Content: "Thinking... ",
				},
				{
					Type: message.StreamMessageToolCall,
					ToolCall: &message.ToolCall{
						Index: 0,
						ID:    "call_echo_01",
						Type:  "function",
						Function: message.FunctionCall{
							Name:      "echo",
							Arguments: `{"val":`,
						},
					},
				},
				{
					Type: message.StreamMessageToolCall,
					ToolCall: &message.ToolCall{
						Index: 0,
						Function: message.FunctionCall{
							Arguments: `"hello"}`,
						},
					},
				},
			},
		}, nil
	}

	userMsg := &message.Message{Role: message.RoleUser, Content: "run echo"}
	reader, err := svc.RunStream(context.Background(), "sess_05", userMsg, agent.WithTools([]tool.Tool{t1}))
	require.NoError(t, err)
	defer reader.Close()

	var events []*message.StreamMessage
	for {
		ev, err := reader.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		events = append(events, ev)
	}

	// 期望收到的消息列表：
	// 1. StreamMessageTextDelta ("Thinking... ")
	// 2. StreamMessageToolCall ("echo", `{"val":"hello"}`)
	// 3. StreamMessageToolResult (echo: {"val":"hello"})
	// 4. StreamMessageTextDelta ("Done!")
	require.NotEmpty(t, events)

	var hasTextDelta, hasToolCall, hasToolResult bool
	for _, ev := range events {
		switch ev.Type {
		case message.StreamMessageTextDelta:
			hasTextDelta = true
			assert.Contains(t, []string{"Thinking... ", "Done!"}, ev.Content)
		case message.StreamMessageToolCall:
			hasToolCall = true
			assert.Equal(t, "call_echo_01", ev.ToolCall.ID)
			assert.Equal(t, "echo", ev.ToolCall.Function.Name)
			assert.Equal(t, `{"val":"hello"}`, ev.ToolCall.Function.Arguments)
		case message.StreamMessageToolResult:
			hasToolResult = true
			assert.Equal(t, "call_echo_01", ev.ToolResult.ToolCallID)
			assert.NoError(t, ev.ToolResult.Error)
			assert.Equal(t, `echo: {"val":"hello"}`, ev.ToolResult.Output)
		}
	}

	assert.True(t, hasTextDelta)
	assert.True(t, hasToolCall)
	assert.True(t, hasToolResult)

	// 验证最终 Session 是否已保存了助理消息和工具结果
	updatedSess, err := sessionSvc.Get(context.Background(), "sess_05")
	require.NoError(t, err)
	require.Len(t, updatedSess.Messages, 4) // User -> Assistant(ToolCall) -> ToolResult -> Assistant(Final)
	assert.Equal(t, "Thinking... ", updatedSess.Messages[1].Content)
	assert.Len(t, updatedSess.Messages[1].ToolCalls, 1)
	assert.Equal(t, `echo: {"val":"hello"}`, updatedSess.Messages[2].Content)
	assert.Equal(t, "Done!", updatedSess.Messages[3].Content)
}

func TestAgent_Run_MaxIterationsDefense(t *testing.T) {
	sessionSvc := newMockSessionService()
	chatModel := &mockChatModel{}
	svc := agent.NewAgentAppService(chatModel, sessionSvc)

	sess := session.NewSession("sess_06", "user_01", nil)
	require.NoError(t, sessionSvc.Save(context.Background(), sess))

	t1 := &dummyTool{name: "loop"}

	// LLM 总是要求调用同一个工具，导致进入循环
	chatModel.generateFn = func(messages []message.Message) (*message.Message, error) {
		return &message.Message{
			Role: message.RoleAssistant,
			ToolCalls: []message.ToolCall{
				{
					ID:   "call_loop_idx",
					Type: "function",
					Function: message.FunctionCall{
						Name:      "loop",
						Arguments: `{}`,
					},
				},
			},
		}, nil
	}

	userMsg := &message.Message{Role: message.RoleUser, Content: "start loop"}
	// 限制 MaxIterations 为 2
	_, err := svc.Run(context.Background(), "sess_06", userMsg, agent.WithTools([]tool.Tool{t1}), agent.WithMaxIterations(2))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max iterations (2)")
}
