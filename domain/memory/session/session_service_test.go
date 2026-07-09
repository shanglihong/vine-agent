package session_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"vine-agent/domain/memory/session"
	"vine-agent/domain/memory/session/mock"
	"vine-agent/domain/message"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionService_Save(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockSessionRepository(ctrl)
	// 设置 TTL 确保缓存有效，便于在测试中观察 Save 对缓存失效的作用
	service := session.NewSessionService(mockRepo)
	ctx := context.Background()

	t.Run("nil session should return error", func(t *testing.T) {
		err := service.Save(ctx, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session cannot be nil")
	})

	t.Run("successful save and cache invalidation", func(t *testing.T) {
		sess := &session.Session{
			ID:     "session-1",
			UserID: "user-1",
		}

		// 1. 模拟物理保存成功
		mockRepo.EXPECT().Save(ctx, sess).Return(nil).Times(1)

		err := service.Save(ctx, sess)
		require.NoError(t, err)

		// 2. 模拟从物理存储获取（由于 Save 让缓存失效，所以 Get 必须穿透到底层物理存储读取）
		mockRepo.EXPECT().Get(ctx, sess.ID).Return(sess, nil).Times(1)

		got, err := service.Get(ctx, sess.ID)
		require.NoError(t, err)
		assert.Equal(t, sess.ID, got.ID)

		// 3. 再次 Get（由于第 2 步中 Get 会把数据写入缓存，这次应当直接从缓存读取，不穿透到 repo）
		got2, err := service.Get(ctx, sess.ID)
		require.NoError(t, err)
		assert.Equal(t, sess.ID, got2.ID)
	})

	t.Run("repository save error", func(t *testing.T) {
		sess := &session.Session{
			ID:     "session-2",
			UserID: "user-1",
		}
		expectedErr := errors.New("db save error")
		mockRepo.EXPECT().Save(ctx, sess).Return(expectedErr).Times(1)

		err := service.Save(ctx, sess)
		assert.Equal(t, expectedErr, err)
	})
}

func TestSessionService_Get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockSessionRepository(ctrl)
	service := session.NewSessionService(mockRepo)
	ctx := context.Background()

	t.Run("cache miss and read from persist successfully", func(t *testing.T) {
		sess := &session.Session{
			ID:       "session-3",
			UserID:   "user-2",
			Metadata: map[string]string{"key": "value"},
			Messages: []message.Message{
				{Role: message.RoleUser, Content: "hi"},
			},
		}

		// 第一次获取，缓存未命中，调用物理存储且只调用一次
		mockRepo.EXPECT().Get(ctx, sess.ID).Return(sess, nil).Times(1)

		got1, err := service.Get(ctx, sess.ID)
		require.NoError(t, err)
		assert.Equal(t, sess.ID, got1.ID)
		assert.Equal(t, "value", got1.Metadata["key"])
		assert.Len(t, got1.Messages, 1)

		// 修改返回的 session（验证深拷贝，防止缓存内容被外部修改）
		got1.Metadata["key"] = "modified"

		// 第二次获取，应该命中缓存，不再调用物理存储
		got2, err := service.Get(ctx, sess.ID)
		require.NoError(t, err)
		assert.Equal(t, sess.ID, got2.ID)
		// 验证第二次获取的是原始值，说明深拷贝生效了
		assert.Equal(t, "value", got2.Metadata["key"])
	})

	t.Run("persist get error", func(t *testing.T) {
		expectedErr := errors.New("db get error")
		mockRepo.EXPECT().Get(ctx, "session-4").Return(nil, expectedErr).Times(1)

		got, err := service.Get(ctx, "session-4")
		assert.Nil(t, got)
		assert.Equal(t, expectedErr, err)
	})
}

func TestSessionService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockSessionRepository(ctrl)
	service := session.NewSessionService(mockRepo)
	ctx := context.Background()

	t.Run("successful delete and cache invalidation", func(t *testing.T) {
		sess := &session.Session{
			ID:     "session-5",
			UserID: "user-3",
		}

		// 1. 模拟写入缓存
		mockRepo.EXPECT().Get(ctx, sess.ID).Return(sess, nil).Times(1)
		_, err := service.Get(ctx, sess.ID)
		require.NoError(t, err)

		// 2. 物理删除成功
		mockRepo.EXPECT().Delete(ctx, sess.ID).Return(nil).Times(1)
		err = service.Delete(ctx, sess.ID)
		require.NoError(t, err)

		// 3. 再次获取（由于缓存被 Delete 清空，它必须再次向物理层请求，这里模拟物理层返回未找到）
		mockRepo.EXPECT().Get(ctx, sess.ID).Return(nil, session.ErrSessionNotFound).Times(1)
		_, err = service.Get(ctx, sess.ID)
		assert.Equal(t, session.ErrSessionNotFound, err)
	})

	t.Run("repository delete error", func(t *testing.T) {
		expectedErr := errors.New("db delete error")
		mockRepo.EXPECT().Delete(ctx, "session-6").Return(expectedErr).Times(1)

		err := service.Delete(ctx, "session-6")
		assert.Equal(t, expectedErr, err)
	})
}

func TestSessionService_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockSessionRepository(ctrl)
	service := session.NewSessionService(mockRepo)
	ctx := context.Background()

	t.Run("successful list", func(t *testing.T) {
		sessions := []*session.Session{
			{ID: "sess-a", UserID: "user-list"},
			{ID: "sess-b", UserID: "user-list"},
		}

		mockRepo.EXPECT().List(ctx, "user-list").Return(sessions, nil).Times(1)

		got, err := service.List(ctx, "user-list")
		require.NoError(t, err)
		assert.Len(t, got, 2)
		assert.Equal(t, "sess-a", got[0].ID)
		assert.Equal(t, "sess-b", got[1].ID)
	})

	t.Run("repository list error", func(t *testing.T) {
		expectedErr := errors.New("db list error")
		mockRepo.EXPECT().List(ctx, "user-list").Return(nil, expectedErr).Times(1)

		got, err := service.List(ctx, "user-list")
		assert.Nil(t, got)
		assert.Equal(t, expectedErr, err)
	})
}

func TestSessionService_Rename(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockSessionRepository(ctrl)
	service := session.NewSessionService(mockRepo)
	ctx := context.Background()

	t.Run("successful rename", func(t *testing.T) {
		sess := &session.Session{
			ID:     "session-rename",
			UserID: "user-1",
			Name:   "old-name",
		}

		// 1. 模拟首次 Get 从物理存储读取
		mockRepo.EXPECT().Get(ctx, sess.ID).Return(sess, nil).Times(1)

		// 2. 模拟物理保存（包含新名字）
		mockRepo.EXPECT().Save(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, s *session.Session) error {
			assert.Equal(t, "new-name", s.Name)
			return nil
		}).Times(1)

		err := service.Rename(ctx, sess.ID, "new-name")
		require.NoError(t, err)
	})

	t.Run("rename get error", func(t *testing.T) {
		expectedErr := errors.New("db get error")
		mockRepo.EXPECT().Get(ctx, "session-rename-fail").Return(nil, expectedErr).Times(1)

		err := service.Rename(ctx, "session-rename-fail", "new-name")
		assert.Equal(t, expectedErr, err)
	})
}

func TestSessionService_ListUpdatedSince(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockSessionRepository(ctrl)
	service := session.NewSessionService(mockRepo)
	ctx := context.Background()

	t.Run("successful list updated since", func(t *testing.T) {
		since := time.Now()
		sessions := []*session.Session{
			{ID: "sess-a", UserID: "user-list"},
		}

		mockRepo.EXPECT().ListUpdatedSince(ctx, since).Return(sessions, nil).Times(1)

		got, err := service.ListUpdatedSince(ctx, since)
		require.NoError(t, err)
		assert.Len(t, got, 1)
		assert.Equal(t, "sess-a", got[0].ID)
	})
}

func TestSessionService_GetBatch(t *testing.T) {
	ctx := context.Background()

	t.Run("successful batch get", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := mock.NewMockSessionRepository(ctrl)
		service := session.NewSessionService(mockRepo)

		sess1 := &session.Session{ID: "sess-1", UserID: "user-1"}
		sess2 := &session.Session{ID: "sess-2", UserID: "user-1"}

		mockRepo.EXPECT().GetBatch(ctx, []string{"sess-1", "sess-2"}).Return(map[string]*session.Session{
			"sess-1": sess1,
			"sess-2": sess2,
		}, nil).Times(1)

		got, err := service.GetBatch(ctx, []string{"sess-1", "sess-2"})
		require.NoError(t, err)
		assert.Len(t, got, 2)
		assert.Equal(t, "sess-1", got["sess-1"].ID)
		assert.Equal(t, "sess-2", got["sess-2"].ID)
	})

	t.Run("partial error get", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := mock.NewMockSessionRepository(ctrl)
		service := session.NewSessionService(mockRepo)

		sess1 := &session.Session{ID: "sess-1", UserID: "user-1"}

		mockRepo.EXPECT().GetBatch(ctx, []string{"sess-1", "sess-2"}).Return(map[string]*session.Session{
			"sess-1": sess1,
		}, errors.New("db error")).Times(1)

		got, err := service.GetBatch(ctx, []string{"sess-1", "sess-2"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
		assert.Len(t, got, 1)
		assert.Equal(t, "sess-1", got["sess-1"].ID)
	})
}
