package project_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"vine-agent/domain/memory/session"
	sessionmock "vine-agent/domain/memory/session/mock"
	"vine-agent/domain/project"
	projectmock "vine-agent/domain/project/mock"
)

func TestProjectService_CreateProject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := projectmock.NewMockProjectRepository(ctrl)
	mockSessionSvc := sessionmock.NewMockSessionService(ctrl)
	service := project.NewProjectService(mockRepo, mockSessionSvc, "/test-root")

	ctx := context.Background()

	t.Run("成功创建项目", func(t *testing.T) {
		mockRepo.EXPECT().
			Save(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, proj *project.Project) error {
				assert.Equal(t, "user-1", proj.UserID)
				assert.Equal(t, "proj-name", proj.Name)
				assert.Contains(t, proj.Path, "/test-root")
				assert.Equal(t, "desc", proj.Description)
				return nil
			}).
			Times(1)

		proj, err := service.CreateProject(ctx, "user-1", "proj-name", "desc", nil)
		assert.NoError(t, err)
		assert.NotNil(t, proj)
	})

	t.Run("参数缺失报错", func(t *testing.T) {
		_, err := service.CreateProject(ctx, "", "name", "", nil)
		assert.Error(t, err)

		_, err = service.CreateProject(ctx, "user-1", "", "", nil)
		assert.Error(t, err)
	})
}

func TestProjectService_DeleteProject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := projectmock.NewMockProjectRepository(ctrl)
	mockSessionSvc := sessionmock.NewMockSessionService(ctrl)
	service := project.NewProjectService(mockRepo, mockSessionSvc, "/test-root")

	ctx := context.Background()

	t.Run("级联删除成功", func(t *testing.T) {
		projectID := "proj-123"
		sessionIDs := []string{"sess-1", "sess-2"}

		// 1. 获取关联的 sessionID
		mockRepo.EXPECT().
			ListSessionsByProject(ctx, projectID).
			Return(sessionIDs, nil).
			Times(1)

		// 2. 依次物理删除每个 session 实体
		mockSessionSvc.EXPECT().
			Delete(ctx, "sess-1").
			Return(nil).
			Times(1)

		mockSessionSvc.EXPECT().
			Delete(ctx, "sess-2").
			Return(nil).
			Times(1)

		// 3. 删除项目及关联表
		mockRepo.EXPECT().
			Delete(ctx, projectID).
			Return(nil).
			Times(1)

		err := service.DeleteProject(ctx, projectID)
		assert.NoError(t, err)
	})

	t.Run("session删除出错则阻断", func(t *testing.T) {
		projectID := "proj-123"
		sessionIDs := []string{"sess-1"}

		mockRepo.EXPECT().
			ListSessionsByProject(ctx, projectID).
			Return(sessionIDs, nil).
			Times(1)

		mockSessionSvc.EXPECT().
			Delete(ctx, "sess-1").
			Return(errors.New("db error")).
			Times(1)

		err := service.DeleteProject(ctx, projectID)
		assert.Error(t, err)
	})
}

func TestProjectService_ListSessionsByProject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := projectmock.NewMockProjectRepository(ctrl)
	mockSessionSvc := sessionmock.NewMockSessionService(ctrl)
	service := project.NewProjectService(mockRepo, mockSessionSvc, "/test-root")

	ctx := context.Background()
	projectID := "proj-1"
	sessionIDs := []string{"sess-1", "sess-2"}

	mockRepo.EXPECT().
		ListSessionsByProject(ctx, projectID).
		Return(sessionIDs, nil).
		Times(1)

	mockSessionsMap := map[string]*session.Session{
		"sess-1": {ID: "sess-1", UserID: "user-1", Name: "Session 1"},
		"sess-2": {ID: "sess-2", UserID: "user-1", Name: "Session 2"},
	}

	mockSessionSvc.EXPECT().
		GetBatch(ctx, sessionIDs).
		Return(mockSessionsMap, nil).
		Times(1)

	list, err := service.ListSessionsByProject(ctx, projectID)
	assert.NoError(t, err)
	assert.Len(t, list, 2)
	assert.Equal(t, "sess-1", list[0].ID)
	assert.Equal(t, "sess-2", list[1].ID)
}
