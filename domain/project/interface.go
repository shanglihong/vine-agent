package project

import (
	"context"
	"errors"

	"vine-agent/domain/memory/session"
)

//go:generate go run github.com/golang/mock/mockgen -source=interface.go -destination=./mock/project_mock.go -package=mock

var (
	// ErrProjectNotFound 项目不存在错误
	ErrProjectNotFound = errors.New("project not found")
)

// ProjectRepository 定义了 Project 领域的物理持久化（仓储）接口契约
type ProjectRepository interface {
	// Save 保存或更新完整的 Project 领域对象
	Save(ctx context.Context, proj *Project) error

	// Get 根据 ID 获取 Project 领域对象。如果不存在，返回 ErrProjectNotFound
	Get(ctx context.Context, id string) (*Project, error)

	// Delete 根据 ID 删除 Project 领域对象。如果不存在，返回 ErrProjectNotFound
	Delete(ctx context.Context, id string) error

	// List 根据 UserID 列出该用户的所有项目，列表按更新时间降序排列
	List(ctx context.Context, userID string) ([]*Project, error)

	// BindSession 关联一个会话与项目（在 project_sessions 关系表中插入记录）
	BindSession(ctx context.Context, projectID string, sessionID string) error

	// ListSessionsByProject 根据项目 ID 查找该项目下的所有会话 ID 列表
	ListSessionsByProject(ctx context.Context, projectID string) ([]string, error)

	// ListUnclassifiedSessions 查找属于该用户但未归属任何项目的会话 ID 列表
	ListUnclassifiedSessions(ctx context.Context, userID string) ([]string, error)

	DeleteSessionInProject(ctx context.Context, projectID string, sessionID string) error

	// GetProjectBySession 根据 sessionID 获取关联的项目。如果未关联任何项目，返回 ErrProjectNotFound
	GetProjectBySession(ctx context.Context, sessionID string) (*Project, error)
}

// ProjectService 定义了 Project 领域服务的核心操作契约
type ProjectService interface {
	CreateProject(ctx context.Context, userID, name, desc string, metadata map[string]string) (*Project, error)
	GetProject(ctx context.Context, id string) (*Project, error)
	UpdateProject(ctx context.Context, id, name, desc string, metadata map[string]string) (*Project, error)
	DeleteProject(ctx context.Context, id string) error
	ListProjects(ctx context.Context, userID string) ([]*Project, error)

	// 关系绑定与会话获取（组合调用 session.SessionService 进行数据加载）
	BindSession(ctx context.Context, projectID string, sessionID string) error
	ListSessionsByProject(ctx context.Context, projectID string) ([]*session.Session, error)
	ListUnclassifiedSessions(ctx context.Context, userID string) ([]*session.Session, error)
	DeleteSessionInProject(ctx context.Context, sessID, projectID string) error

	// GetProjectBySession 根据 sessionID 获取关联的项目。如果未关联任何项目，返回 ErrProjectNotFound
	GetProjectBySession(ctx context.Context, sessionID string) (*Project, error)
}
