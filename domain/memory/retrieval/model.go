package retrieval

import "vine-agent/domain/message"

// SearchResult 跨会话检索时的结果封装，包含完整的 message 实体以及所属的会话上下文
type SearchResult struct {
	SessionID string          `json:"session_id"`
	UserID    string          `json:"user_id"`
	Message   message.Message `json:"message"`
}
