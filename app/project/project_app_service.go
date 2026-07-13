package project

import (
	"context"

	"vine-agent/domain/memory/session"
	"vine-agent/domain/project"
)

// ProjectAppService 项目应用层服务，负责编排项目相关的业务用例
type ProjectAppService struct {
	projectSvc project.ProjectService
}

// NewProjectAppService 构造 ProjectAppService 实例
func NewProjectAppService(projectSvc project.ProjectService) *ProjectAppService {
	return &ProjectAppService{projectSvc: projectSvc}
}

// CreateProject 编排创建项目
func (a *ProjectAppService) CreateProject(ctx context.Context, userID, name, desc string, metadata map[string]string) (*project.Project, error) {
	return a.projectSvc.CreateProject(ctx, userID, name, desc, metadata)
}

// GetProject 获取项目详情
func (a *ProjectAppService) GetProject(ctx context.Context, id string) (*project.Project, error) {
	return a.projectSvc.GetProject(ctx, id)
}

// UpdateProject 更新项目信息
func (a *ProjectAppService) UpdateProject(ctx context.Context, id, name, desc string, metadata map[string]string) (*project.Project, error) {
	return a.projectSvc.UpdateProject(ctx, id, name, desc, metadata)
}

// DeleteProject 删除项目（物理级联删除属于该项目的所有会话与消息）
func (a *ProjectAppService) DeleteProject(ctx context.Context, id string) error {
	return a.projectSvc.DeleteProject(ctx, id)
}

// ListProjects 获取用户的所有项目列表
func (a *ProjectAppService) ListProjects(ctx context.Context, userID string) ([]*project.Project, error) {
	return a.projectSvc.ListProjects(ctx, userID)
}

// BindSession 创建会话后的关系绑定
func (a *ProjectAppService) BindSession(ctx context.Context, projectID string, sessionID string) error {
	return a.projectSvc.BindSession(ctx, projectID, sessionID)
}

// ListSessionsByProject 获取某项目关联的所有会话实体列表
func (a *ProjectAppService) ListSessionsByProject(ctx context.Context, projectID string) ([]*session.Session, error) {
	return a.projectSvc.ListSessionsByProject(ctx, projectID)
}

// ListUnclassifiedSessions 获取某用户下未分类的会话实体列表
func (a *ProjectAppService) ListUnclassifiedSessions(ctx context.Context, userID string) ([]*session.Session, error) {
	return a.projectSvc.ListUnclassifiedSessions(ctx, userID)
}

// DeleteSessionInProject 删除会话
func (a *ProjectAppService) DeleteSessionInProject(ctx context.Context, sessID, projectID string) error {
	return a.projectSvc.DeleteSessionInProject(ctx, sessID, projectID)
}

// GetProjectBySession 根据 sessionID 获取关联的项目。如果未关联任何项目，返回 ErrProjectNotFound
func (a *ProjectAppService) GetProjectBySession(ctx context.Context, sessionID string) (*project.Project, error) {
	return a.projectSvc.GetProjectBySession(ctx, sessionID)
}

// ListSessions 封装 Query 查询过滤与未分类条件分支选择
func (a *ProjectAppService) ListSessions(ctx context.Context, userID, projectID string) ([]*project.Project, error) {
	if projectID == "" {
		return a.projectSvc.ListProjects(ctx, userID)
	}
	p, err := a.projectSvc.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return []*project.Project{p}, nil
}
