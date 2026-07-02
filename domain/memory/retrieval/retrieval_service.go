package retrieval

import (
	"context"
	"fmt"
	"vine-agent/domain/message"
)

type retrievalService struct {
	repo RetrievalRepository
}

// NewRetrievalService 创建 RetrievalService 领域服务的实例
func NewRetrievalService(repo RetrievalRepository) RetrievalService {
	return &retrievalService{
		repo: repo,
	}
}

// IndexMessage 索引单条消息
func (s *retrievalService) IndexMessage(ctx context.Context, sessionID string, userID string, msg message.Message) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID cannot be empty")
	}
	if userID == "" {
		return fmt.Errorf("userID cannot be empty")
	}
	if msg.Content == "" {
		return nil // 忽略空内容的记录
	}
	return s.repo.Save(ctx, sessionID, userID, msg)
}

// SearchSession 限制在当前 session 内做消息全文搜索
func (s *retrievalService) SearchSession(ctx context.Context, sessionID string, query string, limit int) ([]message.Message, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID cannot be empty")
	}
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	return s.repo.SearchSession(ctx, sessionID, query, limit)
}

// SearchUser 在该用户所有 session 范围内做跨会话消息全文搜索
func (s *retrievalService) SearchUser(ctx context.Context, userID string, query string, limit int) ([]SearchResult, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	return s.repo.SearchUser(ctx, userID, query, limit)
}

// ClearSession 清除特定 session 的全部索引记录
func (s *retrievalService) ClearSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID cannot be empty")
	}
	return s.repo.DeleteBySession(ctx, sessionID)
}
