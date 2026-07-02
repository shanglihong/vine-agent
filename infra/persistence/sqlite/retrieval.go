package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"vine-agent/domain/memory/retrieval"
	"vine-agent/domain/message"
)

// RetrievalStore 提供对 SQLite messages_fts 虚拟表的相关全文检索存取操作
type RetrievalStore struct {
	db *sql.DB
}

// NewRetrievalStore 创建基于 SQLite 的 RetrievalRepository 仓储实现
func NewRetrievalStore() (*RetrievalStore, error) {
	db, err := getMemoryDB(MemoryDBPath())
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}
	return &RetrievalStore{db: db}, nil
}

// newRetrievalStoreWithDB 供单元测试或测试环境注入 db 使用
func newRetrievalStoreWithDB(db *sql.DB) *RetrievalStore {
	return &RetrievalStore{db: db}
}

// Save 将一条消息存入全文检索索引表
func (r *RetrievalStore) Save(ctx context.Context, sessionID string, userID string, msg message.Message) error {
	query := `INSERT INTO messages_fts (session_id, user_id, role, content) VALUES (?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, sessionID, userID, string(msg.Role), msg.Content)
	if err != nil {
		return fmt.Errorf("failed to save message to fts index: %w", err)
	}
	return nil
}

// SearchSession 检索指定会话内匹配内容的消息列表
func (r *RetrievalStore) SearchSession(ctx context.Context, sessionID string, queryText string, limit int) ([]message.Message, error) {
	sqlQuery := `
		SELECT role, content 
		FROM messages_fts 
		WHERE session_id = ? AND messages_fts MATCH ? 
		LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, sqlQuery, sessionID, queryText, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages in session %s: %w", sessionID, err)
	}
	defer rows.Close()

	var messages []message.Message
	for rows.Next() {
		var roleStr string
		var content string
		if err := rows.Scan(&roleStr, &content); err != nil {
			return nil, fmt.Errorf("failed to scan matching message row: %w", err)
		}
		messages = append(messages, message.Message{
			Role:    message.Role(roleStr),
			Content: content,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading matching message rows: %w", err)
	}

	return messages, nil
}

// SearchUser 跨会话检索指定用户的所有匹配消息，并关联会话上下文
func (r *RetrievalStore) SearchUser(ctx context.Context, userID string, queryText string, limit int) ([]retrieval.SearchResult, error) {
	sqlQuery := `
		SELECT session_id, user_id, role, content 
		FROM messages_fts 
		WHERE user_id = ? AND messages_fts MATCH ? 
		LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, sqlQuery, userID, queryText, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search user messages: %w", err)
	}
	defer rows.Close()

	var results []retrieval.SearchResult
	for rows.Next() {
		var sessionID string
		var uID string
		var roleStr string
		var content string
		if err := rows.Scan(&sessionID, &uID, &roleStr, &content); err != nil {
			return nil, fmt.Errorf("failed to scan user matching message row: %w", err)
		}
		results = append(results, retrieval.SearchResult{
			SessionID: sessionID,
			UserID:    uID,
			Message: message.Message{
				Role:    message.Role(roleStr),
				Content: content,
			},
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading user matching message rows: %w", err)
	}

	return results, nil
}

// DeleteBySession 清除属于该 sessionID 的所有检索数据
func (r *RetrievalStore) DeleteBySession(ctx context.Context, sessionID string) error {
	query := `DELETE FROM messages_fts WHERE session_id = ?`
	_, err := r.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete messages index for session %s: %w", sessionID, err)
	}
	return nil
}
