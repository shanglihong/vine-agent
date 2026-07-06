package agent

import (
	"context"
)

type contextKey string

const (
	sessionIDKey            contextKey = "session_id"
	userIDKey               contextKey = "user_id"
	confirmedToolCallIDsKey contextKey = "confirmed_tool_call_ids"
)

// WithConfirmedToolCallIDs 将已确认的 ToolCall ID 列表注入 context
func WithConfirmedToolCallIDs(ctx context.Context, ids []string) context.Context {
	m := make(map[string]bool)
	for _, id := range ids {
		m[id] = true
	}
	return context.WithValue(ctx, confirmedToolCallIDsKey, m)
}

// IsToolCallConfirmed 检查特定 ToolCall ID 是否已被确认
func IsToolCallConfirmed(ctx context.Context, id string) bool {
	val, ok := ctx.Value(confirmedToolCallIDsKey).(map[string]bool)
	if !ok {
		return false
	}
	return val[id]
}

// WithSessionID 将 SessionID 注入 context
func WithSessionID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, sessionIDKey, id)
}

// GetSessionID 从 context 提取 SessionID
func GetSessionID(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(sessionIDKey).(string)
	return val, ok
}

// WithUserID 将 UserID 注入 context
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// GetUserID 从 context 提取 UserID，若获取不到则默认返回 "default_user"
func GetUserID(ctx context.Context) string {
	val, ok := ctx.Value(userIDKey).(string)
	if !ok || val == "" {
		return "default_user"
	}
	return val
}
