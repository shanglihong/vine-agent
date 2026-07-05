package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"vine-agent/domain/chat"
	"vine-agent/domain/memory/session"
	"vine-agent/domain/message"
	"vine-agent/domain/tool"
)

// AgentAppService 智能体应用层服务，负责编排模型生成、会话状态持久化与工具调用循环
type AgentAppService struct {
	chatModel  chat.ChatModel
	sessionSvc session.SessionService
}

// NewAgentAppService 创建一个新的 AgentAppService 实例
func NewAgentAppService(chatModel chat.ChatModel, sessionSvc session.SessionService) *AgentAppService {
	return &AgentAppService{
		chatModel:  chatModel,
		sessionSvc: sessionSvc,
	}
}

// defaultOption 构造默认的 Agent 配置选项
func defaultOption() *Option {
	return &Option{
		MaxIterations: 5,
	}
}

// Run 运行智能体（同步非流式）
func (s *AgentAppService) Run(ctx context.Context, sessionID string, userMsg *message.Message, opts ...OptionFunc) (*message.Message, error) {
	opt := defaultOption()
	for _, fn := range opts {
		fn(opt)
	}

	chatOpts := buildChatOptions(opt)

	sess, err := s.sessionSvc.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if userMsg != nil {
		sess.Messages = append(sess.Messages, *userMsg)
		if err := s.sessionSvc.Save(ctx, sess); err != nil {
			return nil, err
		}
	}

	// 检查会话是否由于未决的确认而处于挂起状态
	if sess.GetStatus() == session.SessionStatusPendingConfirmation {
		var lastAssistant *message.Message
		for i := len(sess.Messages) - 1; i >= 0; i-- {
			if sess.Messages[i].IsAssistant() {
				lastAssistant = &sess.Messages[i]
				break
			}
		}
		return nil, &session.InterruptError{
			SessionID: sess.ID,
			Status:    session.SessionStatusPendingConfirmation,
			Message:   lastAssistant,
		}
	}

	for iter := 0; iter < opt.MaxIterations; iter++ {
		assistantMsg, err := s.chatModel.Generate(ctx, sess.Messages, chatOpts...)
		if err != nil {
			return nil, err
		}

		sess.Messages = append(sess.Messages, *assistantMsg)
		if err := s.sessionSvc.Save(ctx, sess); err != nil {
			return nil, err
		}

		if !assistantMsg.HasToolCalls() {
			return assistantMsg, nil
		}

		var hasConfirmation bool
		for _, tc := range assistantMsg.ToolCalls {
			var matchedTool tool.Tool
			for _, t := range opt.Tools {
				if t.Info().Name == tc.Function.Name {
					matchedTool = t
					break
				}
			}

			if matchedTool == nil {
				toolErr := fmt.Errorf("tool %s not found in options", tc.Function.Name)
				toolMsg := message.Message{
					Role:       message.RoleTool,
					Content:    toolErr.Error(),
					ToolCallID: tc.ID,
				}
				sess.Messages = append(sess.Messages, toolMsg)
				if err := s.sessionSvc.Save(ctx, sess); err != nil {
					return nil, err
				}
				continue
			}

			if matchedTool.Info().RequiresConfirmation {
				hasConfirmation = true
				continue
			}

			output, execErr := matchedTool.Execute(ctx, tc.Function.Arguments)
			toolMsg := message.Message{
				Role:       message.RoleTool,
				ToolCallID: tc.ID,
			}
			if execErr != nil {
				toolMsg.Content = fmt.Sprintf("error executing tool: %s", execErr.Error())
			} else {
				toolMsg.Content = output
			}

			sess.Messages = append(sess.Messages, toolMsg)
			if err := s.sessionSvc.Save(ctx, sess); err != nil {
				return nil, err
			}
		}

		if hasConfirmation {
			sess.MarkPendingConfirmation()
			if err := s.sessionSvc.Save(ctx, sess); err != nil {
				return nil, err
			}
			return nil, &session.InterruptError{
				SessionID: sess.ID,
				Status:    session.SessionStatusPendingConfirmation,
				Message:   assistantMsg,
			}
		}
	}

	return nil, fmt.Errorf("agent reached max iterations (%d) without resolving", opt.MaxIterations)
}

// ConfirmTool 提交人工确认决策，并根据决策执行（或写入拒绝）对应工具，执行后更新会话状态
func (s *AgentAppService) ConfirmTool(ctx context.Context, sessionID string, toolCallID string, approved bool, rejectReason string, opts ...OptionFunc) error {
	sess, err := s.sessionSvc.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	if sess.GetStatus() != session.SessionStatusPendingConfirmation {
		return fmt.Errorf("session is not in pending_confirmation status: %s", sess.GetStatus())
	}

	var lastAssistantIdx = -1
	for i := len(sess.Messages) - 1; i >= 0; i-- {
		if sess.Messages[i].IsAssistant() {
			lastAssistantIdx = i
			break
		}
	}
	if lastAssistantIdx == -1 {
		return fmt.Errorf("no assistant message found in session %s", sessionID)
	}

	lastAssistant := sess.Messages[lastAssistantIdx]
	var targetToolCall *message.ToolCall
	for _, tc := range lastAssistant.ToolCalls {
		if tc.ID == toolCallID {
			targetToolCall = &tc
			break
		}
	}
	if targetToolCall == nil {
		return fmt.Errorf("tool call id %s not found in the last assistant message", toolCallID)
	}

	opt := defaultOption()
	for _, fn := range opts {
		fn(opt)
	}

	var toolMsg message.Message
	toolMsg.Role = message.RoleTool
	toolMsg.ToolCallID = toolCallID

	if approved {
		var matchedTool tool.Tool
		for _, t := range opt.Tools {
			if t.Info().Name == targetToolCall.Function.Name {
				matchedTool = t
				break
			}
		}
		if matchedTool == nil {
			return fmt.Errorf("tool %s not configured in options", targetToolCall.Function.Name)
		}

		output, execErr := matchedTool.Execute(ctx, targetToolCall.Function.Arguments)
		if execErr != nil {
			toolMsg.Content = fmt.Sprintf("error executing tool: %s", execErr.Error())
		} else {
			toolMsg.Content = output
		}
	} else {
		reason := rejectReason
		if reason == "" {
			reason = "rejected by user"
		}
		toolMsg.Content = fmt.Sprintf("rejected: %s", reason)
	}

	sess.Messages = append(sess.Messages, toolMsg)

	// 检查该轮 Assistant 发起的全部工具调用是否都已被反馈
	allConfirmed := true
	for _, tc := range lastAssistant.ToolCalls {
		found := false
		for i := lastAssistantIdx + 1; i < len(sess.Messages); i++ {
			msg := sess.Messages[i]
			if msg.Role == message.RoleTool && msg.ToolCallID == tc.ID {
				found = true
				break
			}
		}
		if !found {
			allConfirmed = false
			break
		}
	}

	if allConfirmed {
		sess.ClearStatus()
	}

	if err := s.sessionSvc.Save(ctx, sess); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

// RunStream 运行智能体（流式输出）
func (s *AgentAppService) RunStream(ctx context.Context, sessionID string, userMsg *message.Message, opts ...OptionFunc) (message.StreamMessageReader, error) {
	opt := defaultOption()
	for _, fn := range opts {
		fn(opt)
	}

	chatOpts := buildChatOptions(opt)

	subCtx, cancel := context.WithCancel(ctx)
	reader := &agentEventReader{
		ch:     make(chan *message.StreamMessage, 100),
		errCh:  make(chan error, 1),
		ctx:    subCtx,
		cancel: cancel,
	}

	go func() {
		defer close(reader.ch)
		err := s.runStreamLoop(subCtx, sessionID, userMsg, opt, chatOpts, reader)
		if err != nil {
			reader.sendErr(err)
		}
	}()

	return reader, nil
}

func (s *AgentAppService) runStreamLoop(
	ctx context.Context,
	sessionID string,
	userMsg *message.Message,
	opt *Option,
	chatOpts []chat.OptionFunc,
	reader *agentEventReader,
) error {
	sess, err := s.sessionSvc.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	if userMsg != nil {
		sess.Messages = append(sess.Messages, *userMsg)
		if err := s.sessionSvc.Save(ctx, sess); err != nil {
			return err
		}
	}

	if sess.GetStatus() == session.SessionStatusPendingConfirmation {
		var lastAssistant *message.Message
		for i := len(sess.Messages) - 1; i >= 0; i-- {
			if sess.Messages[i].IsAssistant() {
				lastAssistant = &sess.Messages[i]
				break
			}
		}
		return &session.InterruptError{
			SessionID: sess.ID,
			Status:    session.SessionStatusPendingConfirmation,
			Message:   lastAssistant,
		}
	}

	for iter := 0; iter < opt.MaxIterations; iter++ {
		stream, err := s.chatModel.Stream(ctx, sess.Messages, chatOpts...)
		if err != nil {
			return err
		}

		var fullContent string
		var fullReasoning string
		var tempToolCalls []message.ToolCall

		for {
			msg, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				_ = stream.Close()
				return err
			}

			if msg.Type == message.StreamMessageTextDelta || msg.Type == message.StreamMessageReasoningDelta {
				reader.send(msg)
			}

			switch msg.Type {
			case message.StreamMessageTextDelta:
				fullContent += msg.Content
			case message.StreamMessageReasoningDelta:
				fullReasoning += msg.Content
			case message.StreamMessageToolCall:
				if msg.ToolCall != nil {
					idx := msg.ToolCall.Index
					for len(tempToolCalls) <= idx {
						tempToolCalls = append(tempToolCalls, message.ToolCall{})
					}
					if msg.ToolCall.ID != "" {
						tempToolCalls[idx].ID = msg.ToolCall.ID
					}
					if msg.ToolCall.Type != "" {
						tempToolCalls[idx].Type = msg.ToolCall.Type
					}
					if msg.ToolCall.Function.Name != "" {
						tempToolCalls[idx].Function.Name = msg.ToolCall.Function.Name
					}
					if msg.ToolCall.Function.Arguments != "" {
						tempToolCalls[idx].Function.Arguments += msg.ToolCall.Function.Arguments
					}
				}
			}
		}
		_ = stream.Close()

		assistantMsg := message.Message{
			Role:             message.RoleAssistant,
			Content:          fullContent,
			ReasoningContent: fullReasoning,
		}
		if len(tempToolCalls) > 0 {
			assistantMsg.ToolCalls = tempToolCalls
		}

		sess.Messages = append(sess.Messages, assistantMsg)
		if err := s.sessionSvc.Save(ctx, sess); err != nil {
			return err
		}

		if !assistantMsg.HasToolCalls() {
			return nil
		}

		var hasConfirmation bool
		for _, tc := range assistantMsg.ToolCalls {
			tcCopy := tc
			reader.send(&message.StreamMessage{
				Type:     message.StreamMessageToolCall,
				ToolCall: &tcCopy,
			})

			var matchedTool tool.Tool
			for _, t := range opt.Tools {
				if t.Info().Name == tc.Function.Name {
					matchedTool = t
					break
				}
			}

			if matchedTool == nil {
				toolErr := fmt.Errorf("tool %s not found in options", tc.Function.Name)
				toolMsg := message.Message{
					Role:       message.RoleTool,
					Content:    toolErr.Error(),
					ToolCallID: tc.ID,
				}
				sess.Messages = append(sess.Messages, toolMsg)
				if err := s.sessionSvc.Save(ctx, sess); err != nil {
					return err
				}
				reader.send(&message.StreamMessage{
					Type: message.StreamMessageToolResult,
					ToolResult: &message.StreamToolResult{
						ToolCallID: tc.ID,
						Error:      toolErr,
					},
				})
				continue
			}

			if matchedTool.Info().RequiresConfirmation {
				hasConfirmation = true
				continue
			}

			output, execErr := matchedTool.Execute(ctx, tc.Function.Arguments)
			toolMsg := message.Message{
				Role:       message.RoleTool,
				ToolCallID: tc.ID,
			}
			var evResult message.StreamToolResult
			evResult.ToolCallID = tc.ID

			if execErr != nil {
				toolMsg.Content = fmt.Sprintf("error executing tool: %s", execErr.Error())
				evResult.Error = execErr
			} else {
				toolMsg.Content = output
				evResult.Output = output
			}

			sess.Messages = append(sess.Messages, toolMsg)
			if err := s.sessionSvc.Save(ctx, sess); err != nil {
				return err
			}

			reader.send(&message.StreamMessage{
				Type:       message.StreamMessageToolResult,
				ToolResult: &evResult,
			})
		}

		if hasConfirmation {
			sess.MarkPendingConfirmation()
			if err := s.sessionSvc.Save(ctx, sess); err != nil {
				return err
			}
			return &session.InterruptError{
				SessionID: sess.ID,
				Status:    session.SessionStatusPendingConfirmation,
				Message:   &assistantMsg,
			}
		}
	}

	return fmt.Errorf("agent reached max iterations (%d) without resolving", opt.MaxIterations)
}

func buildChatOptions(opt *Option) []chat.OptionFunc {
	var chatOpts []chat.OptionFunc
	if opt.Temperature != nil {
		chatOpts = append(chatOpts, chat.WithTemperature(*opt.Temperature))
	}
	if opt.MaxTokens != nil {
		chatOpts = append(chatOpts, chat.WithMaxTokens(*opt.MaxTokens))
	}
	if len(opt.Tools) > 0 {
		chatOpts = append(chatOpts, chat.WithTools(opt.Tools))
	}
	return chatOpts
}

// agentEventReader 实现 message.StreamMessageReader 接口
type agentEventReader struct {
	ch     chan *message.StreamMessage
	errCh  chan error
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
}

func (r *agentEventReader) Recv() (*message.StreamMessage, error) {
	select {
	case ev, ok := <-r.ch:
		if !ok {
			select {
			case err := <-r.errCh:
				if err != nil {
					return nil, err
				}
			default:
			}
			return nil, io.EOF
		}
		return ev, nil
	case err := <-r.errCh:
		if err != nil {
			return nil, err
		}
		return nil, io.EOF
	case <-r.ctx.Done():
		return nil, r.ctx.Err()
	}
}

func (r *agentEventReader) Close() error {
	r.cancel()
	return nil
}

func (r *agentEventReader) send(ev *message.StreamMessage) bool {
	select {
	case r.ch <- ev:
		return true
	case <-r.ctx.Done():
		return false
	}
}

func (r *agentEventReader) sendErr(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	select {
	case r.errCh <- err:
	default:
		// 如果已经写入过错误，不再重复写入
	}
}
