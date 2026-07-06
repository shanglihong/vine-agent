package session

import (
	"errors"
	"fmt"

	"vine-agent/domain/message"
)

var (
	ErrSessionNotFound = errors.New("session not found")
)

// InterruptError 代表智能体会话对话被中断（如由于需要敏感工具执行的审批，或文本生成被取消）
type InterruptError struct {
	SessionID string
	Status    string
	Message   *message.Message
	ToolCalls []message.ToolCall
}

func (e *InterruptError) Error() string {
	return fmt.Sprintf("chat interrupted for session %s, status: %s", e.SessionID, e.Status)
}

// NewPendingConfirmationError 创建一个新的 InterruptError 实例
func NewPendingConfirmationError(sessionID string, msg *message.Message, toolCalls []message.ToolCall) *InterruptError {
	return &InterruptError{
		SessionID: sessionID,
		Status:    SessionStatusPendingConfirmation,
		Message:   msg,
		ToolCalls: toolCalls,
	}
}
