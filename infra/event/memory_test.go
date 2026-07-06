package event_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vine-agent/domain/event"
	infraevent "vine-agent/infra/event"
)

// mockEvent 用于测试的 Event 实体
type mockEvent struct {
	id         string
	name       string
	occurredAt time.Time
	payload    any
}

func (e *mockEvent) ID() string            { return e.id }
func (e *mockEvent) Name() string          { return e.name }
func (e *mockEvent) OccurredAt() time.Time { return e.occurredAt }
func (e *mockEvent) Payload() any          { return e.payload }

// mockLogger 记录日志，用于测试捕获
type mockLogger struct {
	mu   sync.Mutex
	logs []string
}

func (m *mockLogger) Printf(format string, v ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, format)
}

func (m *mockLogger) GetLogs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.logs
}

func TestInMemoryEventBus_PublishSubscribe(t *testing.T) {
	bus := infraevent.NewInMemoryEventBus(10, 2, nil)
	defer bus.Shutdown(context.Background())

	ctx := context.Background()
	eventName := "test.event"

	var receivedEvent event.Event
	wg := sync.WaitGroup{}
	wg.Add(1)

	handler := event.HandlerFunc(func(ctx context.Context, ev event.Event) error {
		receivedEvent = ev
		wg.Done()
		return nil
	})

	err := bus.Subscribe(eventName, handler)
	require.NoError(t, err)

	ev := &mockEvent{
		id:         "123",
		name:       eventName,
		occurredAt: time.Now(),
		payload:    "hello world",
	}

	err = bus.Publish(ctx, ev)
	require.NoError(t, err)

	// 等待异步处理完成
	wg.Wait()

	assert.Equal(t, ev.ID(), receivedEvent.ID())
	assert.Equal(t, ev.Name(), receivedEvent.Name())
	assert.Equal(t, ev.Payload(), receivedEvent.Payload())
}

func TestInMemoryEventBus_PanicRecovery(t *testing.T) {
	logger := &mockLogger{}
	bus := infraevent.NewInMemoryEventBus(10, 2, logger)
	defer bus.Shutdown(context.Background())

	ctx := context.Background()
	eventName := "panic.event"

	wg := sync.WaitGroup{}
	wg.Add(2)

	// 第一个 handler 会 panic
	panicHandler := event.HandlerFunc(func(ctx context.Context, ev event.Event) error {
		defer wg.Done()
		panic("something went wrong")
	})

	// 第二个 handler 会正常执行，说明 panic 没有破坏 worker 协程，后续处理器能正常运行
	normalHandlerRun := false
	normalHandler := event.HandlerFunc(func(ctx context.Context, ev event.Event) error {
		normalHandlerRun = true
		wg.Done()
		return nil
	})

	err := bus.Subscribe(eventName, panicHandler)
	require.NoError(t, err)
	err = bus.Subscribe(eventName, normalHandler)
	require.NoError(t, err)

	ev := &mockEvent{
		id:         "456",
		name:       eventName,
		occurredAt: time.Now(),
		payload:    nil,
	}

	err = bus.Publish(ctx, ev)
	require.NoError(t, err)

	wg.Wait()

	assert.True(t, normalHandlerRun, "Normal handler should still run despite panic in other handler")
	assert.NotEmpty(t, logger.GetLogs(), "Should log the panic")
	assert.Contains(t, logger.GetLogs()[0], "Panic recovered")
}

func TestInMemoryEventBus_Shutdown(t *testing.T) {
	bus := infraevent.NewInMemoryEventBus(10, 1, nil)

	ctx := context.Background()
	eventName := "shutdown.event"

	wg := sync.WaitGroup{}
	wg.Add(1)

	// 处理器中加点延迟，模拟正在处理
	handler := event.HandlerFunc(func(ctx context.Context, ev event.Event) error {
		time.Sleep(100 * time.Millisecond)
		wg.Done()
		return nil
	})

	err := bus.Subscribe(eventName, handler)
	require.NoError(t, err)

	ev := &mockEvent{
		id:         "789",
		name:       eventName,
		occurredAt: time.Now(),
	}

	err = bus.Publish(ctx, ev)
	require.NoError(t, err)

	// 立即调用 Shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err = bus.Shutdown(shutdownCtx)
	require.NoError(t, err)

	// 此时 wg.Wait() 应该可以直接完成，说明在 Shutdown 过程中，积压的任务已经被处理完了
	wg.Wait()

	// 关闭后，再发布应该报错
	err = bus.Publish(ctx, ev)
	assert.Equal(t, infraevent.ErrEventBusClosed, err)

	// 关闭后，再订阅也应该报错
	err = bus.Subscribe(eventName, handler)
	assert.Equal(t, infraevent.ErrEventBusClosed, err)
}

func TestInMemoryEventBus_QueueFull(t *testing.T) {
	// 创建一个缓冲区为 0 且协程数为 0 的 EventBus，任何 Publish 都会因为没有 worker 且无缓冲区而返回 QueueFull
	bus := infraevent.NewInMemoryEventBus(0, 0, nil)

	ctx := context.Background()
	ev := &mockEvent{
		id:         "999",
		name:       "full.event",
		occurredAt: time.Now(),
	}

	err := bus.Publish(ctx, ev)
	assert.Equal(t, infraevent.ErrQueueFull, err)
}
