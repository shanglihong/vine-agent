package api

import (
	"net/http"
	"strings"
)

// 6. GET /api/users/{id}/profile
func (h *APIHandler) GetUserProfile(w http.ResponseWriter, r *http.Request) {
	if h.setCORS(w, r) {
		return
	}
	userID := r.PathValue("id")
	if userID == "" {
		h.respondError(w, http.StatusBadRequest, "missing user_id in path")
		return
	}

	if strings.HasPrefix(userID, "sess_") {
		sess, err := h.sessionSvc.Get(r.Context(), userID)
		if err == nil && sess != nil {
			userID = sess.UserID
		}
	}

	prof, err := h.profileRepo.GetByUserID(r.Context(), userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if prof == nil {
		// 返回空画像
		h.respondJSON(w, http.StatusOK, map[string]any{
			"user_id":     userID,
			"preferences": []string{},
			"facts":       []string{},
		})
		return
	}

	h.respondJSON(w, http.StatusOK, prof)
}

// 7. POST /api/users/{id}/evolve?session_id=xxx
func (h *APIHandler) Evolve(w http.ResponseWriter, r *http.Request) {
	if h.setCORS(w, r) {
		return
	}
	userID := r.PathValue("id")
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		h.respondError(w, http.StatusBadRequest, "session_id is required")
		return
	}

	if userID == "" || strings.HasPrefix(userID, "sess_") {
		sess, err := h.sessionSvc.Get(r.Context(), sessionID)
		if err == nil && sess != nil {
			userID = sess.UserID
		}
	}

	if userID == "" {
		h.respondError(w, http.StatusBadRequest, "user_id is required and could not be inferred from session")
		return
	}

	err := h.evolutionAppSvc.TriggerEvolution(r.Context(), sessionID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 演化完成，拉取最新的 Profile 并返回给前端
	prof, err := h.profileRepo.GetByUserID(r.Context(), userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, prof)
}

// 8. GET /api/user
func (h *APIHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	if h.setCORS(w, r) {
		return
	}
	// TODO 后续考虑使用jwt处理用户登录
	u, err := h.userAppSvc.GetUser(r.Context())
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, u)
}
