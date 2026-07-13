package router

import (
	"context"
	"vine-agent/cmd/bootstrap"
	"vine-agent/cmd/http/dto"

	"vine-agent/domain/project"
)

var (
	projectHandler = ProjectHandler{}
)

type ProjectHandler struct{}

func GetProjectHandler() *ProjectHandler {
	return &projectHandler
}

func (h *ProjectHandler) CreateProject(ctx context.Context, req dto.CreatProjectReq) (dto.ProjectResp, error) {
	proj, err := bootstrap.GetAppContainer().ProjectAppService.CreateProject(ctx, req.UserID, req.Name, req.Description, req.Metadata)
	if err != nil {
		return dto.ProjectResp{}, err
	}
	return dto.ProjectResp{
		ID:     proj.ID,
		Status: "created",
	}, nil
}

func (h *ProjectHandler) ListProjects(ctx context.Context, req dto.ProjectUserIdReq) ([]*project.Project, error) {
	list, err := bootstrap.GetAppContainer().ProjectAppService.ListProjects(ctx, req.UserID)
	return list, err
}

func (h *ProjectHandler) GetProject(ctx context.Context, req dto.ProjectIdReq) (*project.Project, error) {
	proj, err := bootstrap.GetAppContainer().ProjectAppService.GetProject(ctx, req.ProjectId)
	return proj, err
}

func (h *ProjectHandler) UpdateProject(ctx context.Context, req dto.ProjectUpdateReq) (dto.ProjectResp, error) {
	_, err := bootstrap.GetAppContainer().ProjectAppService.UpdateProject(ctx, req.ProjectId, req.Name, req.Description, nil)
	if err != nil {
		return dto.ProjectResp{}, err
	}
	return dto.ProjectResp{
		ID:     req.ProjectId,
		Status: "updated",
	}, nil
}

func (h *ProjectHandler) DeleteProject(ctx context.Context, req dto.ProjectIdReq) (dto.ProjectResp, error) {
	err := bootstrap.GetAppContainer().ProjectAppService.DeleteProject(ctx, req.ProjectId)
	if err != nil {
		return dto.ProjectResp{}, err
	}
	return dto.ProjectResp{
		ID:     req.ProjectId,
		Status: "deleted",
	}, nil
}

func (h *ProjectHandler) ListProjectSessions(ctx context.Context, req dto.ProjectIdReq) ([]dto.SessResp, error) {
	sessions, err := bootstrap.GetAppContainer().ProjectAppService.ListSessionsByProject(ctx, req.ProjectId)
	if err != nil {
		return nil, err
	}

	list := make([]dto.SessResp, 0, len(sessions))
	for _, s := range sessions {
		list = append(list, dto.SessResp{
			ID:        s.ID,
			UserID:    s.UserID,
			Name:      s.Name,
			UpdatedAt: s.UpdatedAt,
			Status:    s.GetStatus(),
		})
	}
	return list, nil
}
