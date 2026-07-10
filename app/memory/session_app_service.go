package memory

import (
	"context"

	"vine-agent/domain/memory/session"
	"vine-agent/domain/project"
)

// SessionAppService 会话应用层服务，负责编排会话管理及跨领域聚合根的业务用例
type SessionAppService struct {
	sessionSvc session.SessionService
	projectSvc project.ProjectService
}

// NewSessionAppService 构造 SessionAppService 实例
func NewSessionAppService(sessionSvc session.SessionService, projectSvc project.ProjectService) *SessionAppService {
	return &SessionAppService{
		sessionSvc: sessionSvc,
		projectSvc: projectSvc,
	}
}

// CreateSession 编排会话创建以及可选项目分步绑定动作（容忍最终一致性）
func (a *SessionAppService) CreateSession(ctx context.Context, sessionID, userID, projectID string) (*session.Session, error) {
	sess := session.NewSession(sessionID, userID, nil)
	if err := a.sessionSvc.Save(ctx, sess); err != nil {
		return nil, err
	}

	if projectID != "" {
		// 最终一致性绑定，绑定失败仅打印警告但对用户返回成功创建
		if err := a.projectSvc.BindSession(ctx, projectID, sess.ID); err != nil {
			// 在应用层记录或通过领域事件解耦，此处暂时容错，由外层 Handler 统一做 Warn 日志打印
			return sess, err
		}
	}

	return sess, nil
}

// ListSessions 封装 Query 查询过滤与未分类条件分支选择
func (a *SessionAppService) ListSessions(ctx context.Context, userID, projectID string, hasProjectIDQuery bool) ([]*session.Session, error) {
	if hasProjectIDQuery {
		if projectID == "" {
			// 未分类会话
			return a.projectSvc.ListUnclassifiedSessions(ctx, userID)
		}
		// 特定项目关联会话
		return a.projectSvc.ListSessionsByProject(ctx, projectID)
	}

	// 向下兼容，拉取该用户的所有会话
	return a.sessionSvc.List(ctx, userID)
}
