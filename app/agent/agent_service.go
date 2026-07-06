package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"vine-agent/domain/chat"
	"vine-agent/domain/event"
	"vine-agent/domain/memory/session"
	"vine-agent/domain/message"
	"vine-agent/utils"
)

const defaultMaxIter int = 5

var _ Service = (*agentService)(nil)

// agentService 智能体应用层服务，整合了会话记忆持久化与多轮工具调用流迭代逻辑
type agentService struct {
	chatModel  chat.ChatModel
	sessionSvc session.SessionService
	publisher  event.Publisher

	// 活跃通道管理器，订阅事件后路由分发
	mu      sync.RWMutex
	readers map[string]*agentEventReader
}

// NewService 构造一个新的 Service 实例
func NewService(
	chatModel chat.ChatModel,
	sessionSvc session.SessionService,
	publisher event.Publisher,
	subscriber event.Subscriber,
) Service {
	svc := &agentService{
		chatModel:  chatModel,
		sessionSvc: sessionSvc,
		publisher:  publisher,
		readers:    make(map[string]*agentEventReader),
	}

	if subscriber != nil {
		// 自动订阅 SessionStream 领域事件话题，事件定义由 session 领域内聚提供
		_ = subscriber.Subscribe(session.SessionStreamEventName, event.HandlerFunc(svc.handleStreamEvent))
	}

	return svc
}

// Generate 实现同步非流式模型交互（按规定只做一次生成，不调用工具，自动保存 Session）
func (s *agentService) Generate(ctx context.Context, messages []message.Message, opts ...chat.OptionFunc) (*message.Message, error) {
	if s.sessionSvc == nil {
		resp, err := s.chatModel.Generate(ctx, messages, opts...)
		return resp, err
	}

	sess, err := s.acceptUserMessages(ctx, messages)
	if err != nil {
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
func (s *agentService) Stream(ctx context.Context, messages []message.Message, opts ...chat.OptionFunc) (message.StreamMessageReader, error) {
	if s.sessionSvc == nil {
		reader, err := s.chatModel.Stream(ctx, messages, opts...)
		return reader, err
	}

	sess, err := s.acceptUserMessages(ctx, messages)
	if err != nil {
		return nil, err
	}

	subCtx, cancel := context.WithCancel(ctx)
	reader := &agentEventReader{
		ch:     make(chan *message.StreamMessage, 100),
		errCh:  make(chan error, 1),
		ctx:    subCtx,
		cancel: cancel,
	}

	// 注册当前 Session 的活跃通道读取器
	s.mu.Lock()
	s.readers[sess.ID] = reader
	s.mu.Unlock()

	go func() {
		err := s.runStreamLoop(subCtx, sess, opts)
		// 无论迭代成功或出现异常，均发布结束事件以通知订阅端进行生命周期注销与关闭
		_ = s.publishEndEvent(subCtx, sess, err)
	}()

	return reader, nil
}

func (s *agentService) acceptUserMessages(ctx context.Context, messages []message.Message) (*session.Session, error) {
	sessionID, _ := GetSessionID(ctx)
	sess, err := s.sessionSvc.Get(ctx, sessionID)
	if err != nil {
		sess = session.NewSession(sessionID, GetUserID(ctx), nil)
	}

	// 如果处于挂起确认状态，则执行自动取消逻辑
	if sess.IsPendingConfirmation() {
		sess.CancelPendingConfirmations()
	}

	sess.Messages = append(sess.Messages, messages...)
	if err := s.sessionSvc.Save(ctx, sess); err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *agentService) runStreamLoop(ctx context.Context, sess *session.Session, opts []chat.OptionFunc) error {
	chatOpt := &chat.Option{}
	for _, fn := range opts {
		fn(chatOpt)
	}
	if chatOpt.MaxIterations == nil {
		val := defaultMaxIter
		chatOpt.MaxIterations = &val
	}

	for iter := 0; iter < *chatOpt.MaxIterations; iter++ {
		// stream请求，并聚合响应消息分片
		assistantMsg, err := func() (*message.Message, error) {
			stream, err := s.chatModel.Stream(ctx, sess.Messages, opts...)
			if err != nil {
				return nil, err
			}
			defer func() {
				_ = stream.Close()
			}()
			return message.ReadAndAssembleMessage(stream, func(msg *message.StreamMessage) {
				if msg.IsDelta() {
					_ = s.publishStreamMessage(ctx, sess, msg)
				}
			})
		}()

		if err != nil {
			return err
		}

		// 响应追加会话并保存
		sess.Messages = append(sess.Messages, *assistantMsg)
		if err := s.sessionSvc.Save(ctx, sess); err != nil {
			return err
		}

		toolResults := utils.ParallelMap(ctx, assistantMsg.ToolCalls, func(taskCtx context.Context, tc message.ToolCall) (message.Message, error) {
			_ = s.publishStreamMessage(taskCtx, sess, message.NewStreamMessageToolCall(&tc))
			toolMsg, toolErr := ExecuteToolCall(taskCtx, tc, chatOpt.Tools[tc.Function.Name])
			_ = s.publishStreamMessage(taskCtx, sess, message.NewStreamMessageToolResult(tc.ID, toolMsg.Content, toolErr))
			return toolMsg, toolErr
		}, utils.WithFailFast(false))

		// 遍历工具执行结果，分三类处理：成功追加消息、待确认收集、其他失败追加错误消息
		var pendingConfirms []message.ToolCall
		for _, res := range toolResults {
			if res.Error == nil {
				sess.Messages = append(sess.Messages, res.O)
			} else if _, ok := res.Error.(*ToolConfirmationRequiredError); ok {
				pendingConfirms = append(pendingConfirms, res.I)
			} else {
				sess.Messages = append(sess.Messages, message.NewToolMessage(res.I.ID, res.Error.Error()))
			}
		}

		// 有待确认项：应用中断状态后提前返回（save best-effort）
		if len(pendingConfirms) > 0 {
			interruptErr := session.NewPendingConfirmationError(sess.ID, assistantMsg, pendingConfirms)
			sess.ApplyInterrupt(interruptErr)
			_ = s.sessionSvc.Save(ctx, sess)
			return interruptErr
		}

		if err := s.sessionSvc.Save(ctx, sess); err != nil {
			return err
		}
	}

	return fmt.Errorf("agent reached max iterations (%d) without resolving", *chatOpt.MaxIterations)
}

// handleStreamEvent 监听事件总线投递的流消息事件，并根据 SessionID 分发给对应活跃的通道读取器
func (s *agentService) handleStreamEvent(ctx context.Context, ev event.Event) error {
	streamEv, ok := ev.Payload().(*session.SessionStreamEvent)
	if !ok {
		return nil
	}

	s.mu.RLock()
	reader, exists := s.readers[streamEv.SessionID()]
	s.mu.RUnlock()

	if !exists {
		return nil
	}

	// 如果收到结束事件，则在该 Session 的分发链路尾端注销 readers 映射，并安全关闭 reader.ch
	if streamEv.IsLast() {
		s.mu.Lock()
		delete(s.readers, streamEv.SessionID())
		s.mu.Unlock()

		if streamEv.Error() != nil {
			reader.sendErr(streamEv.Error())
		}
		close(reader.ch)
		return nil
	}

	_ = reader.Send(streamEv.Message())
	return nil
}

// publishStreamMessage 向事件总线发布一条 SessionStreamEvent 领域事件
func (s *agentService) publishStreamMessage(ctx context.Context, sess *session.Session, msg *message.StreamMessage) error {
	if s.publisher == nil {
		return nil
	}
	ev := session.NewSessionStreamEvent(
		uuid.New().String(),
		sess.ID,
		sess.UserID,
		msg,
	)
	return s.publisher.Publish(ctx, ev)
}

// publishEndEvent 向事件总线发布一条代表流结束的 SessionStreamEvent 领域事件
func (s *agentService) publishEndEvent(ctx context.Context, sess *session.Session, err error) error {
	if s.publisher == nil {
		return nil
	}
	ev := session.NewSessionStreamEndEvent(
		uuid.New().String(),
		sess.ID,
		sess.UserID,
		err,
	)
	return s.publisher.Publish(ctx, ev)
}
