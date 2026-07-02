package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"vine-agent/domain/memory/session"
	"vine-agent/domain/message"
	"vine-agent/utils"

	_ "modernc.org/sqlite"
)

var (
	sessionDBOnce sync.Once
	sessionDBConn *sql.DB
	sessionDBErr  error
)

// SessionStore 提供对 SQLite sessions 数据库的具体存取操作
type SessionStore struct {
	db *sql.DB
}

func NewSessionStore() (*SessionStore, error) {
	db, err := getSessionDB()
	if err != nil {
		return nil, err
	}
	return &SessionStore{db: db}, nil
}

func newSessionStoreWithDB(db *sql.DB) *SessionStore {
	return &SessionStore{db: db}
}

// openDatabase 打开指定路径的 SQLite 数据库，并确保其所在的父目录已创建
func openDatabase(path string) (*sql.DB, error) {
	dir := filepath.Dir(path)
	if dir != "." && dir != "/" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory %s: %w", dir, err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}
	return db, nil
}

func getSessionDB() (*sql.DB, error) {
	sessionDBOnce.Do(func() {
		root := utils.FindProjectRoot()
		var dbPath string
		if root != "" {
			dbPath = filepath.Join(root, "data", "memory")
		} else {
			dbPath = "data/memory"
		}
		db, err := openDatabase(dbPath)
		if err != nil {
			sessionDBErr = err
			return
		}

		sessionDBConn = db
	})
	return sessionDBConn, sessionDBErr
}

// ==================== Session 仓储实现 (memory.SessionRepository) ====================

// Save 保存或更新 Session 领域对象
func (r *SessionStore) Save(ctx context.Context, sess *session.Session) error {
	if sess == nil {
		return fmt.Errorf("session cannot be nil")
	}

	messagesJSON, err := json.Marshal(sess.Messages)
	if err != nil {
		return fmt.Errorf("failed to marshal messages for session %s: %w", sess.ID, err)
	}

	metadataJSON, err := json.Marshal(sess.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata for session %s: %w", sess.ID, err)
	}

	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin save transaction for session %s: %w", sess.ID, err)
	}
	defer tx.Rollback()

	query := `
	INSERT INTO sessions (id, user_id, messages, metadata, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		user_id = excluded.user_id,
		messages = excluded.messages,
		metadata = excluded.metadata,
		updated_at = excluded.updated_at
	`
	_, err = tx.ExecContext(ctx, query,
		sess.ID,
		sess.UserID,
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

	return nil
}

// Get 根据 ID 获取 Session 领域对象
func (r *SessionStore) Get(ctx context.Context, id string) (*session.Session, error) {
	query := `SELECT id, user_id, messages, metadata, created_at, updated_at FROM sessions WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	var (
		sessionID    string
		userID       string
		messagesText string
		metadataText string
		createdAt    time.Time
		updatedAt    time.Time
	)

	err := row.Scan(&sessionID, &userID, &messagesText, &metadataText, &createdAt, &updatedAt)
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

	return &session.Session{
		ID:        sessionID,
		UserID:    userID,
		Messages:  messages,
		Metadata:  metadata,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
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
	query := `SELECT id, user_id, metadata, created_at, updated_at FROM sessions WHERE user_id = ? ORDER BY updated_at DESC`
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
			metadataText string
			createdAt    time.Time
			updatedAt    time.Time
		)

		err := rows.Scan(&sessionID, &uID, &metadataText, &createdAt, &updatedAt)
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
