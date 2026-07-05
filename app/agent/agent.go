package agent

import (
	"vine-agent/domain/tool"
)

// Option 定义 Agent 调用的参数配置
type Option struct {
	MaxIterations int
	Tools         []tool.Tool
	Temperature   *float64
	MaxTokens     *int
}

// OptionFunc 用于配置 Options 的函数类型
type OptionFunc func(*Option)

// WithMaxIterations 设定最大迭代次数，防死循环
func WithMaxIterations(max int) OptionFunc {
	return func(o *Option) {
		o.MaxIterations = max
	}
}

// WithTools 设定可供 Agent 调用的工具列表
func WithTools(tools []tool.Tool) OptionFunc {
	return func(o *Option) {
		o.Tools = tools
	}
}

// WithTemperature 设定大模型的采样温度
func WithTemperature(t float64) OptionFunc {
	return func(o *Option) {
		o.Temperature = &t
	}
}

// WithMaxTokens 设定生成最大 token 限制
func WithMaxTokens(m int) OptionFunc {
	return func(o *Option) {
		o.MaxTokens = &m
	}
}
