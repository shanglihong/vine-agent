package event

import (
	"context"
	"errors"
	"log"
	"sync"

	"vine-agent/domain/event"
)

var (
	// ErrEventBusClosed 事件总线已被关闭错误
	ErrEventBusClosed = errors.New("event bus is closed")
	// ErrQueueFull 缓冲队列已满错误
	ErrQueueFull = errors.New("event queue is full")
)

// eventTask 用于封装异步处理所需的 Context 和 Event
type eventTask struct {
	ctx context.Context
	ev  event.Event
}

// InMemoryEventBus 基于内存 Channel + Worker 协程池的并发安全事件总线
type InMemoryEventBus struct {
	subscribersMu sync.RWMutex
	subscribers   map[string][]event.Handler

	taskChan chan eventTask
	wg       sync.WaitGroup

	closeMu sync.Mutex
	closed  bool

	logger Logger
}

// Logger 定义事件总线内部使用的简易日志契约
type Logger interface {
	Printf(format string, v ...any)
}

// defaultLogger 实现默认的 Logger 行为，底层调用标准库 log 打印
type defaultLogger struct{}

func (d *defaultLogger) Printf(format string, v ...any) {
	log.Printf(format, v...)
}

// NewInMemoryEventBus 构造一个内存异步事件总线实例
// bufferSize: 通道缓冲区大小，控制最多允许积压多少未分发事件
// workerNum: 工作协程数量，控制并发消费的协程数
// l: 自定义日志记录器，传入 nil 时默认使用标准库 log 打印
func NewInMemoryEventBus(bufferSize int, workerNum int, l Logger) *InMemoryEventBus {
	if l == nil {
		l = &defaultLogger{}
	}
	bus := &InMemoryEventBus{
		subscribers: make(map[string][]event.Handler),
		taskChan:    make(chan eventTask, bufferSize),
		logger:      l,
	}

	// 启动后台工作协程
	for i := 0; i < workerNum; i++ {
		bus.wg.Add(1)
		go bus.workerLoop()
	}

	return bus
}

// Publish 异步发布一个事件
func (b *InMemoryEventBus) Publish(ctx context.Context, ev event.Event) error {
	b.closeMu.Lock()
	if b.closed {
		b.closeMu.Unlock()
		return ErrEventBusClosed
	}
	b.closeMu.Unlock()

	task := eventTask{
		ctx: ctx,
		ev:  ev,
	}

	select {
	case b.taskChan <- task:
		return nil
	default:
		return ErrQueueFull
	}
}

// Subscribe 注册一个事件订阅者
func (b *InMemoryEventBus) Subscribe(name string, handler event.Handler) error {
	b.closeMu.Lock()
	if b.closed {
		b.closeMu.Unlock()
		return ErrEventBusClosed
	}
	b.closeMu.Unlock()

	b.subscribersMu.Lock()
	defer b.subscribersMu.Unlock()

	b.subscribers[name] = append(b.subscribers[name], handler)
	return nil
}

// Shutdown 优雅关闭事件总线
// 停止接受新事件 -> 关闭 channel -> 等待所有 worker 将积压任务处理完成 -> 退出
func (b *InMemoryEventBus) Shutdown(ctx context.Context) error {
	b.closeMu.Lock()
	if b.closed {
		b.closeMu.Unlock()
		return nil
	}
	b.closed = true
	b.closeMu.Unlock()

	// 关闭 taskChan，这会使工作协程在读空 Channel 后逐个退出
	close(b.taskChan)

	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// workerLoop 消费者循环
func (b *InMemoryEventBus) workerLoop() {
	defer b.wg.Done()

	for task := range b.taskChan {
		b.dispatchEvent(task.ctx, task.ev)
	}
}

// dispatchEvent 查找并执行该事件注册的所有处理器
func (b *InMemoryEventBus) dispatchEvent(ctx context.Context, ev event.Event) {
	b.subscribersMu.RLock()
	handlers, ok := b.subscribers[ev.Name()]
	if !ok || len(handlers) == 0 {
		b.subscribersMu.RUnlock()
		return
	}
	// Copy 副本，减少锁的竞争时间，方便在 Handle 耗时长的情况下不阻塞新订阅
	copiedHandlers := make([]event.Handler, len(handlers))
	copy(copiedHandlers, handlers)
	b.subscribersMu.RUnlock()

	for _, handler := range copiedHandlers {
		b.executeHandler(ctx, handler, ev)
	}
}

// executeHandler 执行单个处理器，保证 Panic 被捕获与错误日志被记录
func (b *InMemoryEventBus) executeHandler(ctx context.Context, handler event.Handler, ev event.Event) {
	defer func() {
		if r := recover(); r != nil {
			b.logger.Printf("[EventBus] Panic recovered in handler for event %q, panic: %v", ev.Name(), r)
		}
	}()

	if err := handler.Handle(ctx, ev); err != nil {
		b.logger.Printf("[EventBus] Error handling event %q: %v", ev.Name(), err)
	}
}
