package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"vine-agent/domain/project"
)

// ProjectStore 提供对 SQLite projects 数据库的具体存取与映射操作
type ProjectStore struct {
	db *sql.DB
}

// NewProjectStore 创建基于 SQLite 的 Project 仓储实现
func NewProjectStore(dbPath string) (*ProjectStore, error) {
	db, err := getMemoryDB(dbPath)
	if err != nil {
		return nil, err
	}
	return &ProjectStore{db: db}, nil
}

func newProjectStoreWithDB(db *sql.DB) *ProjectStore {
	return &ProjectStore{db: db}
}

// ==================== Project 仓储实现 (project.ProjectRepository) ====================

// Save 保存或更新 Project 领域对象
func (s *ProjectStore) Save(ctx context.Context, proj *project.Project) error {
	if proj == nil {
		return fmt.Errorf("project cannot be nil")
	}

	metadataJSON, err := json.Marshal(proj.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata for project %s: %w", proj.ID, err)
	}

	query := `
	INSERT INTO projects (id, user_id, name, path, description, metadata, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		user_id = excluded.user_id,
		name = excluded.name,
		path = excluded.path,
		description = excluded.description,
		metadata = excluded.metadata,
		updated_at = excluded.updated_at
	`

	_, err = s.db.ExecContext(ctx, query,
		proj.ID,
		proj.UserID,
		proj.Name,
		proj.Path,
		proj.Description,
		string(metadataJSON),
		proj.CreatedAt,
		proj.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save project %s: %w", proj.ID, err)
	}

	return nil
}

// Get 根据 ID 获取 Project 领域对象
func (s *ProjectStore) Get(ctx context.Context, id string) (*project.Project, error) {
	query := `SELECT id, user_id, name, path, description, metadata, created_at, updated_at FROM projects WHERE id = ?`
	row := s.db.QueryRowContext(ctx, query, id)

	var (
		projID       string
		userID       string
		name         string
		path         string
		description  string
		metadataText string
		createdAt    time.Time
		updatedAt    time.Time
	)

	err := row.Scan(&projID, &userID, &name, &path, &description, &metadataText, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, project.ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to query project %s: %w", id, err)
	}

	var metadata map[string]string
	if err := json.Unmarshal([]byte(metadataText), &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata for project %s: %w", id, err)
	}

	return &project.Project{
		ID:          projID,
		UserID:      userID,
		Name:        name,
		Path:        path,
		Description: description,
		Metadata:    metadata,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

// Delete 根据 ID 删除 Project 领域对象，物理级联清除其关系绑定
func (s *ProjectStore) Delete(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start delete project transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. 删除关系关联表中的记录
	_, err = tx.ExecContext(ctx, `DELETE FROM project_sessions WHERE project_id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete project_sessions relationship for project %s: %w", id, err)
	}

	// 2. 删除 projects 物理记录
	res, err := tx.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to execute delete project query for %s: %w", id, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected for delete project %s: %w", id, err)
	}
	if rows == 0 {
		return project.ErrProjectNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit delete project transaction for %s: %w", id, err)
	}

	return nil
}

// List 获取用户的所有项目列表
func (s *ProjectStore) List(ctx context.Context, userID string) ([]*project.Project, error) {
	query := `SELECT id, user_id, name, path, description, metadata, created_at, updated_at FROM projects WHERE user_id = ? ORDER BY updated_at DESC`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query projects list for user %s: %w", userID, err)
	}
	defer rows.Close()

	var list []*project.Project
	for rows.Next() {
		var (
			projID       string
			uID          string
			name         string
			path         string
			description  string
			metadataText string
			createdAt    time.Time
			updatedAt    time.Time
		)

		err := rows.Scan(&projID, &uID, &name, &path, &description, &metadataText, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project row: %w", err)
		}

		var metadata map[string]string
		if err := json.Unmarshal([]byte(metadataText), &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata for project list: %w", err)
		}

		list = append(list, &project.Project{
			ID:          projID,
			UserID:      uID,
			Name:        name,
			Path:        path,
			Description: description,
			Metadata:    metadata,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during project rows iteration: %w", err)
	}

	return list, nil
}

// BindSession 关联会话与项目，先清空会话可能已存在的旧绑定
func (s *ProjectStore) BindSession(ctx context.Context, projectID string, sessionID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start bind transaction: %w", err)
	}
	defer tx.Rollback()

	// 保证单项目绑定：删除该 session_id 现有的任何其他绑定
	_, err = tx.ExecContext(ctx, `DELETE FROM project_sessions WHERE session_id = ?`, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete old project_sessions mapping for session %s: %w", sessionID, err)
	}

	// 插入新绑定关系
	_, err = tx.ExecContext(ctx, `INSERT INTO project_sessions (project_id, session_id) VALUES (?, ?)`, projectID, sessionID)
	if err != nil {
		return fmt.Errorf("failed to insert project_sessions mapping (project_id: %s, session_id: %s): %w", projectID, sessionID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit bind transaction (project_id: %s, session_id: %s): %w", projectID, sessionID, err)
	}

	return nil
}

// ListSessionsByProject 获取某项目关联的所有会话 ID 列表
func (s *ProjectStore) ListSessionsByProject(ctx context.Context, projectID string) ([]string, error) {
	query := `SELECT session_id FROM project_sessions WHERE project_id = ?`
	rows, err := s.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions for project %s: %w", projectID, err)
	}
	defer rows.Close()

	var sessionIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan session_id row: %w", err)
		}
		sessionIDs = append(sessionIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during session IDs rows iteration: %w", err)
	}

	return sessionIDs, nil
}

// ListUnclassifiedSessions 获取某用户下所有未分类（未绑定任何项目）的会话 ID 列表
func (s *ProjectStore) ListUnclassifiedSessions(ctx context.Context, userID string) ([]string, error) {
	query := `
	SELECT id FROM sessions 
	WHERE user_id = ? AND id NOT IN (SELECT session_id FROM project_sessions) 
	ORDER BY updated_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query unclassified sessions for user %s: %w", userID, err)
	}
	defer rows.Close()

	var sessionIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan unclassified session_id row: %w", err)
		}
		sessionIDs = append(sessionIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during unclassified session rows iteration: %w", err)
	}

	return sessionIDs, nil
}
