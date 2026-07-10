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
func (a *ProjectAppService) CreateProject(ctx context.Context, userID, name, path, desc string, metadata map[string]string) (*project.Project, error) {
	return a.projectSvc.CreateProject(ctx, userID, name, path, desc, metadata)
}

// GetProject 获取项目详情
func (a *ProjectAppService) GetProject(ctx context.Context, id string) (*project.Project, error) {
	return a.projectSvc.GetProject(ctx, id)
}

// UpdateProject 更新项目信息
func (a *ProjectAppService) UpdateProject(ctx context.Context, id, name, path, desc string, metadata map[string]string) (*project.Project, error) {
	return a.projectSvc.UpdateProject(ctx, id, name, path, desc, metadata)
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
