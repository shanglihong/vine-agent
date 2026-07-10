package project

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"vine-agent/domain/memory/session"
)

type projectService struct {
	repo           ProjectRepository
	sessionSvc     session.SessionService
	projectRootDir string
}

// NewProjectService 构造 ProjectService 领域服务实例，新增 projectRootDir 支持工作区路径自动分配
func NewProjectService(repo ProjectRepository, sessionSvc session.SessionService, projectRootDir string) ProjectService {
	return &projectService{
		repo:           repo,
		sessionSvc:     sessionSvc,
		projectRootDir: projectRootDir,
	}
}

// CreateProject 创建并持久化一个项目，使用的 path 为根目录 + 项目id
func (s *projectService) CreateProject(ctx context.Context, userID, name, desc string, metadata map[string]string) (*Project, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id cannot be empty")
	}
	if name == "" {
		return nil, fmt.Errorf("project name cannot be empty")
	}

	projID := fmt.Sprintf("proj_%d", time.Now().UnixNano())
	finalPath := filepath.Join(s.projectRootDir, projID)

	proj := NewProject(projID, userID, name, finalPath, desc, metadata)
	if err := s.repo.Save(ctx, proj); err != nil {
		return nil, err
	}
	return proj, nil
}

// GetProject 根据 ID 获取项目
func (s *projectService) GetProject(ctx context.Context, id string) (*Project, error) {
	if id == "" {
		return nil, fmt.Errorf("project id cannot be empty")
	}
	return s.repo.Get(ctx, id)
}

// UpdateProject 更新项目的核心字段（工作空间物理路径不可变更）
func (s *projectService) UpdateProject(ctx context.Context, id, name, desc string, metadata map[string]string) (*Project, error) {
	if id == "" {
		return nil, fmt.Errorf("project id cannot be empty")
	}
	if name == "" {
		return nil, fmt.Errorf("project name cannot be empty")
	}

	proj, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	proj.Update(name, proj.Path, desc, metadata)
	if err := s.repo.Save(ctx, proj); err != nil {
		return nil, err
	}
	return proj, nil
}

// DeleteProject 级联物理删除项目及其下的所有会话和消息
func (s *projectService) DeleteProject(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("project id cannot be empty")
	}

	// 1. 获取属于该项目下的所有会话 ID 列表
	sessionIDs, err := s.repo.ListSessionsByProject(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to list sessions for project %s: %w", id, err)
	}

	// 2. 级联调用 sessionSvc 物理删除每个会话（会同时清理消息和 FTS 索引数据）
	for _, sessID := range sessionIDs {
		if err := s.sessionSvc.Delete(ctx, sessID); err != nil {
			return fmt.Errorf("failed to delete session %s for project %s: %w", sessID, id, err)
		}
	}

	// 3. 物理删除项目本身及关系映射记录
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete project %s: %w", id, err)
	}

	return nil
}

// ListProjects 获取用户的所有项目列表
func (s *projectService) ListProjects(ctx context.Context, userID string) ([]*Project, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id cannot be empty")
	}
	return s.repo.List(ctx, userID)
}

// BindSession 关联会话与项目
func (s *projectService) BindSession(ctx context.Context, projectID string, sessionID string) error {
	if projectID == "" || sessionID == "" {
		return fmt.Errorf("projectID and sessionID cannot be empty")
	}
	return s.repo.BindSession(ctx, projectID, sessionID)
}

// ListSessionsByProject 获取某项目下的所有 Session 聚合根实体列表
func (s *projectService) ListSessionsByProject(ctx context.Context, projectID string) ([]*session.Session, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID cannot be empty")
	}

	sessionIDs, err := s.repo.ListSessionsByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	if len(sessionIDs) == 0 {
		return make([]*session.Session, 0), nil
	}

	// 组合调用 sessionSvc 的批量获取接口以装配 Session 实体，不污染 Repository
	sessionsMap, err := s.sessionSvc.GetBatch(ctx, sessionIDs)
	if err != nil {
		return nil, err
	}

	// 按顺序重组列表
	list := make([]*session.Session, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		if sess, ok := sessionsMap[id]; ok {
			list = append(list, sess)
		}
	}
	return list, nil
}

// ListUnclassifiedSessions 获取未归属任何项目的 Session 聚合根实体列表
func (s *projectService) ListUnclassifiedSessions(ctx context.Context, userID string) ([]*session.Session, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id cannot be empty")
	}

	sessionIDs, err := s.repo.ListUnclassifiedSessions(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(sessionIDs) == 0 {
		return make([]*session.Session, 0), nil
	}

	sessionsMap, err := s.sessionSvc.GetBatch(ctx, sessionIDs)
	if err != nil {
		return nil, err
	}

	list := make([]*session.Session, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		if sess, ok := sessionsMap[id]; ok {
			list = append(list, sess)
		}
	}
	return list, nil
}

// DeleteSessionInProject 物理删除某项目下的所有会话和消息
func (s *projectService) DeleteSessionInProject(ctx context.Context, sessID, projectID string) error {
	if sessID == "" {
		return fmt.Errorf("session_id cannot be empty")
	}

	if projectID == "" {
		project, err := s.GetProjectBySession(ctx, sessID)
		if err != nil {
			return fmt.Errorf("failed to get project for session %s: %w", sessID, err)
		}
		projectID = project.ID
	}

	if err := s.sessionSvc.Delete(ctx, sessID); err != nil {
		return fmt.Errorf("failed to delete session %s: %w", sessID, err)
	}
	if err := s.repo.DeleteSessionInProject(ctx, projectID, sessID); err != nil {
		return fmt.Errorf("failed to remove session %s for project %s: %w", sessID, projectID, err)
	}
	return nil
}

// GetProjectBySession 根据 sessionID 获取关联的项目。如果未关联任何项目，返回 ErrProjectNotFound
func (s *projectService) GetProjectBySession(ctx context.Context, sessionID string) (*Project, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID cannot be empty")
	}
	return s.repo.GetProjectBySession(ctx, sessionID)
}
