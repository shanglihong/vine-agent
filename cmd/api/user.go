package api

import (
	"net/http"
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

	prof, err := h.userAppSvc.GetUserProfile(r.Context(), userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
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

	prof, err := h.userAppSvc.EvolveAndGetProfile(r.Context(), userID, sessionID)
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
