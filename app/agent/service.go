package agent

import (
	"context"
	"fmt"
	"io"
	"sync"

	"vine-agent/domain/chat"
	"vine-agent/domain/memory/session"
	"vine-agent/domain/message"
	"vine-agent/utils"
)

const defaultMaxIter int = 5

var _ chat.ChatModel = (*AgentAppService)(nil)

// AgentAppService 智能体应用层服务，整合了会话记忆持久化与多轮工具调用流迭代逻辑
type AgentAppService struct {
	chatModel  chat.ChatModel
	sessionSvc session.SessionService
}

// NewAgentAppService 构造一个新的 AgentAppService 实例
func NewAgentAppService(chatModel chat.ChatModel, sessionSvc session.SessionService) *AgentAppService {
	return &AgentAppService{
		chatModel:  chatModel,
		sessionSvc: sessionSvc,
	}
}

// Generate 实现同步非流式模型交互（按规定只做一次生成，不调用工具，自动保存 Session 并发布事件）
func (s *AgentAppService) Generate(ctx context.Context, messages []message.Message, opts ...chat.OptionFunc) (*message.Message, error) {
	if s.sessionSvc == nil {
		resp, err := s.chatModel.Generate(ctx, messages, opts...)
		return resp, err
	}

	sessionID, _ := GetSessionID(ctx)
	sess, err := s.sessionSvc.Get(ctx, sessionID)
	if err != nil {
		sess = session.NewSession(sessionID, GetUserID(ctx), nil)
	}

	// 新传入的 messages 添加到 session 中
	sess.Messages = append(sess.Messages, messages...)
	if err := s.sessionSvc.Save(ctx, sess); err != nil {
		return nil, err
	}

	resp, err := s.chatModel.Generate(ctx, sess.Messages, opts...)
	if err != nil {
		return nil, err
	}

	sess.Messages = append(sess.Messages, *resp)
	if err := s.sessionSvc.Save(ctx, sess); err != nil {
		return nil, err
	}

	return resp, nil
}

// Stream 实现流式多轮智能体模型交互
func (s *AgentAppService) Stream(ctx context.Context, messages []message.Message, opts ...chat.OptionFunc) (message.StreamMessageReader, error) {
	if s.sessionSvc == nil {
		reader, err := s.chatModel.Stream(ctx, messages, opts...)
		return reader, err
	}

	sessionID, _ := GetSessionID(ctx)
	// 新传入的 messages 添加到 session 中，如果没有 session 则初始化
	sess, err := s.sessionSvc.Get(ctx, sessionID)
	if err != nil {
		sess = session.NewSession(sessionID, GetUserID(ctx), nil)
	}
	sess.Messages = append(sess.Messages, messages...)
	if err := s.sessionSvc.Save(ctx, sess); err != nil {
		return nil, err
	}

	subCtx, cancel := context.WithCancel(ctx)
	reader := &agentEventReader{
		ch:     make(chan *message.StreamMessage, 100),
		errCh:  make(chan error, 1),
		ctx:    subCtx,
		cancel: cancel,
	}

	go func() {
		defer close(reader.ch)
		err := s.runStreamLoop(subCtx, sess, opts, reader)
		if err != nil {
			reader.sendErr(err)
		}
	}()

	return reader, nil
}

func (s *AgentAppService) runStreamLoop(ctx context.Context, sess *session.Session, opts []chat.OptionFunc, sender StreamEventSender) error {
	chatOpt := &chat.Option{}
	for _, fn := range opts {
		fn(chatOpt)
	}
	if chatOpt.MaxIterations == nil {
		val := defaultMaxIter
		chatOpt.MaxIterations = &val
	}

	for iter := 0; iter < *chatOpt.MaxIterations; iter++ {
		assistantMsg, err := func() (*message.Message, error) {
			stream, err := s.chatModel.Stream(ctx, sess.Messages, opts...)
			if err != nil {
				return nil, err
			}
			defer func() {
				_ = stream.Close()
			}()

			return message.ReadAndAssembleMessage(stream, func(msg *message.StreamMessage) {
				_ = sender.Send(msg)
			})
		}()

		if err != nil {
			return err
		}

		sess.Messages = append(sess.Messages, *assistantMsg)
		if err := s.sessionSvc.Save(ctx, sess); err != nil {
			return err
		}

		// 并发执行所有工具调用并收集返回的 message
		toolMsgs, err := utils.ParallelMap(ctx, assistantMsg.ToolCalls, func(taskCtx context.Context, tc message.ToolCall) (message.Message, error) {
			if !sender.Send(&message.StreamMessage{Type: message.StreamMessageToolCall, ToolCall: &tc}) {
				return message.Message{}, taskCtx.Err()
			}

			toolMsg, execErr := ExecuteToolCall(taskCtx, tc, chatOpt.Tools[tc.Function.Name])
			if execErr != nil {
				return message.Message{}, execErr
			}

			if !sender.Send(&message.StreamMessage{
				Type:       message.StreamMessageToolResult,
				ToolResult: message.NewStreamToolResult(tc.ID, toolMsg.Content, execErr),
			}) {
				return message.Message{}, taskCtx.Err()
			}

			return toolMsg, nil
		})
		if err != nil {
			return err
		}

		// 一次性追加并保存 Session 状态
		sess.Messages = append(sess.Messages, toolMsgs...)
		if err := s.sessionSvc.Save(ctx, sess); err != nil {
			return err
		}
	}

	return fmt.Errorf("agent reached max iterations (%d) without resolving", *chatOpt.MaxIterations)
}

// ==========================================
// agentEventReader: 内部流式读取通道
// ==========================================

// StreamEventSender 定义流式事件发送的契约
type StreamEventSender interface {
	Send(msg *message.StreamMessage) bool
}

type agentEventReader struct {
	ch     chan *message.StreamMessage
	errCh  chan error
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
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
	case err := <-r.errCh:
		if err != nil {
			return nil, err
		}
		return nil, io.EOF
	case <-r.ctx.Done():
		return nil, r.ctx.Err()
	}
}

func (r *agentEventReader) Close() error {
	r.cancel()
	return nil
}

func (r *agentEventReader) Send(ev *message.StreamMessage) bool {
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
