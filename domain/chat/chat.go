package chat

import (
	"context"

	"vine-agent/domain/message"
	"vine-agent/domain/tool"
)

//go:generate go run github.com/golang/mock/mockgen -source=chat.go -destination=./mock/chat_mock.go -package=mock

// ChatModel 统一定义大模型非流式与流式对话的抽象接口
type ChatModel interface {
	// Generate 一次性生成响应 (非流式)
	Generate(ctx context.Context, messages []message.Message, opts ...OptionFunc) (*message.Message, error)

	// Stream 逐步增量生成响应 (流式)
	Stream(ctx context.Context, messages []message.Message, opts ...OptionFunc) (message.StreamMessageReader, error)
}

// Option 通用调用参数
type Option struct {
	Model         string
	Temperature   *float64
	MaxTokens     *int
	Tools         map[string]tool.Tool // 工具库定义
	ToolChoice    any         // 指定工具调用行为
	MaxIterations *int        // 限制多轮工具调用的最大迭代次数
}

// OptionFunc 用于配置 Options 的函数类型
type OptionFunc func(*Option)

// WithModel 设定指定调用的模型覆盖默认模型
func WithModel(modelStr string) OptionFunc {
	return func(o *Option) { o.Model = modelStr }
}

// WithTemperature 设定采样温度
func WithTemperature(t float64) OptionFunc {
	return func(o *Option) { o.Temperature = &t }
}

// WithMaxTokens 设定生成最大 token 限制
func WithMaxTokens(m int) OptionFunc {
	return func(o *Option) { o.MaxTokens = &m }
}

// WithTools 设定可用工具声明
func WithTools(tools []tool.Tool) OptionFunc {
	return func(o *Option) {
		o.Tools = make(map[string]tool.Tool)
		for _, t := range tools {
			if t != nil {
				o.Tools[t.Info().Name] = t
			}
		}
	}
}

// WithToolChoice 设定工具选择行为
func WithToolChoice(toolChoice any) OptionFunc {
	return func(o *Option) { o.ToolChoice = toolChoice }
}

// WithMaxIterations 设定工具调用最大迭代次数
func WithMaxIterations(max int) OptionFunc {
	return func(o *Option) { o.MaxIterations = &max }
}
