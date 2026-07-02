package chat_model_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"vine-agent/domain/chat"
	"vine-agent/domain/chat/chat_model"
	"vine-agent/domain/message"
	"vine-agent/infra/client/deepseek"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRoundTripper struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

func TestDeepSeekAdapter_Generate(t *testing.T) {
	// 准备 Mock 响应的 JSON
	respJSON := `{
		"id": "chatcmpl-123",
		"object": "chat.completion",
		"created": 1677652288,
		"model": "deepseek-v4-flash",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "Hello! How can I assist you today?"
			},
			"finish_reason": "stop"
		}],
		"usage": {
			"prompt_tokens": 9,
			"completion_tokens": 12,
			"total_tokens": 21
		}
	}`

	var capturedReqBody []byte

	// 模拟 HTTP Transport
	mockTransport := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			// 校验请求头与地址
			assert.Equal(t, "POST", req.Method)
			assert.Equal(t, "https://api.deepseek.com/v1/chat/completions", req.URL.String())
			assert.Equal(t, "Bearer test-key", req.Header.Get("Authorization"))

			// 读取请求 Body 并缓存，以便后续测试校验
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			capturedReqBody = bodyBytes

			// 返回模拟响应
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(respJSON)),
				Header:     make(http.Header),
			}, nil
		},
	}

	httpClient := &http.Client{Transport: mockTransport}
	client := deepseek.NewClient("test-key", deepseek.WithHTTPClient(httpClient))
	adapter := chat_model.NewDeepSeekAdapter(client)

	ctx := context.Background()
	msgs := []message.Message{
		{Role: message.RoleUser, Content: "hi"},
	}

	// 运行 Generate
	resp, err := adapter.Generate(ctx, msgs, chat.WithModel("custom-model-name"))
	require.NoError(t, err)
	require.NotNil(t, resp)

	// 校验返回的消息结构
	assert.Equal(t, message.RoleAssistant, resp.Role)
	assert.Equal(t, "Hello! How can I assist you today?", resp.Content)

	// 校验捕获的请求体中是否包含指定的自定义 Model 参数
	assert.Contains(t, string(capturedReqBody), `"model":"custom-model-name"`)
	assert.Contains(t, string(capturedReqBody), `"content":"hi"`)
}

func TestDeepSeekAdapter_Stream(t *testing.T) {
	// SSE 分块格式的模拟返回数据
	sseData := "data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1677652288,\"model\":\"deepseek-v4-flash\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"Hello\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1677652288,\"model\":\"deepseek-v4-flash\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" world\"},\"finish_reason\":null}]}\n\n" +
		"data: [DONE]\n\n"

	// 模拟 HTTP Transport
	mockTransport := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "POST", req.Method)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(sseData)),
				Header:     make(http.Header),
			}, nil
		},
	}

	httpClient := &http.Client{Transport: mockTransport}
	client := deepseek.NewClient("test-key", deepseek.WithHTTPClient(httpClient))
	adapter := chat_model.NewDeepSeekAdapter(client)

	ctx := context.Background()
	msgs := []message.Message{
		{Role: message.RoleUser, Content: "stream test"},
	}

	// 运行 Stream
	stream, err := adapter.Stream(ctx, msgs)
	require.NoError(t, err)
	require.NotNil(t, stream)
	defer stream.Close()

	// 循环读取流数据并拼接
	var chunks []string
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if msg != nil {
			chunks = append(chunks, msg.Content)
		}
	}

	// 校验流拼接结果
	assert.Len(t, chunks, 2)
	assert.Equal(t, "Hello", chunks[0])
	assert.Equal(t, " world", chunks[1])
}
