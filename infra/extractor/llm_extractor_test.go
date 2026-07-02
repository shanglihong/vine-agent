package extractor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"vine-agent/domain/chat/mock"
	"vine-agent/domain/message"
	"vine-agent/infra/extractor"
)

func TestLLMExtractor_Extract(t *testing.T) {
	ctx := context.Background()

	t.Run("successful extraction with standard JSON response", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockChatModel := mock.NewMockChatModel(ctrl)
		llmExt := extractor.NewLLMExtractor(mockChatModel)

		existingPrefs := []string{"喜欢写 Go"}
		existingFacts := []string{"现居北京"}

		messages := []message.Message{
			{Role: message.RoleUser, Content: "我其实最讨厌写 Java，并且我还喜欢吃辣"},
		}

		mockResponse := `{
			"preferences": ["喜欢写 Go", "喜欢吃辣"],
			"facts": ["现居北京"]
		}`

		mockChatModel.EXPECT().
			Generate(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, msgs []message.Message, opts ...interface{}) (*message.Message, error) {
				assert.Len(t, msgs, 2)
				assert.Equal(t, message.RoleSystem, msgs[0].Role)
				assert.Equal(t, message.RoleUser, msgs[1].Role)

				assert.Contains(t, msgs[0].Content, "喜欢写 Go")
				assert.Contains(t, msgs[0].Content, "现居北京")

				return &message.Message{
					Role:    message.RoleAssistant,
					Content: mockResponse,
				}, nil
			}).
			Times(1)

		prefs, facts, err := llmExt.Extract(ctx, messages, existingPrefs, existingFacts)
		assert.NoError(t, err)
		assert.Equal(t, []string{"喜欢写 Go", "喜欢吃辣"}, prefs)
		assert.Equal(t, []string{"现居北京"}, facts)
	})

	t.Run("successful extraction with Markdown Code Block wrapper", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockChatModel := mock.NewMockChatModel(ctrl)
		llmExt := extractor.NewLLMExtractor(mockChatModel)

		mockResponseWithMarkdown := "```json\n{\n  \"preferences\": [\"吃甜食\"],\n  \"facts\": [\"软件工程师\"]\n}\n```"

		mockChatModel.EXPECT().
			Generate(ctx, gomock.Any()).
			Return(&message.Message{
				Role:    message.RoleAssistant,
				Content: mockResponseWithMarkdown,
			}, nil).
			Times(1)

		prefs, facts, err := llmExt.Extract(ctx, nil, nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, []string{"吃甜食"}, prefs)
		assert.Equal(t, []string{"软件工程师"}, facts)
	})

	t.Run("chat model returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockChatModel := mock.NewMockChatModel(ctrl)
		llmExt := extractor.NewLLMExtractor(mockChatModel)

		expectedErr := errors.New("request deepseek api timeout")
		mockChatModel.EXPECT().
			Generate(ctx, gomock.Any()).
			Return(nil, expectedErr).
			Times(1)

		prefs, facts, err := llmExt.Extract(ctx, nil, nil, nil)
		assert.ErrorIs(t, err, expectedErr)
		assert.Nil(t, prefs)
		assert.Nil(t, facts)
	})

	t.Run("json unmarshal failed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockChatModel := mock.NewMockChatModel(ctrl)
		llmExt := extractor.NewLLMExtractor(mockChatModel)

		invalidJSONResponse := `{"preferences": "not an array"}`

		mockChatModel.EXPECT().
			Generate(ctx, gomock.Any()).
			Return(&message.Message{
				Role:    message.RoleAssistant,
				Content: invalidJSONResponse,
			}, nil).
			Times(1)

		prefs, facts, err := llmExt.Extract(ctx, nil, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal extracted JSON")
		assert.Nil(t, prefs)
		assert.Nil(t, facts)
	})
}
