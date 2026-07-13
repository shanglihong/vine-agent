package router

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"vine-agent/cmd/bootstrap"
	"vine-agent/cmd/http/dto"
	"vine-agent/domain/chat"
	"vine-agent/domain/memory/session"
	"vine-agent/domain/message"
	"vine-agent/domain/tool"
	"vine-agent/infra/tools"

	"github.com/gin-gonic/gin"
)

var (
	systemTools = []tool.Tool{
		tools.NewWebSearchTool(),
		tools.NewWebCrawlTool(),
		tools.NewListDirTool(),
		tools.NewReadFilesTool(),
		tools.NewWriteFileTool(),
	}
	toolsMap           = make(map[string]tool.Tool)
	activeStreams      sync.Map
	sessionChatHandler = SessionChatHandler{}
)

func init() {
	for _, t := range systemTools {
		toolsMap[t.Info().Name] = t
	}
}

type SessionChatHandler struct{}

func GetSessionChatHandler() *SessionChatHandler {
	return &sessionChatHandler
}

func (h *SessionChatHandler) ChatAgent(ctx context.Context, req dto.SessChatReq) (message.StreamMessageReader, error) {
	// 调用智能体流式生成
	userMsg := message.Message{
		Role:    message.RoleUser,
		Content: req.Message,
	}
	toolsList := getTools(req.Tools)

	reader, err := bootstrap.GetAppContainer().AgentService.Stream(ctx, []message.Message{userMsg},
		chat.WithTools(toolsList),
		chat.WithModel(req.Model),
	)
	return reader, err
}

func (h *SessionChatHandler) ResumeChatAgent(ctx context.Context, req dto.SessResumeChatReq) (message.StreamMessageReader, error) {
	// 构建大模型工具参数
	toolsList := make([]tool.Tool, 0, len(systemTools))
	if req.Tools != nil {
		req.Tools = append(req.Tools, "read_files", "write_files", "list_dir")
	}
	for _, toolName := range req.Tools {
		if t, ok := toolsMap[toolName]; ok {
			toolsList = append(toolsList, t)
		}
	}
	// 恢复挂起的会话流
	reader, err := bootstrap.GetAppContainer().InteractionService.ResumeStream(ctx, req.ConfirmedToolCallIDs,
		chat.WithTools(toolsList),
	)
	return reader, err
}

func (h *SessionChatHandler) Chat(c *gin.Context) {
	var req dto.SessChatReq

	if err := Bind(c, &req); err != nil {
		_ = c.Error(err)
		return
	}

	reader, err := h.ChatAgent(c.Request.Context(), req)
	if err != nil {
		_ = c.Error(err)
	}
	activeStreams.Store(req.SessionID, reader)
	defer func() {
		activeStreams.Delete(req.SessionID)
		_ = reader.Close()
	}()

	h.stream2Resp(c, reader)
}

func (h *SessionChatHandler) Resume(c *gin.Context) {
	var req dto.SessResumeChatReq
	if err := Bind(c, &req); err != nil {
		_ = c.Error(err)
		return
	}
	reader, err := h.ResumeChatAgent(c.Request.Context(), req)
	if err != nil {
		_ = c.Error(err)
	}
	activeStreams.Store(req.UserID, reader)
	defer func() {
		activeStreams.Delete(req.UserID)
		_ = reader.Close()
	}()

	h.stream2Resp(c, reader)
}

func (h *SessionChatHandler) Cancel(c *gin.Context) {
	var req dto.SessIdReq
	if err := Bind(c, &req); err != nil {
		_ = c.Error(err)
		return
	}

	activeStreams.Delete(req.SessionID)
	c.JSON(http.StatusOK, dto.NewSuccessResp(dto.SessResp{Status: "cancelled"}))
}

func (h *SessionChatHandler) stream2Resp(c *gin.Context, reader message.StreamMessageReader) bool {
	return c.Stream(
		func(w io.Writer) bool {
			msg, readerErr := reader.Recv()
			if readerErr != nil {
				if errors.Is(readerErr, io.EOF) {
					c.SSEvent("done", "")
					return false
				}
				var interruptErr *session.InterruptError
				if errors.As(readerErr, &interruptErr) {
					c.SSEvent("interrupt", map[string]any{
						"session_id":    interruptErr.SessionID,
						"pending_tools": interruptErr.ToolCalls,
					})
					return false
				}
				c.SSEvent("error", map[string]string{"message": readerErr.Error()})
				return false
			}

			if msg != nil {
				switch msg.Type {
				case message.StreamMessageTextDelta:
					// 🔥 关键修改：将字符串包装成 JSON
					jsonData := map[string]string{
						"content": msg.Content,
					}
					c.SSEvent("text_delta", jsonData)

				case message.StreamMessageReasoningDelta:
					jsonData := map[string]string{
						"content": msg.Content,
					}
					c.SSEvent("reasoning_delta", jsonData)

				case message.StreamMessageToolCall:
					c.SSEvent("tool_call", msg.ToolCall)

				case message.StreamMessageToolResult:
					c.SSEvent("tool_result", msg.ToolResult)
				}
			}
			return true
		})
}

func getTools(toolsName []string) []tool.Tool {
	// 构建大模型工具参数
	toolsList := make([]tool.Tool, 0, len(systemTools))
	if toolsName != nil {
		toolsName = append(toolsName, "read_files", "write_files", "list_dir")
	}
	for _, toolName := range toolsName {
		if t, ok := toolsMap[toolName]; ok {
			toolsList = append(toolsList, t)
		}
	}
	return toolsList
}
