package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
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
	if err := c.ShouldBindUri(&req); err != nil {
		_ = c.Error(err)
		return
	}
	if err := c.ShouldBind(&req); err != nil {
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

	h.stream2Resp(c, reader, err)
}

func (h *SessionChatHandler) Resume(c *gin.Context) {
	var req dto.SessResumeChatReq
	if err := c.ShouldBindUri(&req); err != nil {
		_ = c.Error(err)
		return
	}
	if err := c.ShouldBind(&req); err != nil {
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

	h.stream2Resp(c, reader, err)
}

func (h *SessionChatHandler) Cancel(c *gin.Context) {
	var req dto.SessIdReq
	if err := c.ShouldBindUri(&req); err != nil {
		_ = c.Error(err)
		return
	}
	if err := c.ShouldBind(&req); err != nil {
		_ = c.Error(err)
		return
	}

	activeStreams.Delete(req.SessionID)
	c.JSON(http.StatusOK, dto.NewSuccessResp(dto.SessResp{Status: "cancelled"}))

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

func (h *SessionChatHandler) stream2Resp(c *gin.Context, reader message.StreamMessageReader, err error) bool {
	return c.Stream(
		func(w io.Writer) bool {
			msg, readerErr := reader.Recv()
			if readerErr != nil {
				if errors.Is(err, io.EOF) {
					c.SSEvent("done", "")
					return false // 执行结束
				}
				var interruptErr *session.InterruptError
				if errors.As(err, &interruptErr) {
					c.SSEvent("interrupt", map[string]any{
						"session_id":    interruptErr.SessionID,
						"pending_tools": interruptErr.ToolCalls,
					})
					return false
				}
				c.SSEvent("error", map[string]string{"message": err.Error()})
				return false
			}
			if msg != nil {
				switch msg.Type {
				case message.StreamMessageTextDelta:
					c.SSEvent("text_delta", msg.Content)
				case message.StreamMessageReasoningDelta:
					c.SSEvent("reasoning_delta", msg.Content)
				case message.StreamMessageToolCall:
					c.SSEvent("tool_call", msg.ToolCall)
				case message.StreamMessageToolResult:
					c.SSEvent("tool_result", msg.ToolResult)
				}
			}

			return true // 开启下一轮
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
