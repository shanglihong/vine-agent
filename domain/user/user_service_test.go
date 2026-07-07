package user_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"vine-agent/domain/user"
	"vine-agent/domain/user/mock"
)

func TestUserService_GetUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockUserRepository(ctrl)
	svc := user.NewUserService(mockRepo)

	ctx := context.Background()

	t.Run("empty user id should return error", func(t *testing.T) {
		u, err := svc.GetUser(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, u)
		assert.Contains(t, err.Error(), "user id cannot be empty")
	})

	t.Run("user not found should return custom error", func(t *testing.T) {
		mockRepo.EXPECT().Get(ctx, "nonexistent").Return(nil, user.ErrUserNotFound).Times(1)

		u, err := svc.GetUser(ctx, "nonexistent")
		assert.ErrorIs(t, err, user.ErrUserNotFound)
		assert.Nil(t, u)
	})

	t.Run("successful get user", func(t *testing.T) {
		expectedUser := &user.User{
			ID:        "user_test_999",
			Username:  "TestUser",
			Email:     "test@example.com",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo.EXPECT().Get(ctx, "user_test_999").Return(expectedUser, nil).Times(1)

		u, err := svc.GetUser(ctx, "user_test_999")
		assert.NoError(t, err)
		assert.Equal(t, expectedUser, u)
	})
}
