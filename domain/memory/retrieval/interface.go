package retrieval

import (
	"context"
	"vine-agent/domain/message"
)

//go:generate go run github.com/golang/mock/mockgen -source=interface.go -destination=./mock/retrieval_mock.go -package=mock

// RetrievalRepository 定义了消息全文检索的物理持久化契约
type RetrievalRepository interface {
	// Save 索引一条消息
	Save(ctx context.Context, sessionID string, userID string, msg message.Message) error

	// SearchSession 在指定会话内进行全文检索，返回匹配的消息实体列表
	SearchSession(ctx context.Context, sessionID string, query string, limit int) ([]message.Message, error)

	// SearchUser 在指定用户的所有会话内进行全文检索，返回匹配的结果封装列表
	SearchUser(ctx context.Context, userID string, query string, limit int) ([]SearchResult, error)

	// DeleteBySession 删除指定会话的所有索引消息
	DeleteBySession(ctx context.Context, sessionID string) error
}

// RetrievalService 定义了消息全文检索的领域服务契约
type RetrievalService interface {
	// IndexMessage 索引单条消息
	IndexMessage(ctx context.Context, sessionID string, userID string, msg message.Message) error

	// SearchSession 在会话内全文检索消息，返回消息实体列表
	SearchSession(ctx context.Context, sessionID string, query string, limit int) ([]message.Message, error)

	// SearchUser 在用户的所有会话内全文检索消息，返回带会话上下文的结果列表
	SearchUser(ctx context.Context, userID string, query string, limit int) ([]SearchResult, error)

	// ClearSession 清理某个会话的所有检索数据
	ClearSession(ctx context.Context, sessionID string) error
}
