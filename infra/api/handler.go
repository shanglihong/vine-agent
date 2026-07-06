package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"vine-agent/app/agent"
	memory_app "vine-agent/app/memory"
	"vine-agent/domain/chat"
	"vine-agent/domain/memory/profile"
	"vine-agent/domain/memory/session"
	"vine-agent/domain/message"
	"vine-agent/domain/tool"
)

// APIHandler 聚合应用层和领域层服务，处理 HTTP API 请求
type APIHandler struct {
	agentSvc        agent.Service
	interactionSvc  agent.InteractionService
	sessionSvc      session.SessionService
	profileRepo     profile.ProfileRepository
	evolutionAppSvc *memory_app.EvolutionAppService
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
		payload = []byte(str)
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
	mux.HandleFunc("GET /api/users/{id}/profile", h.GetUserProfile)
	mux.HandleFunc("POST /api/users/{id}/evolve", h.Evolve)
}

// 1. GET /api/sessions?user_id=xxx
func (h *APIHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	if h.setCORS(w, r) {
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		h.respondError(w, http.StatusBadRequest, "missing user_id query parameter")
		return
	}

	sessions, err := h.sessionSvc.List(r.Context(), userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 转换成轻量级返回结构
	type sessResp struct {
		ID        string    `json:"id"`
		UserID    string    `json:"user_id"`
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
			UpdatedAt: s.UpdatedAt,
			Status:    status,
		})
	}

	h.respondJSON(w, http.StatusOK, list)
}

// 2. POST /api/sessions
func (h *APIHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	if h.setCORS(w, r) {
		return
	}
	var req struct {
		SessionID string `json:"session_id"`
		UserID    string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SessionID == "" || req.UserID == "" {
		h.respondError(w, http.StatusBadRequest, "session_id and user_id are required")
		return
	}

	sess := session.NewSession(req.SessionID, req.UserID, nil)
	if err := h.sessionSvc.Save(r.Context(), sess); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"session_id": sess.ID, "status": "created"})
}

// 3. GET /api/sessions/{id}/messages
func (h *APIHandler) GetSessionMessages(w http.ResponseWriter, r *http.Request) {
	if h.setCORS(w, r) {
		return
	}
	sessionID := r.PathValue("id")
	if sessionID == "" {
		h.respondError(w, http.StatusBadRequest, "missing session_id in path")
		return
	}

	sess, err := h.sessionSvc.Get(r.Context(), sessionID)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			h.respondError(w, http.StatusNotFound, "session not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]any{
		"session_id": sess.ID,
		"user_id":    sess.UserID,
		"messages":   sess.Messages,
		"status":     sess.Metadata["status"],
	})
}

// 4. POST /api/sessions/{id}/chat
func (h *APIHandler) Chat(w http.ResponseWriter, r *http.Request) {
	if h.setCORS(w, r) {
		return
	}
	sessionID := r.PathValue("id")
	var req struct {
		UserID  string `json:"user_id"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.UserID == "" || req.Message == "" {
		h.respondError(w, http.StatusBadRequest, "user_id and message are required")
		return
	}

	// 构造 context，注入 UserID 和 SessionID
	ctx := agent.WithUserID(r.Context(), req.UserID)
	ctx = agent.WithSessionID(ctx, sessionID)

	// 调用智能体流式生成
	userMsg := message.Message{
		Role:    message.RoleUser,
		Content: req.Message,
	}

	// 构建大模型工具参数
	toolsList := make([]tool.Tool, 0, len(h.tools))
	for _, t := range h.tools {
		toolsList = append(toolsList, t)
	}

	reader, err := h.agentSvc.Stream(ctx, []message.Message{userMsg},
		chat.WithTools(toolsList),
	)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() {
		_ = reader.Close()
	}()

	// 开启 SSE 响应
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.logger.Println("SSE flusher conversion failed")
		return
	}

	for {
		msg, err := reader.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				sendSSEEvent(w, "done", "")
				flusher.Flush()
				break
			}
			var interruptErr *session.InterruptError
			if errors.As(err, &interruptErr) {
				// 获取挂起的工具调用列表返回给前端
				sendSSEEvent(w, "interrupt", map[string]any{
					"session_id":    interruptErr.SessionID,
					"pending_tools": interruptErr.ToolCalls,
				})
				flusher.Flush()
				break
			}
			sendSSEEvent(w, "error", map[string]string{"message": err.Error()})
			flusher.Flush()
			break
		}

		if msg != nil {
			switch msg.Type {
			case message.StreamMessageTextDelta:
				sendSSEEvent(w, "text_delta", msg.Content)
			case message.StreamMessageToolCall:
				sendSSEEvent(w, "tool_call", msg.ToolCall)
			case message.StreamMessageToolResult:
				sendSSEEvent(w, "tool_result", msg.ToolResult)
			}
			flusher.Flush()
		}
	}
}

// 5. POST /api/sessions/{id}/confirm
func (h *APIHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	if h.setCORS(w, r) {
		return
	}
	sessionID := r.PathValue("id")
	var req struct {
		UserID                string   `json:"user_id"`
		ConfirmedToolCallIDs []string `json:"confirmed_tool_call_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.UserID == "" {
		h.respondError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	ctx := agent.WithUserID(r.Context(), req.UserID)
	ctx = agent.WithSessionID(ctx, sessionID)

	// 构建大模型工具参数
	toolsList := make([]tool.Tool, 0, len(h.tools))
	for _, t := range h.tools {
		toolsList = append(toolsList, t)
	}

	// 恢复挂起的会话流
	reader, err := h.interactionSvc.ResumeStream(ctx, req.ConfirmedToolCallIDs,
		chat.WithTools(toolsList),
	)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() {
		_ = reader.Close()
	}()

	// 开启 SSE 响应
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.logger.Println("SSE flusher conversion failed")
		return
	}

	for {
		msg, err := reader.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				sendSSEEvent(w, "done", "")
				flusher.Flush()
				break
			}
			var interruptErr *session.InterruptError
			if errors.As(err, &interruptErr) {
				sendSSEEvent(w, "interrupt", map[string]any{
					"session_id":    interruptErr.SessionID,
					"pending_tools": interruptErr.ToolCalls,
				})
				flusher.Flush()
				break
			}
			sendSSEEvent(w, "error", map[string]string{"message": err.Error()})
			flusher.Flush()
			break
		}

		if msg != nil {
			switch msg.Type {
			case message.StreamMessageTextDelta:
				sendSSEEvent(w, "text_delta", msg.Content)
			case message.StreamMessageToolCall:
				sendSSEEvent(w, "tool_call", msg.ToolCall)
			case message.StreamMessageToolResult:
				sendSSEEvent(w, "tool_result", msg.ToolResult)
			}
			flusher.Flush()
		}
	}
}

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
	if userID == "" || sessionID == "" {
		h.respondError(w, http.StatusBadRequest, "user_id and session_id are required")
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

// ==========================================
// Mock 工具具体定义
// ==========================================

type getWeatherTool struct{}

func (t *getWeatherTool) Info() tool.Definition {
	return tool.Definition{
		Name:        "get_weather",
		Description: "获取指定城市的实时天气",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"location": map[string]any{
					"type":        "string",
					"description": "城市名称，例如 北京, 上海, 杭州",
				},
			},
			"required": []any{"location"},
		},
		RequiresConfirmation: false,
	}
}

func (t *getWeatherTool) Execute(ctx context.Context, args string) (string, error) {
	var params struct {
		Location string `json:"location"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", err
	}
	if params.Location == "" {
		params.Location = "本地"
	}
	return fmt.Sprintf("【工具反馈】城市：%s，天气：晴转多云，气温：28℃，PM2.5：35，空气质量优。", params.Location), nil
}

type deleteUserDataTool struct{}

func (t *deleteUserDataTool) Info() tool.Definition {
	return tool.Definition{
		Name:        "delete_user_data",
		Description: "高危操作：清空该用户的全部偏好和事实长期记忆 (需要人工确认)",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"user_id": map[string]any{
					"type":        "string",
					"description": "要删除数据的用户唯一标识 ID",
				},
			},
			"required": []any{"user_id"},
		},
		RequiresConfirmation: true,
	}
}

func (t *deleteUserDataTool) Execute(ctx context.Context, args string) (string, error) {
	var params struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", err
	}
	return fmt.Sprintf("【工具反馈】已成功执行敏感数据删除指令，用户 ID [%s] 的画像数据已被物理抹除。", params.UserID), nil
}
