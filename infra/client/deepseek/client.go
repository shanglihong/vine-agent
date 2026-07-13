package deepseek

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// 定义常用模型常量
const (
	ModelV4Flash = "deepseek-v4-flash"
	ModelV4Pro   = "deepseek-v4-pro"
)

// Message 独立 API 消息 DTO，与领域层解耦
type Message struct {
	Role             string     `json:"role"`
	Content          string     `json:"content"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
}

// ToolCall API 工具调用消息 DTO
type ToolCall struct {
	Index    int      `json:"index"`
	ID       string   `json:"id,omitempty"`
	Type     string   `json:"type,omitempty"`
	Function Function `json:"function,omitempty"`
}

// Function API 具体函数 DTO
type Function struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// ChatCompletionRequest 对话请求参数
type ChatCompletionRequest struct {
	Model           string          `json:"model"`
	Messages        []Message       `json:"messages"`
	Temperature     *float64        `json:"temperature,omitempty"`
	TopP            *float64        `json:"top_p,omitempty"` // 采样概率阈值
	MaxTokens       *int            `json:"max_tokens,omitempty"`
	Stream          bool            `json:"stream,omitempty"`
	StreamOptions   *StreamOptions  `json:"stream_options,omitempty"`
	Tools           []Tool          `json:"tools,omitempty"`            // 可用工具声明列表
	ToolChoice      any             `json:"tool_choice,omitempty"`      // "none", "auto", "required" 或特定的 ToolChoice 结构
	ResponseFormat  *ResponseFormat `json:"response_format,omitempty"`  // 控制输出格式，如指定 JSON Mode
	Stop            any             `json:"stop,omitempty"`             // 停止生成标识，支持 string 或 []string
	Thinking        *Thinking       `json:"thinking,omitempty"`         // 控制思考模式转换
	ReasoningEffort string          `json:"reasoning_effort,omitempty"` // 控制推理强度 ("high" 或 "max")
	LogProbs        *bool           `json:"logprobs,omitempty"`         // 是否返回 token 的对数概率
	TopLogProbs     *int            `json:"top_logprobs,omitempty"`     // 返回 top N 概率的 token
	UserID          string          `json:"user_id,omitempty"`          // 用户 ID，用于限速、KVCache 隔离
}

// ResponseFormat 响应格式控制
type ResponseFormat struct {
	Type string `json:"type"` // "text" 或 "json_object"
}

// Thinking 思考模式设置
type Thinking struct {
	Type string `json:"type"` // "enabled" 或 "disabled"
}

// StreamOptions 额外的流选项
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// Tool 工具声明
type Tool struct {
	Type     string             `json:"type"` // 目前仅支持 "function"
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition 函数声明定义
type FunctionDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters"`       // 输入参数定义，通常传入 JSON Schema
	Strict      bool   `json:"strict,omitempty"` // 是否开启 strict 模式，保证输出符合 schema 定义
}

// ChatCompletionResponse 非流式响应
type ChatCompletionResponse struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []Choice `json:"choices"`
	Usage             *Usage   `json:"usage,omitempty"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"` // 系统指纹
}

// Choice 单个候选回答
type Choice struct {
	Index        int       `json:"index"`
	Message      Message   `json:"message"`
	FinishReason string    `json:"finish_reason"`
	Logprobs     *Logprobs `json:"logprobs,omitempty"` // token 对数概率（若请求中开启）
}

// Usage 词数消耗详情
type Usage struct {
	PromptTokens            int                      `json:"prompt_tokens"`
	CompletionTokens        int                      `json:"completion_tokens"`
	TotalTokens             int                      `json:"total_tokens"`
	PromptTokensDetails     *PromptTokensDetails     `json:"prompt_tokens_details,omitempty"`     // 提示词详情统计（主要是 KVCache 命中情况）
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"` // 补全详情统计（主要是推理思维链 token数）
}

// PromptTokensDetails 提示词 Token 详情
type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"` // 命中的 KVCache token 数量
}

// CompletionTokensDetails 补全 Token 详情
type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"` // 推理产生的思维链 token 数量
}

// ChatCompletionStreamResponse 流式响应的分块结构体
type ChatCompletionStreamResponse struct {
	ID                string         `json:"id"`
	Object            string         `json:"object"`
	Created           int64          `json:"created"`
	Model             string         `json:"model"`
	Choices           []StreamChoice `json:"choices"`
	Usage             *Usage         `json:"usage,omitempty"`
	SystemFingerprint string         `json:"system_fingerprint,omitempty"` // 系统指纹
}

// StreamChoice 流式候选回答
type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        StreamDelta `json:"delta"`
	FinishReason string      `json:"finish_reason"`
	Logprobs     *Logprobs   `json:"logprobs,omitempty"` // token 对数概率（若请求中开启）
}

// StreamDelta 流式增量内容
type StreamDelta struct {
	Role             string           `json:"role,omitempty"`
	Content          string           `json:"content,omitempty"`
	ReasoningContent string           `json:"reasoning_content,omitempty"` // DeepSeek-R1 的推理思维链增量
	ToolCalls        []StreamToolCall `json:"tool_calls,omitempty"`        // 流式工具调用增量
}

// StreamToolCall 流式工具调用增量
type StreamToolCall struct {
	Index    int                `json:"index"`
	ID       string             `json:"id,omitempty"`
	Type     string             `json:"type,omitempty"`
	Function StreamFunctionCall `json:"function,omitempty"`
}

// StreamFunctionCall 流式函数调用增量
type StreamFunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"` // 增量参数片段，需要拼接
}

// Client DeepSeek 客户端
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// ClientOption 客户端可选配置项
type ClientOption func(*Client)

// WithBaseURL 设置自定义 API Base URL
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = strings.TrimSuffix(url, "/")
	}
}

// WithHTTPClient 设置自定义 http.Client
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// NewClient 初始化 DeepSeek 客户端
func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    "https://api.deepseek.com",
		httpClient: &http.Client{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// CreateChatCompletion 发送非流式对话请求
func (c *Client) CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	req.Stream = false
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp.Body, resp.StatusCode)
	}

	var completionResp ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completionResp); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &completionResp, nil
}

// ChatCompletionStream 流式对话读取器
type ChatCompletionStream struct {
	response *http.Response
	reader   *bufio.Reader
}

// Recv 读取下一个流响应分块
func (s *ChatCompletionStream) Recv() (*ChatCompletionStreamResponse, error) {
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 检查是否是 SSE 数据行
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			return nil, io.EOF
		}

		var chunk ChatCompletionStreamResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, fmt.Errorf("unmarshal chunk failed: %w", err)
		}

		return &chunk, nil
	}
}

// Close 关闭流并释放资源
func (s *ChatCompletionStream) Close() error {
	if s.response != nil && s.response.Body != nil {
		return s.response.Body.Close()
	}
	return nil
}

// CreateChatCompletionStream 发送流式对话请求
func (c *Client) CreateChatCompletionStream(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionStream, error) {
	req.Stream = true
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Cache-Control", "no-cache")
	httpReq.Header.Set("Connection", "keep-alive")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, parseErrorResponse(resp.Body, resp.StatusCode)
	}

	return &ChatCompletionStream{
		response: resp,
		reader:   bufio.NewReader(resp.Body),
	}, nil
}

// Logprobs 对数概率容器
type Logprobs struct {
	Content []Logprob `json:"content"`
}

// Logprob 详细对数概率信息
type Logprob struct {
	Token       string       `json:"token"`
	Logprob     float64      `json:"logprob"`
	Bytes       []int        `json:"bytes,omitempty"` // UTF-8 编码的字节整数数组
	TopLogprobs []TopLogprob `json:"top_logprobs"`    // 候选的候选 token
}

// TopLogprob 候选对数概率详情
type TopLogprob struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
	Bytes   []int   `json:"bytes,omitempty"`
}

// ErrorResponse API 调用报错时的 JSON 响应
type ErrorResponse struct {
	Error *APIError `json:"error"`
}

// APIError 详细错误明细
type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   any    `json:"param"`
	Code    string `json:"code"`
}

// parseErrorResponse 解析外部错误并返回带结构的友好错误信息
func parseErrorResponse(body io.Reader, statusCode int) error {
	bodyBytes, _ := io.ReadAll(body)
	var errResp ErrorResponse
	if err := json.Unmarshal(bodyBytes, &errResp); err == nil && errResp.Error != nil {
		return fmt.Errorf("http error (status %d, code: %s, type: %s): %s", statusCode, errResp.Error.Code, errResp.Error.Type, errResp.Error.Message)
	}
	return fmt.Errorf("http error (status %d): %s", statusCode, string(bodyBytes))
}
