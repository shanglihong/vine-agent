package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"vine-agent/app/agent"
	"vine-agent/domain/chat"
	"vine-agent/domain/memory/session"
	"vine-agent/domain/message"
	"vine-agent/domain/tool"
)

// 1. GET /api/sessions?user_id=xxx&project_id=yyy
func (h *APIHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	if h.setCORS(w, r) {
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		h.respondError(w, http.StatusBadRequest, "missing user_id query parameter")
		return
	}

	hasProjectIDQuery := r.URL.Query().Has("project_id")
	projectID := r.URL.Query().Get("project_id")

	sessions, err := h.sessionAppSvc.ListSessions(r.Context(), userID, projectID, hasProjectIDQuery)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 转换成轻量级返回结构
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

// 2. POST /api/sessions
func (h *APIHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	if h.setCORS(w, r) {
		return
	}
	var req struct {
		SessionID string `json:"session_id"`
		UserID    string `json:"user_id"`
		ProjectID string `json:"project_id"` // 可选项目关联
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SessionID == "" || req.UserID == "" {
		h.respondError(w, http.StatusBadRequest, "session_id and user_id are required")
		return
	}

	// 1. 调用 sessionAppSvc 应用层服务完成创建和最终一致性绑定
	sess, err := h.sessionAppSvc.CreateSession(r.Context(), req.SessionID, req.UserID, req.ProjectID)
	if err != nil {
		if sess == nil {
			h.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		// 仅绑定失败，容错返回成功但打印 Warn 日志
		h.logger.Printf("Warning: failed to bind session %s to project %s: %v", sess.ID, req.ProjectID, err)
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
		"name":       sess.Name,
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
		UserID  string   `json:"user_id"`
		Message string   `json:"message"`
		Model   string   `json:"model"`
		Tools   []string `json:"tools"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Message == "" {
		h.respondError(w, http.StatusBadRequest, "message is required")
		return
	}
	if req.UserID == "" {
		// 从 sessionId 换取 userId
		sess, err := h.sessionSvc.Get(r.Context(), sessionID)
		if err == nil && sess != nil {
			req.UserID = sess.UserID
		}
	}
	if req.UserID == "" {
		h.respondError(w, http.StatusBadRequest, "user_id is required and could not be inferred from session")
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
	if req.Tools != nil {
		enabled := make(map[string]bool, len(req.Tools))
		for _, name := range req.Tools {
			enabled[name] = true
		}
		for _, t := range h.tools {
			if enabled[t.Info().Name] {
				toolsList = append(toolsList, t)
			}
		}
	} else {
		for _, t := range h.tools {
			toolsList = append(toolsList, t)
		}
	}

	reader, err := h.agentSvc.Stream(ctx, []message.Message{userMsg},
		chat.WithTools(toolsList),
		chat.WithModel(req.Model),
	)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.activeStreams.Store(sessionID, reader)
	defer func() {
		h.activeStreams.Delete(sessionID)
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
			case message.StreamMessageReasoningDelta:
				sendSSEEvent(w, "reasoning_delta", msg.Content)
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
		UserID               string   `json:"user_id"`
		ConfirmedToolCallIDs []string `json:"confirmed_tool_call_ids"`
		Tools                []string `json:"tools"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.UserID == "" {
		// 从 sessionId 换取 userId
		sess, err := h.sessionSvc.Get(r.Context(), sessionID)
		if err == nil && sess != nil {
			req.UserID = sess.UserID
		}
	}
	if req.UserID == "" {
		h.respondError(w, http.StatusBadRequest, "user_id is required and could not be inferred from session")
		return
	}

	ctx := agent.WithUserID(r.Context(), req.UserID)
	ctx = agent.WithSessionID(ctx, sessionID)

	// 构建大模型工具参数
	toolsList := make([]tool.Tool, 0, len(h.tools))
	if req.Tools != nil {
		enabled := make(map[string]bool, len(req.Tools))
		for _, name := range req.Tools {
			enabled[name] = true
		}
		for _, t := range h.tools {
			if enabled[t.Info().Name] {
				toolsList = append(toolsList, t)
			}
		}
	} else {
		for _, t := range h.tools {
			toolsList = append(toolsList, t)
		}
	}

	// 恢复挂起的会话流
	reader, err := h.interactionSvc.ResumeStream(ctx, req.ConfirmedToolCallIDs,
		chat.WithTools(toolsList),
	)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.activeStreams.Store(sessionID, reader)
	defer func() {
		h.activeStreams.Delete(sessionID)
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
			case message.StreamMessageReasoningDelta:
				sendSSEEvent(w, "reasoning_delta", msg.Content)
			case message.StreamMessageToolCall:
				sendSSEEvent(w, "tool_call", msg.ToolCall)
			case message.StreamMessageToolResult:
				sendSSEEvent(w, "tool_result", msg.ToolResult)
			}
			flusher.Flush()
		}
	}
}

// 6. POST /api/sessions/{id}/cancel
func (h *APIHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	if h.setCORS(w, r) {
		return
	}
	sessionID := r.PathValue("id")
	if sessionID == "" {
		h.respondError(w, http.StatusBadRequest, "missing session_id in path")
		return
	}

	val, ok := h.activeStreams.Load(sessionID)
	if !ok {
		h.respondJSON(w, http.StatusOK, map[string]string{"status": "no active stream found"})
		return
	}

	reader, ok := val.(message.StreamMessageReader)
	if !ok {
		h.respondError(w, http.StatusInternalServerError, "invalid reader type in stream map")
		return
	}

	if err := reader.Interrupt(); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.activeStreams.Delete(sessionID)
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// 7. DELETE /api/sessions/{id}
func (h *APIHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	if h.setCORS(w, r) {
		return
	}
	sessionID := r.PathValue("id")
	if sessionID == "" {
		h.respondError(w, http.StatusBadRequest, "missing session_id in path")
		return
	}

	err := h.sessionSvc.Delete(r.Context(), sessionID)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			h.respondError(w, http.StatusNotFound, "session not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"session_id": sessionID, "status": "deleted"})
}

// 8. POST /api/sessions/{id}/rename
func (h *APIHandler) RenameSession(w http.ResponseWriter, r *http.Request) {
	if h.setCORS(w, r) {
		return
	}
	sessionID := r.PathValue("id")
	if sessionID == "" {
		h.respondError(w, http.StatusBadRequest, "missing session_id in path")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		h.respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	err := h.sessionSvc.Rename(r.Context(), sessionID, req.Name)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			h.respondError(w, http.StatusNotFound, "session not found")
			return
		}
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"session_id": sessionID, "status": "renamed", "name": req.Name})
}
