package agent

import (
	"context"

	"vine-agent/domain/chat"
	"vine-agent/domain/memory/session"
	"vine-agent/domain/message"
)

//go:generate go run github.com/golang/mock/mockgen -source=interface.go -destination=./mock/agent_mock.go -package=mock

// Service 智能体应用服务契约接口
type Service interface {
	chat.ChatModel
}

// StreamResult 表示一次流式会话的结构化结果
type StreamResult struct {
	// Content 流式消息中累计拼接的文本内容
	Content string
	// Interrupt 非 nil 时代表会话被中断（如等待工具审批），携带中断详情与待确认工具调用列表
	Interrupt *session.InterruptError
}

// InteractionService 面向用户交互的应用层服务
// 封装流式结果解读、工具审批确认与中断恢复等用户感知操作
type InteractionService interface {
	// ReadStream 消费 StreamMessageReader 并解析为结构化结果。
	// 若流以 InterruptError 结束，则将其填充至 StreamResult.Interrupt，由调用方决定后续确认流程。
	ReadStream(ctx context.Context, reader message.StreamMessageReader) (*StreamResult, error)

	// ResumeStream 携带用户已确认的工具调用 ID 恢复被中断的流式会话。
	// SessionID 从 ctx 中读取（通过 WithSessionID 注入）。
	ResumeStream(ctx context.Context, confirmedToolCallIDs []string, opts ...chat.OptionFunc) (message.StreamMessageReader, error)
}
