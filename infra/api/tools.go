package api

import (
	"context"
	"encoding/json"
	"fmt"
	"vine-agent/domain/tool"
)

// ==========================================
// Mock 工具具体定义
// ==========================================

type getWeatherTool struct{}

func (t *getWeatherTool) Info() tool.Definition {
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

func (t *getWeatherTool) Execute(ctx context.Context, args string) (string, error) {
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

type getCurrentCityTool struct{}

func (t *getCurrentCityTool) Info() tool.Definition {
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

func (t *getCurrentCityTool) Execute(ctx context.Context, args string) (string, error) {
	return "【工具反馈】当前城市：杭州，省份：浙江，国家：中国，时区：Asia/Shanghai。", nil
}
