package chat_model

import (
	"context"
	"fmt"

	"vine-agent/domain/chat"
	"vine-agent/domain/message"
	"vine-agent/domain/tool"
	"vine-agent/infra/client/deepseek"
)

// deepSeekAdapter 适配器，使得 deepseek.Client 实现 chat.ChatModel 接口
type deepSeekAdapter struct {
	client *deepseek.Client
}

// NewDeepSeekAdapter 构造一个新的 DeepSeek 适配器
func NewDeepSeekAdapter(client *deepseek.Client) chat.ChatModel {
	return &deepSeekAdapter{client: client}
}

// buildRequest 构建 deepseek.ChatCompletionRequest
func buildRequest(messages []message.Message, opts ...chat.OptionFunc) deepseek.ChatCompletionRequest {
	opt := &chat.Option{}
	for _, fn := range opts {
		fn(opt)
	}

	modelName := deepseek.ModelV4Flash
	if opt.Model != "" {
		modelName = opt.Model
	}

	req := deepseek.ChatCompletionRequest{
		Model:       modelName,
		Messages:    toDeepSeekMessages(messages),
		Temperature: opt.Temperature,
		MaxTokens:   opt.MaxTokens,
	}

	if opt.Tools != nil {
		req.Tools = convertTools(opt.Tools)
	}
	if opt.ToolChoice != nil {
		req.ToolChoice = opt.ToolChoice
	}

	return req
}

// Generate 接口实现
func (a *deepSeekAdapter) Generate(ctx context.Context, messages []message.Message, opts ...chat.OptionFunc) (*message.Message, error) {
	req := buildRequest(messages, opts...)
	resp, err := a.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty choices in response")
	}

	msg := fromDeepSeekMessage(resp.Choices[0].Message)
	return &msg, nil
}

// Stream 接口实现
func (a *deepSeekAdapter) Stream(ctx context.Context, messages []message.Message, opts ...chat.OptionFunc) (chat.StreamReader, error) {
	req := buildRequest(messages, opts...)
	stream, err := a.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, err
	}

	return &deepseekStreamReaderAdapter{
		stream: stream,
	}, nil
}

// deepseekStreamReaderAdapter 流式桥接适配器
type deepseekStreamReaderAdapter struct {
	stream *deepseek.ChatCompletionStream
}

func (r *deepseekStreamReaderAdapter) Recv() (*message.Message, error) {
	chunk, err := r.stream.Recv()
	if err != nil {
		return nil, err
	}
	if len(chunk.Choices) == 0 {
		return &message.Message{}, nil
	}
	delta := chunk.Choices[0].Delta

	var toolCalls []message.ToolCall
	if len(delta.ToolCalls) > 0 {
		toolCalls = make([]message.ToolCall, len(delta.ToolCalls))
		for i, tc := range delta.ToolCalls {
			toolCalls[i] = message.ToolCall{
				Index: tc.Index,
				ID:    tc.ID,
				Type:  tc.Type,
				Function: message.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}

	return &message.Message{
		Role:             message.Role(delta.Role),
		Content:          delta.Content,
		ReasoningContent: delta.ReasoningContent,
		ToolCalls:        toolCalls,
	}, nil
}

func (r *deepseekStreamReaderAdapter) Close() error {
	return r.stream.Close()
}

// 转换 DTO 工具函数

func toDeepSeekMessages(msgs []message.Message) []deepseek.Message {
	res := make([]deepseek.Message, len(msgs))
	for i, m := range msgs {
		res[i] = deepseek.Message{
			Role:             string(m.Role),
			Content:          m.Content,
			ReasoningContent: m.ReasoningContent,
			ToolCallID:       m.ToolCallID,
		}
		if len(m.ToolCalls) > 0 {
			res[i].ToolCalls = toDeepSeekToolCalls(m.ToolCalls)
		}
	}
	return res
}

func toDeepSeekToolCalls(tcs []message.ToolCall) []deepseek.ToolCall {
	res := make([]deepseek.ToolCall, len(tcs))
	for i, tc := range tcs {
		res[i] = deepseek.ToolCall{
			Index: i,
			ID:    tc.ID,
			Type:  tc.Type,
			Function: deepseek.Function{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}
	return res
}

func fromDeepSeekMessage(m deepseek.Message) message.Message {
	res := message.Message{
		Role:             message.Role(m.Role),
		Content:          m.Content,
		ReasoningContent: m.ReasoningContent,
		ToolCallID:       m.ToolCallID,
	}
	if len(m.ToolCalls) > 0 {
		res.ToolCalls = fromDeepSeekToolCalls(m.ToolCalls)
	}
	return res
}

func fromDeepSeekToolCalls(tcs []deepseek.ToolCall) []message.ToolCall {
	res := make([]message.ToolCall, len(tcs))
	for i, tc := range tcs {
		res[i] = message.ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: message.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}
	return res
}

func convertTools(tools []tool.Tool) []deepseek.Tool {
	if tools == nil {
		return nil
	}

	var result []deepseek.Tool
	for _, t := range tools {
		if t == nil {
			continue
		}
		info := t.Info()
		result = append(result, deepseek.Tool{
			Type: "function",
			Function: deepseek.FunctionDefinition{
				Name:        info.Name,
				Description: info.Description,
				Parameters:  info.Parameters,
			},
		})
	}
	return result
}
