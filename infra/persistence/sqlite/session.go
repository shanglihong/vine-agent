package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"vine-agent/domain/memory/session"
	"vine-agent/domain/message"
)

// SessionStore 提供对 SQLite sessions 数据库的具体存取操作
type SessionStore struct {
	db *sql.DB
}

// NewSessionStore 创建一个基于 SQLite 的 Session 仓储实现，接收 dbPath 注入
func NewSessionStore(dbPath string) (*SessionStore, error) {
	db, err := getMemoryDB(dbPath)
	if err != nil {
		return nil, err
	}
	return &SessionStore{db: db}, nil
}

func newSessionStoreWithDB(db *sql.DB) *SessionStore {
	return &SessionStore{db: db}
}

// ==================== Session 仓储实现 (memory.SessionRepository) ====================

// Save 保存或更新 Session 领域对象
func (r *SessionStore) Save(ctx context.Context, sess *session.Session) error {
	if sess == nil {
		return fmt.Errorf("session cannot be nil")
	}

	metadataJSON, err := json.Marshal(sess.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata for session %s: %w", sess.ID, err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin save transaction for session %s: %w", sess.ID, err)
	}
	defer tx.Rollback()

	// 如果消息没有变动，先尝试做部分更新 UPDATE，避免传输和解析大数据量的 messages 字段
	if !sess.IsMessagesDirty() {
		updateQuery := `
		UPDATE sessions SET
			user_id = ?,
			name = ?,
			metadata = ?,
			updated_at = ?
		WHERE id = ?
		`
		result, err := tx.ExecContext(ctx, updateQuery,
			sess.UserID,
			sess.Name,
			string(metadataJSON),
			sess.UpdatedAt,
			sess.ID,
		)
		if err == nil {
			rows, _ := result.RowsAffected()
			if rows > 0 {
				if err := tx.Commit(); err != nil {
					return fmt.Errorf("failed to commit save session transaction: %w", err)
				}
				sess.InitMessagesSnapshot()
				return nil
			}
		}
	}

	messagesJSON, err := json.Marshal(sess.Messages)
	if err != nil {
		return fmt.Errorf("failed to marshal messages for session %s: %w", sess.ID, err)
	}

	query := `
	INSERT INTO sessions (id, user_id, name, messages, metadata, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		user_id = excluded.user_id,
		name = excluded.name,
		messages = excluded.messages,
		metadata = excluded.metadata,
		updated_at = excluded.updated_at
	`
	_, err = tx.ExecContext(ctx, query,
		sess.ID,
		sess.UserID,
		sess.Name,
		string(messagesJSON),
		string(metadataJSON),
		sess.CreatedAt,
		sess.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to execute save session query: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit save session transaction: %w", err)
	}

	// 保存成功后更新快照指纹
	sess.InitMessagesSnapshot()
	return nil
}

// Get 根据 ID 获取 Session 领域对象
func (r *SessionStore) Get(ctx context.Context, id string) (*session.Session, error) {
	query := `SELECT id, user_id, name, messages, metadata, created_at, updated_at FROM sessions WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	var (
		sessionID    string
		userID       string
		name         string
		messagesText string
		metadataText string
		createdAt    time.Time
		updatedAt    time.Time
	)

	err := row.Scan(&sessionID, &userID, &name, &messagesText, &metadataText, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, session.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to query session %s: %w", id, err)
	}

	var messages []message.Message
	if err := json.Unmarshal([]byte(messagesText), &messages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages for session %s: %w", id, err)
	}

	var metadata map[string]string
	if err := json.Unmarshal([]byte(metadataText), &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata for session %s: %w", id, err)
	}

	sess := &session.Session{
		ID:        sessionID,
		UserID:    userID,
		Name:      name,
		Messages:  messages,
		Metadata:  metadata,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
	sess.InitMessagesSnapshot()
	return sess, nil
}

// Delete 根据 ID 删除 Session 领域对象
func (r *SessionStore) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM sessions WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete session %s: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for delete session %s: %w", id, err)
	}

	if rowsAffected == 0 {
		return session.ErrSessionNotFound
	}

	return nil
}

// List 根据 UserID 列出该用户的所有会话，列表不携带历史消息详情
func (r *SessionStore) List(ctx context.Context, userID string) ([]*session.Session, error) {
	query := `SELECT id, user_id, name, metadata, created_at, updated_at FROM sessions WHERE user_id = ? ORDER BY updated_at DESC`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query session list for user %s: %w", userID, err)
	}
	defer rows.Close()

	var sessions []*session.Session
	for rows.Next() {
		var (
			sessionID    string
			uID          string
			name         string
			metadataText string
			createdAt    time.Time
			updatedAt    time.Time
		)

		err := rows.Scan(&sessionID, &uID, &name, &metadataText, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session row: %w", err)
		}

		var metadata map[string]string
		if err := json.Unmarshal([]byte(metadataText), &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata for session list: %w", err)
		}

		sessions = append(sessions, &session.Session{
			ID:        sessionID,
			UserID:    uID,
			Name:      name,
			Metadata:  metadata,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			Messages:  nil, // 列表不携带详细消息列表
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during session rows iteration: %w", err)
	}

	return sessions, nil
}

// ListUpdatedSince 列出在指定时间点之后更新过的所有会话，列表不携带历史消息详情
func (r *SessionStore) ListUpdatedSince(ctx context.Context, since time.Time) ([]*session.Session, error) {
	query := `SELECT id, user_id, name, metadata, created_at, updated_at FROM sessions WHERE updated_at >= ? ORDER BY updated_at DESC`
	rows, err := r.db.QueryContext(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query session list updated since %v: %w", since, err)
	}
	defer rows.Close()

	var sessions []*session.Session
	for rows.Next() {
		var (
			sessionID    string
			uID          string
			name         string
			metadataText string
			createdAt    time.Time
			updatedAt    time.Time
		)

		err := rows.Scan(&sessionID, &uID, &name, &metadataText, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session row: %w", err)
		}

		var metadata map[string]string
		if err := json.Unmarshal([]byte(metadataText), &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata for session list: %w", err)
		}

		sessions = append(sessions, &session.Session{
			ID:        sessionID,
			UserID:    uID,
			Name:      name,
			Metadata:  metadata,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			Messages:  nil, // 列表不携带详细消息列表
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during session rows iteration: %w", err)
	}

	return sessions, nil
}

// GetBatch 批量获取 Session 实体
func (r *SessionStore) GetBatch(ctx context.Context, ids []string) (map[string]*session.Session, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(
		`SELECT id, user_id, name, messages, metadata, created_at, updated_at FROM sessions WHERE id IN (%s)`,
		strings.Join(placeholders, ","),
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query batch sessions: %w", err)
	}
	defer rows.Close()

	res := make(map[string]*session.Session)
	var batchErrs []error

	for rows.Next() {
		var (
			sessionID    string
			userID       string
			name         string
			messagesText string
			metadataText string
			createdAt    time.Time
			updatedAt    time.Time
		)
		err := rows.Scan(&sessionID, &userID, &name, &messagesText, &metadataText, &createdAt, &updatedAt)
		if err != nil {
			batchErrs = append(batchErrs, fmt.Errorf("failed to scan session row: %w", err))
			continue
		}

		var messages []message.Message
		if err := json.Unmarshal([]byte(messagesText), &messages); err != nil {
			batchErrs = append(batchErrs, fmt.Errorf("failed to unmarshal messages for session %s: %w", sessionID, err))
			continue
		}

		var metadata map[string]string
		if err := json.Unmarshal([]byte(metadataText), &metadata); err != nil {
			batchErrs = append(batchErrs, fmt.Errorf("failed to unmarshal metadata for session %s: %w", sessionID, err))
			continue
		}

		sess := &session.Session{
			ID:        sessionID,
			UserID:    userID,
			Name:      name,
			Messages:  messages,
			Metadata:  metadata,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}
		sess.InitMessagesSnapshot()
		res[sessionID] = sess
	}

	if err := rows.Err(); err != nil {
		batchErrs = append(batchErrs, fmt.Errorf("rows error: %w", err))
	}

	var joinedErr error
	if len(batchErrs) > 0 {
		joinedErr = errors.Join(batchErrs...)
	}

	return res, joinedErr
}
