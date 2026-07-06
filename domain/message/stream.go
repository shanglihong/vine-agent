package message

import (
	"errors"
	"io"
)

// StreamMessageType 定义流式消息类型
type StreamMessageType string

const (
	StreamMessageTextDelta      StreamMessageType = "text_delta"      // 文本生成片段
	StreamMessageReasoningDelta StreamMessageType = "reasoning_delta" // 思考生成片段
	StreamMessageToolCall       StreamMessageType = "tool_call"       // 工具调用触发
	StreamMessageToolResult     StreamMessageType = "tool_result"     // 工具调用结果
)

// StreamMessage 代表流式输出的统一消息结构
type StreamMessage struct {
	Type       StreamMessageType `json:"type"`
	Content    string            `json:"content,omitempty"`     // 文本 delta（text_delta 或 reasoning_delta）
	ToolCall   *ToolCall         `json:"tool_call,omitempty"`   // 工具调用详情（tool_call）
	ToolResult *StreamToolResult `json:"tool_result,omitempty"` // 工具执行结果（tool_result）
}

// StreamToolResult 携带工具执行完毕的数据
type StreamToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Output     string `json:"output,omitempty"`
	Error      error  `json:"error,omitempty"`
}

// StreamMessageReader 流式消息读取器接口
type StreamMessageReader interface {
	// Recv 逐步读取流式消息，当流结束时返回 io.EOF
	Recv() (*StreamMessage, error)
	// Close 关闭消息流并释放相关资源
	Close() error
}

// IsTextDelta 判定当前流消息是否为文本生成片段
func (s *StreamMessage) IsTextDelta() bool {
	return s.Type == StreamMessageTextDelta
}

// IsReasoningDelta 判定当前流消息是否为推理生成片段
func (s *StreamMessage) IsReasoningDelta() bool {
	return s.Type == StreamMessageReasoningDelta
}

// IsDelta 判定当前流消息是否为生成片段（文本或推理）
func (s *StreamMessage) IsDelta() bool {
	return s.IsTextDelta() || s.IsReasoningDelta()
}

// ReadAndAssembleMessage 从 StreamMessageReader 中读取所有的流片段，累积拼接成一个完整的 Message 实体。
// 在读取过程中，如果传入了 callback 回调，会将每个流片段实时反馈。
func ReadAndAssembleMessage(stream StreamMessageReader, callback func(*StreamMessage)) (*Message, error) {
	var fullContent string
	var fullReasoning string
	var tempToolCalls []ToolCall

	for {
		msg, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		// 单个消息的回调（由外部决定如何处理，例如推送到事件总线）
		callback(msg)

		switch msg.Type {
		case StreamMessageTextDelta:
			fullContent += msg.Content
		case StreamMessageReasoningDelta:
			fullReasoning += msg.Content
		case StreamMessageToolCall:
			tempToolCalls = MergeStreamToolCall(tempToolCalls, msg.ToolCall)
		}
	}

	assistantMsg := Message{
		Role:             RoleAssistant,
		Content:          fullContent,
		ReasoningContent: fullReasoning,
	}
	if len(tempToolCalls) > 0 {
		assistantMsg.ToolCalls = tempToolCalls
	}

	return &assistantMsg, nil
}

// NewStreamToolResult 创建一个流式工具执行结果的实例指针
func NewStreamToolResult(toolCallID string, output string, err error) *StreamToolResult {
	res := &StreamToolResult{
		ToolCallID: toolCallID,
	}
	if err != nil {
		res.Error = err
	} else {
		res.Output = output
	}
	return res
}

// NewStreamMessageToolCall 快捷构造一个工具调用类型的流消息
func NewStreamMessageToolCall(tc *ToolCall) *StreamMessage {
	return &StreamMessage{
		Type:     StreamMessageToolCall,
		ToolCall: tc,
	}
}

// NewStreamMessageToolResult 快捷构造一个工具执行结果类型的流消息
func NewStreamMessageToolResult(toolCallID string, output string, err error) *StreamMessage {
	return &StreamMessage{
		Type:       StreamMessageToolResult,
		ToolResult: NewStreamToolResult(toolCallID, output, err),
	}
}
