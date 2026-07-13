package router

import (
	"context"
	"vine-agent/cmd/bootstrap"
	"vine-agent/cmd/http/dto"
)

var (
	sessionHandler = SessionHandler{}
)

type SessionHandler struct{}

func GetSessionHandler() *SessionHandler {
	return &sessionHandler
}

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

func (h *SessionHandler) CreateSession(ctx context.Context, req dto.CreateSessReq) (dto.SessResp, error) {
	sess, err := bootstrap.GetAppContainer().SessionAppService.CreateSession(ctx, req.SessionID, req.UserID, req.ProjectID)
	if err != nil {
		return dto.SessResp{}, err
	}
	return dto.SessResp{
		ID:     sess.ID,
		Status: "created",
	}, nil
}

func (h *SessionHandler) GetSessionMessages(ctx context.Context, req dto.SessIdReq) (dto.SessResp, error) {
	sess, err := bootstrap.GetDomainContainer().SessionService.Get(ctx, req.SessionID)
	if err != nil {
		return dto.SessResp{}, err
	}
	return dto.SessResp{
		ID:       sess.ID,
		UserID:   sess.UserID,
		Name:     sess.Name,
		Messages: sess.Messages,
		Status:   sess.GetStatus(),
	}, nil
}

func (h *SessionHandler) DeleteSession(ctx context.Context, req dto.SessIdReq) (dto.SessResp, error) {
	err := bootstrap.GetAppContainer().ProjectAppService.DeleteSessionInProject(ctx, req.SessionID, req.ProjectID)
	if err != nil {
		return dto.SessResp{}, err
	}
	return dto.SessResp{
		ID:     req.SessionID,
		Status: "deleted",
	}, nil
}

func (h *SessionHandler) RenameSession(ctx context.Context, req dto.SessRenameReq) (dto.SessResp, error) {
	err := bootstrap.GetDomainContainer().SessionService.Rename(ctx, req.SessionID, req.Name)
	if err != nil {
		return dto.SessResp{}, err
	}
	return dto.SessResp{
		ID:     req.SessionID,
		Name:   req.Name,
		Status: "renamed",
	}, nil
}
