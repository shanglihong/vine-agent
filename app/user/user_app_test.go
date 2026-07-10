package user_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"vine-agent/app/memory"
	"vine-agent/app/user"
	"vine-agent/domain/memory/profile"
	profilemock "vine-agent/domain/memory/profile/mock"
	"vine-agent/domain/memory/session"
	sessionmock "vine-agent/domain/memory/session/mock"
	usermock "vine-agent/domain/user/mock"
)

func TestUserAppService_GetUserProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := usermock.NewMockUserService(ctrl)
	mockSessionSvc := sessionmock.NewMockSessionService(ctrl)
	mockProfileRepo := profilemock.NewMockProfileRepository(ctrl)
	mockEvolveSvc := profilemock.NewMockEvolutionService(ctrl)

	evolutionAppSvc := memory.NewEvolutionAppService(mockSessionSvc, mockProfileRepo, mockEvolveSvc)
	appSvc := user.NewUserAppService(mockUserSvc, mockSessionSvc, mockProfileRepo, evolutionAppSvc)

	ctx := context.Background()

	t.Run("直接通过 UserID 获取画像", func(t *testing.T) {
		mockProfileRepo.EXPECT().
			GetByUserID(ctx, "user-1").
			Return(&profile.Profile{UserID: "user-1", Preferences: []string{"test-pref"}}, nil).
			Times(1)

		prof, err := appSvc.GetUserProfile(ctx, "user-1")
		assert.NoError(t, err)
		assert.NotNil(t, prof)
		assert.Equal(t, "user-1", prof.UserID)
	})

	t.Run("通过 SessionID 反推 UserID 并获取画像", func(t *testing.T) {
		mockSessionSvc.EXPECT().
			Get(ctx, "sess_123").
			Return(&session.Session{ID: "sess_123", UserID: "user-inferred"}, nil).
			Times(1)

		mockProfileRepo.EXPECT().
			GetByUserID(ctx, "user-inferred").
			Return(&profile.Profile{UserID: "user-inferred"}, nil).
			Times(1)

		prof, err := appSvc.GetUserProfile(ctx, "sess_123")
		assert.NoError(t, err)
		assert.NotNil(t, prof)
		assert.Equal(t, "user-inferred", prof.UserID)
	})
}
