package deepseek

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// mockRoundTripper 自定义传输层以拦截 HTTP 请求
type mockRoundTripper struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

func TestClient_CreateChatCompletion(t *testing.T) {
	mockResponseJSON := `{
		"id": "chatcmpl-mock-123",
		"object": "chat.completion",
		"created": 1685493051,
		"model": "deepseek-v4-flash",
		"choices": [
			{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "这是 Mock 的非流式最终回答。",
					"reasoning_content": "思考：这是一个测试。"
				},
				"finish_reason": "stop"
			}
		],
		"usage": {
			"prompt_tokens": 10,
			"completion_tokens": 15,
			"total_tokens": 25
		}
	}`

	mockClient := &http.Client{
		Transport: &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				// 校验请求 Path 和方法
				if req.Method != "POST" || !strings.Contains(req.URL.Path, "/v1/chat/completions") {
					t.Errorf("unexpected request url or method: %s %s", req.Method, req.URL.String())
				}
				// 校验 Authorization Header
				auth := req.Header.Get("Authorization")
				if auth != "Bearer sk-test-key" {
					t.Errorf("unexpected authorization header: %s", auth)
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(mockResponseJSON)),
				}, nil
			},
		},
	}

	client := NewClient("sk-test-key", WithHTTPClient(mockClient))

	messages := []Message{
		{Role: "user", Content: "你好"},
	}

	resp, err := client.CreateChatCompletion(context.Background(), ChatCompletionRequest{
		Model:    ModelV4Flash,
		Messages: messages,
	})
	if err != nil {
		t.Fatalf("CreateChatCompletion failed: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("expected choices but got none")
	}

	choice := resp.Choices[0]
	if choice.Message.Role != "assistant" {
		t.Errorf("unexpected role: got %s, want assistant", choice.Message.Role)
	}
	if choice.Message.Content != "这是 Mock 的非流式最终回答。" {
		t.Errorf("unexpected content: got %s", choice.Message.Content)
	}
	if choice.Message.ReasoningContent != "思考：这是一个测试。" {
		t.Errorf("unexpected reasoning_content: got %s", choice.Message.ReasoningContent)
	}
}

func TestClient_CreateChatCompletionStream(t *testing.T) {
	// 构建 SSE 数据流
	ssePayload := "data: {\"choices\": [{\"delta\": {\"role\": \"assistant\", \"reasoning_content\": \"思考中\"}}]}\n\n" +
		"data: {\"choices\": [{\"delta\": {\"content\": \"天空\"}}]}\n\n" +
		"data: {\"choices\": [{\"delta\": {\"content\": \"是蓝色的\"}}]}\n\n" +
		"data: [DONE]\n\n"

	mockClient := &http.Client{
		Transport: &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type": []string{"text/event-stream"},
					},
					Body: io.NopCloser(strings.NewReader(ssePayload)),
				}, nil
			},
		},
	}

	client := NewClient("sk-test-key", WithHTTPClient(mockClient))

	messages := []Message{
		{Role: "user", Content: "为什么天空是蓝色的？"},
	}

	stream, err := client.CreateChatCompletionStream(context.Background(), ChatCompletionRequest{
		Model:    ModelV4Pro,
		Messages: messages,
	})
	if err != nil {
		t.Fatalf("CreateChatCompletionStream failed: %v", err)
	}
	defer stream.Close()

	var reasoningBuilder strings.Builder
	var contentBuilder strings.Builder

	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("recv error: %v", err)
		}

		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			if delta.ReasoningContent != "" {
				reasoningBuilder.WriteString(delta.ReasoningContent)
			}
			if delta.Content != "" {
				contentBuilder.WriteString(delta.Content)
			}
		}
	}

	if reasoningBuilder.String() != "思考中" {
		t.Errorf("unexpected reasoning output: got %s", reasoningBuilder.String())
	}
	if contentBuilder.String() != "天空是蓝色的" {
		t.Errorf("unexpected content output: got %s", contentBuilder.String())
	}
}

func TestClient_APIError(t *testing.T) {
	mockErrorJSON := `{
		"error": {
			"message": "API key 格式错误",
			"type": "invalid_request_error",
			"code": "invalid_api_key"
		}
	}`

	mockClient := &http.Client{
		Transport: &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(mockErrorJSON)),
				}, nil
			},
		},
	}

	client := NewClient("invalid-key", WithHTTPClient(mockClient))

	messages := []Message{
		{Role: "user", Content: "测试"},
	}

	_, err := client.CreateChatCompletion(context.Background(), ChatCompletionRequest{
		Model:    ModelV4Flash,
		Messages: messages,
	})
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	// 验证错误消息是否被结构化解析
	if !strings.Contains(err.Error(), "http error") ||
		!strings.Contains(err.Error(), "invalid_api_key") ||
		!strings.Contains(err.Error(), "API key 格式错误") {
		t.Errorf("unexpected error format: %v", err)
	}
}
