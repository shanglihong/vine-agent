package session

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"vine-agent/domain/message"
)

const (
	SessionStatusKey                 = "status"
	SessionStatusPendingConfirmation = "pending_confirmation"
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
			s.Metadata["pending_confirm_tool_call_ids"] = joinStrings(ids, ",")
			s.Metadata["pending_confirm_tool_names"] = joinStrings(names, ",")
		}
		return true
	}
	return false
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

const LastEvolvedMsgCountKey = "last_evolved_msg_count"

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
