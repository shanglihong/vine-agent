package retrieval_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"vine-agent/domain/memory/retrieval"
	"vine-agent/domain/memory/retrieval/mock"
	"vine-agent/domain/message"
)

func TestRetrievalService_IndexMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockRetrievalRepository(ctrl)
	svc := retrieval.NewRetrievalService(mockRepo)
	ctx := context.Background()

	t.Run("成功索引非空消息", func(t *testing.T) {
		msg := message.Message{Role: message.RoleUser, Content: "你好，全文本检索"}
		mockRepo.EXPECT().Save(ctx, "session-1", "user-1", msg).Return(nil).Times(1)

		err := svc.IndexMessage(ctx, "session-1", "user-1", msg)
		assert.NoError(t, err)
	})

	t.Run("忽略空内容消息", func(t *testing.T) {
		msg := message.Message{Role: message.RoleUser, Content: ""}
		mockRepo.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		err := svc.IndexMessage(ctx, "session-1", "user-1", msg)
		assert.NoError(t, err)
	})

	t.Run("缺少 SessionID 报错", func(t *testing.T) {
		msg := message.Message{Role: message.RoleUser, Content: "test"}
		err := svc.IndexMessage(ctx, "", "user-1", msg)
		assert.Error(t, err)
	})

	t.Run("仓储返回错误时应传递错误", func(t *testing.T) {
		msg := message.Message{Role: message.RoleUser, Content: "test"}
		mockRepo.EXPECT().Save(ctx, "session-1", "user-1", msg).Return(errors.New("db error")).Times(1)

		err := svc.IndexMessage(ctx, "session-1", "user-1", msg)
		assert.EqualError(t, err, "db error")
	})
}

func TestRetrievalService_SearchSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockRetrievalRepository(ctrl)
	svc := retrieval.NewRetrievalService(mockRepo)
	ctx := context.Background()

	t.Run("成功检索消息列表", func(t *testing.T) {
		expectedMsgs := []message.Message{
			{Role: message.RoleUser, Content: "你好，全文本检索"},
		}
		mockRepo.EXPECT().SearchSession(ctx, "session-1", "检索", 10).Return(expectedMsgs, nil).Times(1)

		res, err := svc.SearchSession(ctx, "session-1", "检索", 10)
		assert.NoError(t, err)
		assert.Equal(t, expectedMsgs, res)
	})

	t.Run("空查询词直接返回空结果", func(t *testing.T) {
		res, err := svc.SearchSession(ctx, "session-1", "", 10)
		assert.NoError(t, err)
		assert.Nil(t, res)
	})
}

func TestRetrievalService_SearchUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockRetrievalRepository(ctrl)
	svc := retrieval.NewRetrievalService(mockRepo)
	ctx := context.Background()

	t.Run("跨会话成功检索", func(t *testing.T) {
		expectedResults := []retrieval.SearchResult{
			{
				SessionID: "session-1",
				UserID:    "user-1",
				Message:   message.Message{Role: message.RoleUser, Content: "你好，全文本检索"},
			},
		}
		mockRepo.EXPECT().SearchUser(ctx, "user-1", "检索", 20).Return(expectedResults, nil).Times(1)

		res, err := svc.SearchUser(ctx, "user-1", "检索", 0)
		assert.NoError(t, err)
		assert.Equal(t, expectedResults, res)
	})
}

func TestRetrievalService_ClearSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockRetrievalRepository(ctrl)
	svc := retrieval.NewRetrievalService(mockRepo)
	ctx := context.Background()

	t.Run("成功清除会话索引", func(t *testing.T) {
		mockRepo.EXPECT().DeleteBySession(ctx, "session-1").Return(nil).Times(1)

		err := svc.ClearSession(ctx, "session-1")
		assert.NoError(t, err)
	})
}
