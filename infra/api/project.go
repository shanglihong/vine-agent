package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"vine-agent/domain/project"
)

// 1. POST /api/projects
func (h *APIHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	if h.setCORS(w, r) {
		return
	}

	var req struct {
		UserID      string            `json:"user_id"`
		Name        string            `json:"name"`
		Path        string            `json:"path"`
		Description string            `json:"description"`
		Metadata    map[string]string `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" || req.Name == "" {
		h.respondError(w, http.StatusBadRequest, "user_id and name are required")
		return
	}

	proj, err := h.projectAppSvc.CreateProject(r.Context(), req.UserID, req.Name, req.Path, req.Description, req.Metadata)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{
		"id":     proj.ID,
		"status": "created",
	})
}

// 2. GET /api/projects?user_id=xxx
func (h *APIHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	if h.setCORS(w, r) {
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		h.respondError(w, http.StatusBadRequest, "missing user_id query parameter")
		return
	}

	list, err := h.projectAppSvc.ListProjects(r.Context(), userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, list)
}

// 3. GET /api/projects/{id}
func (h *APIHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	if h.setCORS(w, r) {
		return
	}

	projectID := r.PathValue("id")
	if projectID == "" {
		h.respondError(w, http.StatusBadRequest, "missing project id in path")
		return
	}

	proj, err := h.projectAppSvc.GetProject(r.Context(), projectID)
	if err != nil {
		if errors.Is(err, project.ErrProjectNotFound) {
			h.respondError(w, http.StatusNotFound, "project not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, proj)
}

// 4. PUT /api/projects/{id}
func (h *APIHandler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	if h.setCORS(w, r) {
		return
	}

	projectID := r.PathValue("id")
	if projectID == "" {
		h.respondError(w, http.StatusBadRequest, "missing project id in path")
		return
	}

	var req struct {
		Name        string            `json:"name"`
		Path        string            `json:"path"`
		Description string            `json:"description"`
		Metadata    map[string]string `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		h.respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	_, err := h.projectAppSvc.UpdateProject(r.Context(), projectID, req.Name, req.Path, req.Description, req.Metadata)
	if err != nil {
		if errors.Is(err, project.ErrProjectNotFound) {
			h.respondError(w, http.StatusNotFound, "project not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// 5. DELETE /api/projects/{id}
func (h *APIHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	if h.setCORS(w, r) {
		return
	}

	projectID := r.PathValue("id")
	if projectID == "" {
		h.respondError(w, http.StatusBadRequest, "missing project id in path")
		return
	}

	err := h.projectAppSvc.DeleteProject(r.Context(), projectID)
	if err != nil {
		if errors.Is(err, project.ErrProjectNotFound) {
			h.respondError(w, http.StatusNotFound, "project not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// 6. GET /api/projects/{id}/sessions
func (h *APIHandler) ListProjectSessions(w http.ResponseWriter, r *http.Request) {
	if h.setCORS(w, r) {
		return
	}

	projectID := r.PathValue("id")
	if projectID == "" {
		h.respondError(w, http.StatusBadRequest, "missing project id in path")
		return
	}

	sessions, err := h.projectAppSvc.ListSessionsByProject(r.Context(), projectID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	type sessResp struct {
		ID        string    `json:"id"`
		UserID    string    `json:"user_id"`
		Name      string    `json:"name"`
		UpdatedAt time.Time `json:"updated_at"`
		Status    string    `json:"status,omitempty"`
	}

	list := make([]sessResp, 0, len(sessions))
	for _, s := range sessions {
		status := ""
		if s.Metadata != nil {
			status = s.Metadata["status"]
		}
		list = append(list, sessResp{
			ID:        s.ID,
			UserID:    s.UserID,
			Name:      s.Name,
			UpdatedAt: s.UpdatedAt,
			Status:    status,
		})
	}

	h.respondJSON(w, http.StatusOK, list)
}
