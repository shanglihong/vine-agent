package tool

import (
	"context"
)

// Tool 统一大模型工具调用的执行与定义契约
type Tool interface {
	// Info 返回工具元数据，包括工具名称、描述以及 JSON Schema 参数约束
	Info() Definition

	// Execute 执行工具的业务逻辑，入参是大模型生成的 JSON 字符串参数，返回执行结果字符串
	Execute(ctx context.Context, args string) (string, error)
}

// Definition 工具元数据定义，供大模型识别其调用参数和行为
type Definition struct {
	Name                 string `json:"name"`
	Description          string `json:"description"`
	Parameters           any    `json:"parameters"` // 通常为 JSON Schema（可以为 map[string]any 或结构体）
	RequiresConfirmation bool   `json:"requires_confirmation,omitempty"` // 是否需要人工确认
}