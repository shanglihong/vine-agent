package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"vine-agent/domain/tool"
)

type WeatherTool struct{}

// NewWeatherTool 创建 WeatherTool 实例
func NewWeatherTool() tool.Tool {
	return &WeatherTool{}
}

func (t *WeatherTool) Info() tool.Definition {
	return tool.Definition{
		Name:        "get_weather",
		Description: "获取指定城市的实时天气",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"location": map[string]any{
					"type":        "string",
					"description": "城市名称，例如 北京, 上海, 杭州",
				},
			},
			"required": []any{"location"},
		},
		RequiresConfirmation: false,
	}
}

func (t *WeatherTool) Execute(ctx context.Context, args string) (string, error) {
	var params struct {
		Location string `json:"location"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", err
	}
	if params.Location == "" {
		params.Location = "本地"
	}
	return fmt.Sprintf("【工具反馈】城市：%s，天气：晴转多云，气温：28℃，PM2.5：35，空气质量优。", params.Location), nil
}
