package profile_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"vine-agent/domain/memory/profile"
	"vine-agent/domain/memory/profile/mock"
	"vine-agent/domain/message"
)

func TestEvolutionService_Evolve(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExtractor := mock.NewMockExtractor(ctrl)
	svc := profile.NewEvolutionService(mockExtractor)

	ctx := context.Background()

	t.Run("empty messages should do nothing", func(t *testing.T) {
		prof := profile.NewProfile("user123")
		err := svc.Evolve(ctx, prof, nil)
		assert.NoError(t, err)
		assert.Empty(t, prof.Preferences)
		assert.Empty(t, prof.Facts)
	})

	t.Run("successful extraction and update", func(t *testing.T) {
		prof := profile.NewProfile("user123")
		prof.Preferences = []string{"喜欢写 Python"}
		prof.Facts = []string{"现居北京"}

		messages := []message.Message{
			{Role: message.RoleUser, Content: "我最近主要写 Go 语言了"},
		}

		latestPrefs := []string{"喜欢写 Go 语言"}
		latestFacts := []string{"现居北京"}

		mockExtractor.EXPECT().
			Extract(ctx, messages, gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, msgs []message.Message, prefs []string, facts []string) ([]string, []string, error) {
				assert.Len(t, prefs, 1)
				assert.Equal(t, "喜欢写 Python", prefs[0])

				assert.Len(t, facts, 1)
				assert.Equal(t, "现居北京", facts[0])

				return latestPrefs, latestFacts, nil
			}).
			Times(1)

		err := svc.Evolve(ctx, prof, messages)
		assert.NoError(t, err)

		// 验证聚合根已经执行了覆盖更新
		assert.Len(t, prof.Preferences, 1)
		assert.Equal(t, "喜欢写 Go 语言", prof.Preferences[0])

		assert.Len(t, prof.Facts, 1)
		assert.Equal(t, "现居北京", prof.Facts[0])
	})

	t.Run("extractor returns error", func(t *testing.T) {
		prof := profile.NewProfile("user123")
		messages := []message.Message{
			{Role: message.RoleUser, Content: "测试错误情况"},
		}

		expectedErr := errors.New("llm extraction failed")
		mockExtractor.EXPECT().
			Extract(ctx, messages, gomock.Any(), gomock.Any()).
			Return(nil, nil, expectedErr).
			Times(1)

		err := svc.Evolve(ctx, prof, messages)
		assert.ErrorIs(t, err, expectedErr)
	})
}
