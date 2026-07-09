package agent_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"vine-agent/app/agent"
	chat_mock "vine-agent/domain/chat/mock"
	"vine-agent/domain/memory/session"
	session_mock "vine-agent/domain/memory/session/mock"
	"vine-agent/domain/message"
)

type customStreamReader struct {
	recvFunc      func() (*message.StreamMessage, error)
	closeFunc     func() error
	interruptFunc func() error
}

func (c *customStreamReader) Recv() (*message.StreamMessage, error) {
	if c.recvFunc != nil {
		return c.recvFunc()
	}
	return nil, nil
}

func (c *customStreamReader) Close() error {
	if c.closeFunc != nil {
		return c.closeFunc()
	}
	return nil
}

func (c *customStreamReader) Interrupt() error {
	if c.interruptFunc != nil {
		return c.interruptFunc()
	}
	return c.Close()
}

func TestAgentService_Stream_Interrupt(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChat := chat_mock.NewMockChatModel(ctrl)
	mockSessionSvc := session_mock.NewMockSessionService(ctrl)

	sessionID := "test-session"
	userID := "test-user"
	ctx := context.Background()
	ctx = agent.WithSessionID(ctx, sessionID)
	ctx = agent.WithUserID(ctx, userID)

	t.Run("Normal Error: should NOT append InterruptedMessage", func(t *testing.T) {
		sess := session.NewSession(sessionID, userID, nil)
		mockSessionSvc.EXPECT().Get(gomock.Any(), sessionID).Return(sess, nil).Times(1)

		mockReader := &customStreamReader{
			recvFunc: func() (*message.StreamMessage, error) {
				return nil, errors.New("mock model internal error")
			},
		}
		mockChat.EXPECT().Stream(gomock.Any(), gomock.Any(), gomock.Any()).Return(mockReader, nil).Times(1)

		var savedSess *session.Session
		mockSessionSvc.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, s *session.Session) error {
			savedSess = s
			return nil
		}).Times(1)

		svc := agent.NewService(mockChat, mockSessionSvc)
		userMsg := message.Message{Role: message.RoleUser, Content: "hello"}
		reader, err := svc.Stream(ctx, []message.Message{userMsg})
		assert.NoError(t, err)

		// 消费读取器直到出错
		_, err = reader.Recv()
		assert.Error(t, err)

		// 稍微等待下异步循环运行完毕
		time.Sleep(50 * time.Millisecond)

		assert.NotNil(t, savedSess)
		// 应该只有 1 条消息（用户输入），不包含 Interrupted 消息
		assert.Len(t, savedSess.Messages, 1)
		assert.Equal(t, message.RoleUser, savedSess.Messages[0].Role)
	})

	t.Run("User Interrupt: should append InterruptedMessage", func(t *testing.T) {
		sess := session.NewSession(sessionID, userID, nil)
		mockSessionSvc.EXPECT().Get(gomock.Any(), sessionID).Return(sess, nil).Times(1)

		blockCh := make(chan struct{})
		mockReader := &customStreamReader{
			recvFunc: func() (*message.StreamMessage, error) {
				<-blockCh // 阻塞直到测试显式解阻塞
				return nil, context.Canceled
			},
		}
		mockChat.EXPECT().Stream(gomock.Any(), gomock.Any(), gomock.Any()).Return(mockReader, nil).Times(1)

		var savedSess *session.Session
		mockSessionSvc.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, s *session.Session) error {
			savedSess = s
			return nil
		}).Times(2)

		svc := agent.NewService(mockChat, mockSessionSvc)
		userMsg := message.Message{Role: message.RoleUser, Content: "hello"}
		reader, err := svc.Stream(ctx, []message.Message{userMsg})
		assert.NoError(t, err)

		// 验证 reader 实现了中断接口
		type interrupter interface {
			Interrupt() error
		}
		it, ok := reader.(interrupter)
		assert.True(t, ok)

		// 触发中断
		err = it.Interrupt()
		assert.NoError(t, err)

		// 释放阻塞
		close(blockCh)

		// 稍微等待下异步循环运行完毕
		time.Sleep(50 * time.Millisecond)

		// 检查最终保存的 Session 是否追加了 Interrupted 消息
		assert.NotNil(t, savedSess)
		assert.Len(t, savedSess.Messages, 2)
		assert.Equal(t, message.RoleUser, savedSess.Messages[0].Role)
		assert.Equal(t, message.RoleInterrupted, savedSess.Messages[1].Role)
	})
}
