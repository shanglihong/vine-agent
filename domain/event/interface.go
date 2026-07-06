//go:generate go run github.com/golang/mock/mockgen -source=interface.go -destination=./mock/event_mock.go -package=mock

package event

import (
	"context"
	"time"
)

// Event 领域事件接口契约
type Event interface {
	ID() string            // 事件唯一标识
	Name() string          // 事件名称（即 Topic）
	OccurredAt() time.Time // 事件发生时间
	Payload() any          // 事件携带的 Payload 数据
}

// Handler 事件处理器契约
type Handler interface {
	Handle(ctx context.Context, ev Event) error
}

// HandlerFunc 允许使用普通的函数作为事件处理器
type HandlerFunc func(ctx context.Context, ev Event) error

// Handle 实现 Handler 接口契约
func (f HandlerFunc) Handle(ctx context.Context, ev Event) error {
	return f(ctx, ev)
}

// Publisher 事件发布接口契约
type Publisher interface {
	Publish(ctx context.Context, ev Event) error
}

// Subscriber 事件订阅接口契约
type Subscriber interface {
	Subscribe(name string, handler Handler) error
}

// EventBus 事件总线接口契约，组合了发布、订阅以及生命周期管理
type EventBus interface {
	Publisher
	Subscriber
	Shutdown(ctx context.Context) error
}
