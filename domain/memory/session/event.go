package session

import (
	"time"

	"vine-agent/domain/event"
	"vine-agent/domain/message"
)

// SessionStreamEventName 定义会话流式事件话题名称
const SessionStreamEventName = "session.stream"

// SessionStreamEvent 结合 Session 信息与 StreamMessage 载荷的事件定义
type SessionStreamEvent struct {
	id         string
	occurredAt time.Time
	sessionID  string
	userID     string
	msg        *message.StreamMessage
	isLast     bool
	err        error
}

var _ event.Event = (*SessionStreamEvent)(nil)

// NewSessionStreamEvent 构造一个普通的流式数据事件实例
func NewSessionStreamEvent(id string, sessionID, userID string, msg *message.StreamMessage) *SessionStreamEvent {
	return &SessionStreamEvent{
		id:         id,
		occurredAt: time.Now(),
		sessionID:  sessionID,
		userID:     userID,
		msg:        msg,
		isLast:     false,
		err:        nil,
	}
}

// NewSessionStreamEndEvent 构造一个流式对话结束事件实例，代表后续无更多流事件
func NewSessionStreamEndEvent(id string, sessionID, userID string, err error) *SessionStreamEvent {
	return &SessionStreamEvent{
		id:         id,
		occurredAt: time.Now(),
		sessionID:  sessionID,
		userID:     userID,
		msg:        nil,
		isLast:     true,
		err:        err,
	}
}

// ID 实现 event.Event 接口
func (e *SessionStreamEvent) ID() string {
	return e.id
}

// Name 实现 event.Event 接口
func (e *SessionStreamEvent) Name() string {
	return SessionStreamEventName
}

// OccurredAt 实现 event.Event 接口
func (e *SessionStreamEvent) OccurredAt() time.Time {
	return e.occurredAt
}

// Payload 实现 event.Event 接口，直接返回事件自身
func (e *SessionStreamEvent) Payload() any {
	return e
}

// SessionID 获取会话 ID
func (e *SessionStreamEvent) SessionID() string {
	return e.sessionID
}

// UserID 获取用户 ID
func (e *SessionStreamEvent) UserID() string {
	return e.userID
}

// Message 获取流式消息载荷，在结束事件中可能为 nil
func (e *SessionStreamEvent) Message() *message.StreamMessage {
	return e.msg
}

// IsLast 判定是否为该 Stream 流的最后一条结束标记事件
func (e *SessionStreamEvent) IsLast() bool {
	return e.isLast
}

// Error 获取结束时的异常错误（若有）
func (e *SessionStreamEvent) Error() error {
	return e.err
}
