package router1

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"vine-agent/app/agent"
	"vine-agent/cmd/bootstrap"
	"vine-agent/cmd/http/dto"
	"vine-agent/domain/chat"
	"vine-agent/domain/memory/session"
	"vine-agent/domain/message"
	"vine-agent/domain/tool"
)

type SessionHandler struct{}

// ListSessions GET /http/sessions?user_id=xxx&project_id=yyy
func (h *SessionHandler) ListSessions(c *gin.Context) {
	var sessReq dto.SessReq
	err := c.ShouldBind(&sessReq)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if sessReq.ProjectID == "" {
		_ = c.Error(errors.New("user_id required"))
		return
	}

	sessions, err := bootstrap.GetAppContainer().SessionAppService.ListSessions(c.Request.Context(), sessReq.UserID, sessReq.ProjectID)
	if err != nil {
		_ = c.Error(err)
		return
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

	bindErr := c.ShouldBind(dto.NewSuccessResp(list))
	if bindErr != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}
}

// 2. POST /http/sessions
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

// 3. GET /http/sessions/{id}/messages
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

// 4. POST /http/sessions/{id}/chat
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
	ctx := r.Context()
	if req.UserID != "" && req.UserID != agent.GetUserID(ctx) {
		ctx = agent.WithUserID(ctx, req.UserID)
	}

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
		// 默认开启的文件工具列表
		fileTools := map[string]bool{
			"read_files": true,
			"write_file": true,
			"list_dir":   true,
		}
		for _, t := range h.tools {
			if enabled[t.Info().Name] || fileTools[t.Info().Name] {
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

// 5. POST /http/sessions/{id}/confirm
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
	ctx := r.Context()
	if req.UserID != "" && req.UserID != agent.GetUserID(ctx) {
		ctx = agent.WithUserID(ctx, req.UserID)
	}

	// 构建大模型工具参数
	toolsList := make([]tool.Tool, 0, len(h.tools))
	if req.Tools != nil {
		enabled := make(map[string]bool, len(req.Tools))
		for _, name := range req.Tools {
			enabled[name] = true
		}
		// 默认开启的文件工具列表
		fileTools := map[string]bool{
			"read_files": true,
			"write_file": true,
			"list_dir":   true,
		}
		for _, t := range h.tools {
			if enabled[t.Info().Name] || fileTools[t.Info().Name] {
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

// 6. POST /http/sessions/{id}/cancel
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

// 7. DELETE /http/sessions/{id}
func (h *APIHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	if h.setCORS(w, r) {
		return
	}
	sessionID := r.PathValue("id")
	if sessionID == "" {
		h.respondError(w, http.StatusBadRequest, "missing session_id in path")
		return
	}

	err := h.projectAppSvc.DeleteSessionInProject(r.Context(), sessionID, "")
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

// 8. POST /http/sessions/{id}/rename
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
