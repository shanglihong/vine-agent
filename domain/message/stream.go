package message

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
	Content    string            `json:"content,omitempty"`      // 文本 delta（text_delta 或 reasoning_delta）
	ToolCall   *ToolCall         `json:"tool_call,omitempty"`    // 工具调用详情（tool_call）
	ToolResult *StreamToolResult `json:"tool_result,omitempty"`  // 工具执行结果（tool_result）
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
