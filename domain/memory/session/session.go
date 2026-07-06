package session

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"vine-agent/domain/message"
)

const (
	SessionStatusKey                 = "status"
	SessionStatusPendingConfirmation = "pending_confirmation"

	LastEvolvedMsgCountKey            = "last_evolved_msg_count"
	MetadataPendingConfirmToolCallIDs = "pending_confirm_tool_call_ids"
	MetadataPendingConfirmToolNames   = "pending_confirm_tool_names"
)

// Session 代表一个 AI 对话会话领域对象（聚合根，作为短期记忆）
type Session struct {
	ID        string            `json:"id"`
	UserID    string            `json:"user_id"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Metadata  map[string]string `json:"metadata"`
	Messages  []message.Message `json:"messages"`
}

// NewSession 创建一个新的会话实例。如果传入的 id 为空，会内置自动生成一个基于时间戳的唯一会话 ID
func NewSession(id, userID string, metadata map[string]string) *Session {
	if id == "" {
		id = fmt.Sprintf("sess_%d", time.Now().UnixNano())
	}
	if metadata == nil {
		metadata = make(map[string]string)
	}
	now := time.Now()
	return &Session{
		ID:        id,
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  metadata,
		Messages:  make([]message.Message, 0),
	}
}

// GetStatus 获取会话的当前状态
func (s *Session) GetStatus() string {
	if s.Metadata == nil {
		return ""
	}
	return s.Metadata[SessionStatusKey]
}

// MarkPendingConfirmation 将会话标记为等待工具执行确认状态
func (s *Session) MarkPendingConfirmation() {
	if s.Metadata == nil {
		s.Metadata = make(map[string]string)
	}
	s.Metadata[SessionStatusKey] = SessionStatusPendingConfirmation
}

// IsPendingConfirmation 检查会话当前是否处于等待工具执行确认的状态
func (s *Session) IsPendingConfirmation() bool {
	return s.GetStatus() == SessionStatusPendingConfirmation
}

// ClearStatus 清除会话的状态
func (s *Session) ClearStatus() {
	if s.Metadata != nil {
		delete(s.Metadata, SessionStatusKey)
	}
}

// ApplyInterrupt 根据中断错误更新会话的状态。如果传入的错误代表中断，则更新状态并返回 true
func (s *Session) ApplyInterrupt(err error) bool {
	if err == nil {
		return false
	}
	var interruptErr *InterruptError
	if errors.As(err, &interruptErr) {
		if s.Metadata == nil {
			s.Metadata = make(map[string]string)
		}
		s.Metadata[SessionStatusKey] = interruptErr.Status
		if len(interruptErr.ToolCalls) > 0 {
			var ids []string
			var names []string
			for _, tc := range interruptErr.ToolCalls {
				ids = append(ids, tc.ID)
				names = append(names, tc.Function.Name)
			}
			s.Metadata[MetadataPendingConfirmToolCallIDs] = joinStrings(ids, ",")
			s.Metadata[MetadataPendingConfirmToolNames] = joinStrings(names, ",")
		}
		return true
	}
	return false
}

// GetLastEvolvedMsgCount 获取上次完成偏好演进的消息总数。如果未记录或解析失败，则返回 0
func (s *Session) GetLastEvolvedMsgCount() int {
	if s.Metadata == nil {
		return 0
	}
	val, ok := s.Metadata[LastEvolvedMsgCountKey]
	if !ok {
		return 0
	}
	count, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}
	return count
}

// UpdateLastEvolvedMsgCount 将上次已演进的消息总数更新为当前消息的总长度
func (s *Session) UpdateLastEvolvedMsgCount() {
	if s.Metadata == nil {
		s.Metadata = make(map[string]string)
	}
	s.Metadata[LastEvolvedMsgCountKey] = fmt.Sprintf("%d", len(s.Messages))
}

// GetPendingConfirmToolCallIDs 获取当前挂起等待确认的工具调用 ID 列表。
func (s *Session) GetPendingConfirmToolCallIDs() []string {
	if s.Metadata == nil {
		return nil
	}
	pendingIDsStr := s.Metadata[MetadataPendingConfirmToolCallIDs]
	if pendingIDsStr == "" {
		return nil
	}
	parts := strings.Split(pendingIDsStr, ",")
	var ids []string
	for _, id := range parts {
		id = strings.TrimSpace(id)
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// ClearPendingConfirmations 清除会话中所有与挂起工具调用相关的元数据和状态。
func (s *Session) ClearPendingConfirmations() {
	s.ClearStatus()
	if s.Metadata != nil {
		delete(s.Metadata, MetadataPendingConfirmToolCallIDs)
		delete(s.Metadata, MetadataPendingConfirmToolNames)
	}
}

// CancelPendingConfirmations 检查会话状态是否为 pending_confirmation。
// 如果是，说明存在挂起的工具调用审批，而用户此时直接输入了新消息。
// 我们自动将挂起的工具调用标记为“已被取消”状态以维持消息流时序合规与状态一致性。
func (s *Session) CancelPendingConfirmations() {
	if !s.IsPendingConfirmation() {
		return
	}

	pendingIDs := s.GetPendingConfirmToolCallIDs()
	for _, id := range pendingIDs {
		// 自动追加一个 Tool 消息，表示调用被用户取消了
		cancelMsg := message.NewToolMessage(id, "Tool execution cancelled because user sent a new message instead of approving.")
		s.Messages = append(s.Messages, cancelMsg)
	}

	// 清理会话挂起状态
	s.ClearPendingConfirmations()
}

func joinStrings(elems []string, sep string) string {
	if len(elems) == 0 {
		return ""
	}
	res := elems[0]
	for _, val := range elems[1:] {
		res += sep + val
	}
	return res
}
