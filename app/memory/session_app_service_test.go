package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"vine-agent/app/memory"
	sessionmock "vine-agent/domain/memory/session/mock"
	projectmock "vine-agent/domain/project/mock"
)

func TestSessionAppService_CreateSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSessionSvc := sessionmock.NewMockSessionService(ctrl)
	mockProjectSvc := projectmock.NewMockProjectService(ctrl)
	appSvc := memory.NewSessionAppService(mockSessionSvc, mockProjectSvc)

	ctx := context.Background()

	t.Run("创建会话不关联项目", func(t *testing.T) {
		mockSessionSvc.EXPECT().
			Save(ctx, gomock.Any()).
			Return(nil).
			Times(1)

		sess, err := appSvc.CreateSession(ctx, "sess-1", "user-1", "")
		assert.NoError(t, err)
		assert.NotNil(t, sess)
		assert.Equal(t, "sess-1", sess.ID)
	})

	t.Run("创建会话并绑定项目成功", func(t *testing.T) {
		mockSessionSvc.EXPECT().
			Save(ctx, gomock.Any()).
			Return(nil).
			Times(1)

		mockProjectSvc.EXPECT().
			BindSession(ctx, "proj-1", "sess-2").
			Return(nil).
			Times(1)

		sess, err := appSvc.CreateSession(ctx, "sess-2", "user-1", "proj-1")
		assert.NoError(t, err)
		assert.NotNil(t, sess)
	})

	t.Run("创建会话绑定项目失败容错（最终一致性）", func(t *testing.T) {
		mockSessionSvc.EXPECT().
			Save(ctx, gomock.Any()).
			Return(nil).
			Times(1)

		mockProjectSvc.EXPECT().
			BindSession(ctx, "proj-1", "sess-3").
			Return(errors.New("db error")).
			Times(1)

		sess, err := appSvc.CreateSession(ctx, "sess-3", "user-1", "proj-1")
		// 最终一致性绑定失败不阻断，但返回 error 供外层感知
		assert.Error(t, err)
		assert.NotNil(t, sess)
		assert.Equal(t, "sess-3", sess.ID)
	})
}
