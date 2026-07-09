package agent

import (
	"context"
	"errors"
	"io"

	"vine-agent/domain/chat"
	"vine-agent/domain/memory/session"
	"vine-agent/domain/message"
	"vine-agent/utils"
)

var _ InteractionService = (*interactionService)(nil)

// interactionService 实现 InteractionService，依赖 Service 完成底层 Agent 会话驱动
type interactionService struct {
	agentSvc   Service
	sessionSvc session.SessionService
}

// NewInteractionService 构造 InteractionService 实例
func NewInteractionService(agentSvc Service, sessionSvc session.SessionService) InteractionService {
	return &interactionService{
		agentSvc:   agentSvc,
		sessionSvc: sessionSvc,
	}
}

// ReadStream 消费 StreamMessageReader，将文本内容累积拼接，并结构化识别流结束状态。
// 若流以 InterruptError 结束，填充至 StreamResult.Interrupt；其他错误原样返回。
func (s *interactionService) ReadStream(ctx context.Context, reader message.StreamMessageReader) (*StreamResult, error) {
	if reader == nil {
		return nil, errors.New("reader is nil")
	}
	result := &StreamResult{}
	for {
		msg, err := reader.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			var interruptErr *session.InterruptError
			if errors.As(err, &interruptErr) {
				result.Interrupt = interruptErr
				return result, nil
			}
			return nil, err
		}
		if msg != nil && msg.Type == message.StreamMessageTextDelta {
			result.Content += msg.Content
		}
	}
	return result, nil
}

// ResumeStream 将用户已确认的工具调用 ID 注入 ctx，并恢复被中断的流式会话。
// 内部从 sessionSvc 加载当前 Session，校验其处于 pending_confirmation 状态后启动流式循环。
func (s *interactionService) ResumeStream(ctx context.Context, confirmedToolCallIDs []string, opts ...chat.OptionFunc) (message.StreamMessageReader, error) {
	// 1. 从 ctx 获取 SessionID，并加载 Session
	sessionID, ok := GetSessionID(ctx)
	if !ok || sessionID == "" {
		return nil, errors.New("session id not found in context")
	}

	sess, err := s.sessionSvc.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, session.ErrSessionNotFound
	}

	// 2. 校验状态是否为 pending_confirmation
	if !sess.IsPendingConfirmation() {
		return nil, errors.New("session is not in pending_confirmation status")
	}

	// 3. 匹配 toolCallId
	pendingIDs := sess.GetPendingConfirmToolCallIDs()
	if len(pendingIDs) == 0 {
		return nil, errors.New("no pending confirmation tool calls found in session metadata")
	}
	pendingMap := make(map[string]bool)
	for _, id := range pendingIDs {
		pendingMap[id] = true
	}

	// 从后往前查找最近的那个包含 ToolCalls 的 assistantMsg
	var assistantMsg *message.Message
	for i := len(sess.Messages) - 1; i >= 0; i-- {
		msg := sess.Messages[i]
		if msg.Role == message.RoleAssistant && len(msg.ToolCalls) > 0 {
			assistantMsg = &msg
			break
		}
	}
	if assistantMsg == nil {
		return nil, errors.New("cannot find original assistant message with tool calls")
	}

	// 筛选出这次需要处理的 ToolCalls
	var targetToolCalls []message.ToolCall
	for _, tc := range assistantMsg.ToolCalls {
		if pendingMap[tc.ID] {
			targetToolCalls = append(targetToolCalls, tc)
		}
	}
	if len(targetToolCalls) == 0 {
		return nil, errors.New("no matching pending tool calls found in the assistant message")
	}

	// 4. 执行匹配的 tool
	chatOpt := &chat.Option{}
	for _, fn := range opts {
		fn(chatOpt)
	}

	execCtx := WithConfirmedToolCallIDs(ctx, confirmedToolCallIDs)

	toolResults := utils.ParallelMap(execCtx, targetToolCalls, func(taskCtx context.Context, tc message.ToolCall) (message.Message, error) {
		toolMsg, toolErr := ExecuteToolCall(taskCtx, tc, chatOpt.Tools[tc.Function.Name])
		return toolMsg, toolErr
	}, utils.WithFailFast(false))

	// 处理执行结果并分流
	var newPendingConfirms []message.ToolCall
	for _, res := range toolResults {
		if res.Error == nil {
			sess.AppendMessage(res.O)
		} else if _, ok := res.Error.(*ToolConfirmationRequiredError); ok {
			newPendingConfirms = append(newPendingConfirms, res.I)
		} else {
			toolMsg, _ := ConvertToolErrorToMessage(res.I.ID, res.Error)
			sess.AppendMessage(toolMsg)
		}
	}

	// 5. 判断整个 session 是否都处理完了中断
	if len(newPendingConfirms) > 0 {
		// 中断未完全处理完毕，不清除挂起状态，重新计算并写入 metadata，返回 InterruptError
		interruptErr := session.NewPendingConfirmationError(sess.ID, assistantMsg, newPendingConfirms)
		sess.ApplyInterrupt(interruptErr)
		if err := s.sessionSvc.Save(ctx, sess); err != nil {
			return nil, err
		}
		return nil, interruptErr
	}

	// 处理完了所有挂起的中断
	sess.ClearPendingConfirmations()
	if err := s.sessionSvc.Save(ctx, sess); err != nil {
		return nil, err
	}

	// 恢复会话，传入 execCtx（保持 confirmedToolCallIDs 注入）以及空消息，继续流式会话
	return s.agentSvc.Stream(execCtx, nil, opts...)
}
