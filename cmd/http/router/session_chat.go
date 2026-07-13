package router

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"vine-agent/app/agent"
	"vine-agent/cmd/bootstrap"
	"vine-agent/cmd/http/dto"
	"vine-agent/cmd/http/router1"
	"vine-agent/domain/chat"
	"vine-agent/domain/memory/session"
	"vine-agent/domain/message"
	"vine-agent/domain/tool"
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

// 6. POST /http/sessions/{id}/cancel
func (h *router1.APIHandler) Cancel(w http.ResponseWriter, r *http.Request) {
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




// 4. POST /http/sessions/{id}/chat
func (h *router1.APIHandler) Chat(w http.ResponseWriter, r *http.Request) {
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
				router1.sendSSEEvent(w, "done", "")
				flusher.Flush()
				break
			}
			var interruptErr *session.InterruptError
			if errors.As(err, &interruptErr) {
				router1.sendSSEEvent(w, "interrupt", map[string]any{
					"session_id":    interruptErr.SessionID,
					"pending_tools": interruptErr.ToolCalls,
				})
				flusher.Flush()
				break
			}
			router1.sendSSEEvent(w, "error", map[string]string{"message": err.Error()})
			flusher.Flush()
			break
		}

		if msg != nil {
			switch msg.Type {
			case message.StreamMessageTextDelta:
				router1.sendSSEEvent(w, "text_delta", msg.Content)
			case message.StreamMessageReasoningDelta:
				router1.sendSSEEvent(w, "reasoning_delta", msg.Content)
			case message.StreamMessageToolCall:
				router1.sendSSEEvent(w, "tool_call", msg.ToolCall)
			case message.StreamMessageToolResult:
				router1.sendSSEEvent(w, "tool_result", msg.ToolResult)
			}
			flusher.Flush()
		}
	}
}

// 5. POST /http/sessions/{id}/confirm
func (h *router1.APIHandler) Confirm(w http.ResponseWriter, r *http.Request) {
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
				router1.sendSSEEvent(w, "done", "")
				flusher.Flush()
				break
			}
			var interruptErr *session.InterruptError
			if errors.As(err, &interruptErr) {
				router1.sendSSEEvent(w, "interrupt", map[string]any{
					"session_id":    interruptErr.SessionID,
					"pending_tools": interruptErr.ToolCalls,
				})
				flusher.Flush()
				break
			}
			router1.sendSSEEvent(w, "error", map[string]string{"message": err.Error()})
			flusher.Flush()
			break
		}

		if msg != nil {
			switch msg.Type {
			case message.StreamMessageTextDelta:
				router1.sendSSEEvent(w, "text_delta", msg.Content)
			case message.StreamMessageReasoningDelta:
				router1.sendSSEEvent(w, "reasoning_delta", msg.Content)
			case message.StreamMessageToolCall:
				router1.sendSSEEvent(w, "tool_call", msg.ToolCall)
			case message.StreamMessageToolResult:
				router1.sendSSEEvent(w, "tool_result", msg.ToolResult)
			}
			flusher.Flush()
		}
	}
}

