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

type deleteUserDataTool struct{}

func (t *deleteUserDataTool) Info() tool.Definition {
	return tool.Definition{
		Name:        "delete_user_data",
		Description: "高危操作：清空该用户的全部偏好和事实长期记忆 (需要人工确认)",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"user_id": map[string]any{
					"type":        "string",
					"description": "要删除数据的用户唯一标识 ID",
				},
			},
			"required": []any{"user_id"},
		},
		RequiresConfirmation: true,
	}
}

func (t *deleteUserDataTool) Execute(ctx context.Context, args string) (string, error) {
	var params struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", err
	}
	return fmt.Sprintf("【工具反馈】已成功执行敏感数据删除指令，用户 ID [%s] 的画像数据已被物理抹除。", params.UserID), nil
}
