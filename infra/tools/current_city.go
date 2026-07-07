package tools

import (
	"context"
	"vine-agent/domain/tool"
)

type CurrentCityTool struct{}

// NewCurrentCityTool 创建 CurrentCityTool 实例
func NewCurrentCityTool() tool.Tool {
	return &CurrentCityTool{}
}

func (t *CurrentCityTool) Info() tool.Definition {
	return tool.Definition{
		Name:        "get_current_city",
		Description: "获取用户当前所在的城市",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
			"required":   []any{},
		},
		RequiresConfirmation: false,
	}
}

func (t *CurrentCityTool) Execute(ctx context.Context, args string) (string, error) {
	return "【工具反馈】当前城市：杭州，省份：浙江，国家：中国，时区：Asia/Shanghai。", nil
}
