package router

import (
	"context"
	"vine-agent/cmd/bootstrap"
	"vine-agent/cmd/http/dto"
	"vine-agent/domain/memory/session"
)

type SessionHandler struct{}

func (h *SessionHandler) ListSessions(ctx context.Context, sessReq dto.SessReq) ([]dto.SessResp, error) {
	sessions, err := bootstrap.GetAppContainer().SessionAppService.ListSessions(ctx, sessReq.UserID, sessReq.ProjectID)
	if err != nil {
		return nil, err
	}
	sessionProjMap := make(map[string]string)
	for _, s := range sessions {
		sessionProjMap[s.ID] = sessReq.ProjectID
	}

	list := make([]dto.SessResp, 0, len(sessions))
	for _, s := range sessions {
		projID := sessionProjMap[s.ID]
		list = append(list, dto.SessResp{
			ID:        s.ID,
			UserID:    s.UserID,
			Name:      s.Name,
			UpdatedAt: s.UpdatedAt,
			Status:    s.GetStatus(),
			ProjectID: projID,
		})
	}
	return list, nil
}

func (h *SessionHandler) CreateSession(ctx context.Context, req dto.CreateSessReq) (*session.Session, error) {
	sess, err := bootstrap.GetAppContainer().SessionAppService.CreateSession(ctx, req.SessionID, req.UserID, req.ProjectID)
	return sess, err
}

func (h *SessionHandler) GetSessionMessages(ctx context.Context, req dto.SessIdReq) (*session.Session, error) {
	sess, err := bootstrap.GetDomainContainer().SessionService.Get(ctx, req.SessionID)
	return sess, err
}

func (h *SessionHandler) DeleteSession(ctx context.Context, req dto.SessIdReq) (dto.Null, error) {
	err := bootstrap.GetAppContainer().ProjectAppService.DeleteSessionInProject(ctx, req.SessionID, req.ProjectID)
	return dto.Null{}, err
}

func (h *SessionHandler) RenameSession(ctx context.Context, req dto.SessRenameReq) (dto.Null, error) {
	err := bootstrap.GetDomainContainer().SessionService.Rename(ctx, req.SessionID, req.Name)
	return dto.Null{}, err
}
