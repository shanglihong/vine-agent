package extractor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"vine-agent/domain/chat"
	"vine-agent/domain/memory/profile"
	"vine-agent/domain/message"
	"vine-agent/utils"
)

type llmExtractor struct {
	chatModel chat.ChatModel
}

// NewLLMExtractor 创建基于 ChatModel 的 Profile.Extractor 接口实现
func NewLLMExtractor(chatModel chat.ChatModel) profile.Extractor {
	return &llmExtractor{
		chatModel: chatModel,
	}
}

// promptData 传递给 text/template 的动态渲染数据结构
type promptData struct {
	ExistingPreferences []string
	ExistingFacts       []string
}

// extractedResult 对应大模型输出的 JSON 格式
type extractedResult struct {
	Preferences []string `json:"preferences"`
	Facts       []string `json:"facts"`
}

// Extract 实现 profile.Extractor 接口
func (e *llmExtractor) Extract(ctx context.Context, messages []message.Message, existingPrefs []string, existingFacts []string) ([]string, []string, error) {
	// 1. 拼装模板数据
	data := promptData{
		ExistingPreferences: existingPrefs,
		ExistingFacts:       existingFacts,
	}

	// 2. 渲染 System Prompt
	systemPrompt, err := utils.RenderPromptFile("infra/extractor/prompts/memory_extractor.md", data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to render prompt template: %w", err)
	}

	systemMsg := message.Message{
		Role:    message.RoleSystem,
		Content: systemPrompt,
	}

	// 3. 构建发送给大模型的完整消息列表
	callMsgs := make([]message.Message, 0, len(messages)+1)
	callMsgs = append(callMsgs, systemMsg)
	callMsgs = append(callMsgs, messages...)

	// 4. 调用大模型非流式接口
	respMsg, err := e.chatModel.Generate(ctx, callMsgs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to call chat model for extraction: %w", err)
	}

	// 5. 剥离 Markdown 块外壳（如 ```json ... ```）以提高容错性
	content := strings.TrimSpace(respMsg.Content)
	if strings.HasPrefix(content, "```") {
		// 剥离开头的 ``` 或 ```json
		if idx := strings.Index(content, "\n"); idx != -1 {
			content = content[idx+1:]
		}
		// 剥离结尾的 ```
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}

	// 6. 反序列化 JSON 提取结果
	var rawResult extractedResult
	if err := json.Unmarshal([]byte(content), &rawResult); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal extracted JSON (content: %s): %w", content, err)
	}

	return rawResult.Preferences, rawResult.Facts, nil
}
