package message

// Role 消息发起者角色枚举
type Role string

// 定义角色枚举常量
const (
	RoleSystem      Role = "system"
	RoleUser        Role = "user"
	RoleAssistant   Role = "assistant"
	RoleTool        Role = "tool"
	RoleInterrupted Role = "interrupted"
)

// Message 代表对话领域模型中的一条核心消息实体
type Message struct {
	Role             Role       `json:"role"`
	Content          string     `json:"content"`
	ReasoningContent string     `json:"reasoning_content,omitempty"` // 仅在 DeepSeek R1 响应时携带
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`        // 助理消息中可能包含的工具调用
	ToolCallID       string     `json:"tool_call_id,omitempty"`      // 当角色为 tool 时，此字段必须填写，对应助理端发起的某次 ToolCall 的 ID，用于绑定匹配
}

// ToolCall 助理端发起的工具调用
type ToolCall struct {
	Index    int          `json:"index"` // 在流式增量返回过程中，标示当前片段属于第几个工具调用的局部切片下标索引（用以定位并拼接参数）
	ID       string       `json:"id"`    // 大模型生成的该次工具调用全局唯一标识 ID，在工具执行完毕回传结果消息时，必须携带该 ID 供模型绑定对应
	Type     string       `json:"type"`  // 固定为 "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall 函数调用详情
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // 参数的 JSON 字符串
}

// IsAssistant 返回该消息是否为助理消息
func (m *Message) IsAssistant() bool {
	return m.Role == RoleAssistant
}

// IsValidLLMRole 校验该消息角色是否为大模型支持的标准角色 (system, user, assistant, tool)
func (m *Message) IsValidLLMRole() bool {
	return m.Role == RoleSystem || m.Role == RoleUser || m.Role == RoleAssistant || m.Role == RoleTool
}

// IsToolCall 返回该消息是否为助理发起的工具调用消息
func (m *Message) IsToolCall() bool {
	return m.IsAssistant() && len(m.ToolCalls) > 0
}

// HasToolCalls 返回该消息是否包含工具调用
func (m *Message) HasToolCalls() bool {
	return len(m.ToolCalls) > 0
}

// Merge 将另一个 ToolCall 的增量合并到当前 ToolCall 实例中（用于流式拼接参数）
func (tc *ToolCall) Merge(other *ToolCall) {
	if other == nil {
		return
	}
	if other.ID != "" {
		tc.ID = other.ID
	}
	if other.Type != "" {
		tc.Type = other.Type
	}
	if other.Function.Name != "" {
		tc.Function.Name = other.Function.Name
	}
	if other.Function.Arguments != "" {
		tc.Function.Arguments += other.Function.Arguments
	}
}

// MergeStreamToolCall 将流式的增量 ToolCall 合并到已有的 ToolCall 切片中，并返回更新后的切片
func MergeStreamToolCall(tcs []ToolCall, delta *ToolCall) []ToolCall {
	if delta == nil {
		return tcs
	}
	idx := delta.Index
	for len(tcs) <= idx {
		tcs = append(tcs, ToolCall{})
	}
	tcs[idx].Merge(delta)
	return tcs
}

// NewToolMessage 创建一个工具执行结果的消息实体
func NewToolMessage(toolCallID, content string) Message {
	return Message{
		Role:       RoleTool,
		ToolCallID: toolCallID,
		Content:    content,
	}
}

// NewInterruptedMessage 创建一个中断标记的消息实体
func NewInterruptedMessage() Message {
	return Message{
		Role:    RoleInterrupted,
		Content: "对话已中断",
	}
}
