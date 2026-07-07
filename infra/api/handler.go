package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"vine-agent/app/agent"
	memory_app "vine-agent/app/memory"
	user_app "vine-agent/app/user"
	"vine-agent/domain/memory/profile"
	"vine-agent/domain/memory/session"
	"vine-agent/domain/tool"
)

// APIHandler 聚合应用层和领域层服务，处理 HTTP API 请求
type APIHandler struct {
	agentSvc        agent.Service
	interactionSvc  agent.InteractionService
	sessionSvc      session.SessionService
	profileRepo     profile.ProfileRepository
	evolutionAppSvc *memory_app.EvolutionAppService
	userAppSvc      *user_app.UserAppService
	logger          *log.Logger
	tools           map[string]tool.Tool
}

// NewAPIHandler 构造 APIHandler
func NewAPIHandler(
	agentSvc agent.Service,
	interactionSvc agent.InteractionService,
	sessionSvc session.SessionService,
	profileRepo profile.ProfileRepository,
	evolutionAppSvc *memory_app.EvolutionAppService,
	userAppSvc *user_app.UserAppService,
	logger *log.Logger,
) *APIHandler {
	tools := map[string]tool.Tool{
		"get_weather":      &getWeatherTool{},
		"delete_user_data": &deleteUserDataTool{},
	}
	return &APIHandler{
		agentSvc:        agentSvc,
		interactionSvc:  interactionSvc,
		sessionSvc:      sessionSvc,
		profileRepo:     profileRepo,
		evolutionAppSvc: evolutionAppSvc,
		userAppSvc:      userAppSvc,
		logger:          logger,
		tools:           tools,
	}
}

// Helper: 统一设置跨域 CORS 响应头
func (h *APIHandler) setCORS(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return true
	}
	return false
}

// Helper: 统一响应 JSON 格式数据
func (h *APIHandler) respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// Helper: 统一响应错误消息
func (h *APIHandler) respondError(w http.ResponseWriter, status int, errMsg string) {
	h.respondJSON(w, status, map[string]string{"error": errMsg})
}

// Helper: SSE 格式数据组装发送
func sendSSEEvent(w io.Writer, eventType string, data any) {
	var payload []byte
	var err error
	if str, ok := data.(string); ok {
		payload, err = json.Marshal(str)
		if err != nil {
			payload = []byte(str)
		}
	} else {
		payload, err = json.Marshal(data)
		if err != nil {
			payload = []byte(fmt.Sprintf(`{"error":"%s"}`, err.Error()))
		}
	}
	_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(payload))
}

// RegisterRoutes 在 http.ServeMux 上注册路由
func (h *APIHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/sessions", h.ListSessions)
	mux.HandleFunc("POST /api/sessions", h.CreateSession)
	mux.HandleFunc("GET /api/sessions/{id}/messages", h.GetSessionMessages)
	mux.HandleFunc("POST /api/sessions/{id}/chat", h.Chat)
	mux.HandleFunc("POST /api/sessions/{id}/confirm", h.Confirm)
	mux.HandleFunc("GET /api/user", h.GetUser)
	mux.HandleFunc("GET /api/users/{id}/profile", h.GetUserProfile)
	mux.HandleFunc("POST /api/users/{id}/evolve", h.Evolve)
}
