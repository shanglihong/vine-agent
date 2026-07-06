package agent

import (
	"context"
	"io"
	"sync"

	"vine-agent/domain/message"
)

// ==========================================
// agentEventReader: 内部流式读取通道
// ==========================================

type agentEventReader struct {
	ch     chan *message.StreamMessage
	errCh  chan error
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
	closed bool
}

func (r *agentEventReader) Recv() (*message.StreamMessage, error) {
	select {
	case ev, ok := <-r.ch:
		if !ok {
			select {
			case err := <-r.errCh:
				if err != nil {
					return nil, err
				}
			default:
			}
			return nil, io.EOF
		}
		return ev, nil
	case <-r.ctx.Done():
		return nil, r.ctx.Err()
	}
}

func (r *agentEventReader) Close() error {
	r.cancel()
	return nil
}

func (r *agentEventReader) Send(ev *message.StreamMessage) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	select {
	case r.ch <- ev:
		return true
	case <-r.ctx.Done():
		return false
	}
}

func (r *agentEventReader) sendErr(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	select {
	case r.errCh <- err:
	default:
	}
}
