package agent

import (
	"context"
	"errors"

	"vine-agent/domain/chat"
	"vine-agent/domain/memory/session"
	"vine-agent/domain/message"
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
func (s *interactionService) ReadStream(_ context.Context, reader message.StreamMessageReader) (*StreamResult, error) {
	// TODO: 实现流读取与解析逻辑
	//   1. 循环 Recv() 直到 io.EOF 或 error
	//   2. 累积 StreamMessageTextDelta 为 Content
	//   3. 若 error 为 *session.InterruptError，填充 result.Interrupt，返回 result, nil
	//   4. 其他 error 原样向上透传
	_ = errors.New("") // 消除 errors 未使用的编译报错，待实现时删除
	panic("not implemented")
}

// ResumeStream 将用户已确认的工具调用 ID 注入 ctx，并恢复被中断的流式会话。
// 内部从 sessionSvc 加载当前 Session，校验其处于 pending_confirmation 状态后启动流式循环。
func (s *interactionService) ResumeStream(ctx context.Context, confirmedToolCallIDs []string, opts ...chat.OptionFunc) (message.StreamMessageReader, error) {
	// TODO: 实现中断恢复逻辑
	//   1. 从 ctx 取 SessionID，加载 Session，校验状态为 pending_confirmation
	//   2. 清除 Session 中断状态（ClearStatus）
	//   3. 将 confirmedToolCallIDs 注入 ctx（WithConfirmedToolCallIDs）
	//   4. 调用 agentSvc.Stream(ctx, nil/空消息, opts...) 继续执行
	panic("not implemented")
}
