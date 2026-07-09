package agent

import (
	"context"
	"errors"
	"fmt"

	"vine-agent/domain/chat"
	"vine-agent/domain/memory/session"
	"vine-agent/domain/message"
	"vine-agent/utils"
)

const defaultMaxIter int = 5

var _ Service = (*agentService)(nil)

type toolBeforeFunc func(message.ToolCall)
type toolAfterFunc func(message.ToolCall, message.Message, error)

// agentService 智能体应用层服务，整合了会话记忆持久化与多轮工具调用流迭代逻辑
type agentService struct {
	chatModel  chat.ChatModel
	sessionSvc session.SessionService
}

// NewService 构造一个新的 Service 实例
func NewService(
	chatModel chat.ChatModel,
	sessionSvc session.SessionService,
) Service {
	svc := &agentService{
		chatModel:  chatModel,
		sessionSvc: sessionSvc,
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

	sess.AppendMessage(*resp)
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

	// 初始化读取器
	subCtx, cancel := context.WithCancel(ctx)
	reader := &agentEventReader{
		ch:     make(chan *message.StreamMessage, 100),
		errCh:  make(chan error, 1),
		ctx:    subCtx,
		cancel: cancel,
	}

	go func() {
		defer reader.closeChannel()
		loopErr := s.runStreamLoop(subCtx, reader, sess, opts)
		if loopErr != nil {
			if reader.IsUserCancelled() {
				// 用户取消，不发送错误
			} else {
				reader.sendErr(loopErr)
			}
		}
		if reader.IsUserCancelled() {
			sess.AppendMessage(message.NewInterruptedMessage())
			_ = s.sessionSvc.Save(context.WithoutCancel(ctx), sess)
		}
	}()

	return reader, nil
}

func (s *agentService) acceptUserMessages(ctx context.Context, messages []message.Message) (*session.Session, error) {
	sessionID, _ := GetSessionID(ctx)
	sess, err := s.sessionSvc.Get(ctx, sessionID)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			sess = session.NewSession(sessionID, GetUserID(ctx), nil)
		} else {
			return nil, err
		}
	}

	// 如果处于挂起确认状态，则执行自动取消逻辑
	if sess.IsPendingConfirmation() {
		sess.CancelPendingConfirmations()
	}

	// 如果会话还没有名字且有新消息，将第一条消息内容设为名字
	if sess.Name == "" && len(messages) > 0 {
		sess.Name = messages[0].Content
	}

	sess.AppendMessages(messages)
	if err := s.sessionSvc.Save(ctx, sess); err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *agentService) runStreamLoop(ctx context.Context, reader *agentEventReader, sess *session.Session, opts []chat.OptionFunc) error {
	chatOpt := &chat.Option{}
	for _, fn := range opts {
		fn(chatOpt)
	}
	if chatOpt.MaxIterations == nil {
		val := defaultMaxIter
		chatOpt.MaxIterations = &val
	}

	for iter := 0; iter < *chatOpt.MaxIterations; iter++ {
		assistantMsg, err := s.stream(ctx, sess.Messages, opts, func(msg *message.StreamMessage) {
			if msg.IsDelta() {
				reader.Send(msg)
			}
		})
		if err != nil {
			return err
		}
		sess.AppendMessage(*assistantMsg)
		if sessionErr := s.sessionSvc.Save(ctx, sess); sessionErr != nil {
			return sessionErr
		}

		// 没有工具调用，结束循环
		if !assistantMsg.HasToolCalls() {
			return nil
		}

		toolBeforeExc := func(tc message.ToolCall) {
			reader.Send(message.NewStreamMessageToolCall(&tc))
		}
		toolAfterExc := func(tc message.ToolCall, m message.Message, err error) {
			reader.Send(message.NewStreamMessageToolResult(tc.ID, m.Content, err))
		}

		results, pendingResults := s.toolExc(ctx, toolBeforeExc, toolAfterExc, assistantMsg, chatOpt)
		if len(results) > 0 {
			sess.AppendMessages(results)
			if sessionErr := s.sessionSvc.Save(ctx, sess); sessionErr != nil {
				return sessionErr
			}
		}
		if len(pendingResults) > 0 {
			interruptErr := session.NewPendingConfirmationError(sess.ID, assistantMsg, pendingResults)
			sess.ApplyInterrupt(interruptErr)
			_ = s.sessionSvc.Save(ctx, sess)
			return interruptErr
		}
	}

	return fmt.Errorf("agent reached max iterations (%d) without resolving", *chatOpt.MaxIterations)
}

func (s *agentService) toolExc(ctx context.Context,
	toolBeforeFunc toolBeforeFunc,
	toolAfterFunc toolAfterFunc,
	assistantMsg *message.Message,
	chatOpt *chat.Option) ([]message.Message, []message.ToolCall) {

	// 工具调用
	toolResults := utils.ParallelMap(ctx, assistantMsg.ToolCalls, func(taskCtx context.Context, tc message.ToolCall) (message.Message, error) {
		toolBeforeFunc(tc)
		toolMsg, toolErr := ExecuteToolCall(taskCtx, tc, chatOpt.Tools[tc.Function.Name])
		toolAfterFunc(tc, toolMsg, toolErr)
		return toolMsg, toolErr
	}, utils.WithFailFast(false))

	// 遍历工具执行结果，分三类处理：成功追加消息、待确认收集、其他失败追加错误消息
	var toolConfirmationRequiredError *ToolConfirmationRequiredError
	var pendingConfirms []message.ToolCall
	var messages []message.Message
	for _, res := range toolResults {
		if res.Error == nil {
			messages = append(messages, res.O)
		} else if errors.As(res.Error, &toolConfirmationRequiredError) {
			pendingConfirms = append(pendingConfirms, res.I)
		} else {
			messages = append(messages, message.NewToolMessage(res.I.ID, res.Error.Error()))
		}
	}

	return messages, pendingConfirms
}

func (s *agentService) stream(ctx context.Context, messages []message.Message, opts []chat.OptionFunc, callback func(*message.StreamMessage)) (*message.Message, error) {
	stream, err := s.chatModel.Stream(ctx, messages, opts...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = stream.Close()
	}()

	assistantMsg, err := message.ReadAndAssembleMessage(stream, callback)
	if err != nil {
		return nil, err
	}

	return assistantMsg, nil
}
