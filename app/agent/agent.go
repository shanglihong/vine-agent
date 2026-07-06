package agent

import (
	"context"
)

type contextKey string

const (
	sessionIDKey contextKey = "session_id"
	userIDKey    contextKey = "user_id"
)

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
