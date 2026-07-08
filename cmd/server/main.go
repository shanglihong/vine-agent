package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"vine-agent/app/agent"
	memory_app "vine-agent/app/memory"
	user_app "vine-agent/app/user"
	"vine-agent/config"
	"vine-agent/domain/chat"
	"vine-agent/domain/chat/chat_model"
	"vine-agent/domain/memory/profile"
	"vine-agent/domain/memory/session"
	"vine-agent/domain/message"
	"vine-agent/domain/tool"
	"vine-agent/domain/user"
	"vine-agent/infra/api"
	"vine-agent/infra/client/deepseek"
	infraevent "vine-agent/infra/event"
	"vine-agent/infra/extractor"
	"vine-agent/infra/persistence/file"
	"vine-agent/infra/persistence/sqlite"
	"vine-agent/infra/tools"
)

func main() {
	logger := log.New(os.Stdout, "[vine-server] ", log.LstdFlags)
	logger.Println("开始初始化 Vine-Agent 接口服务...")

	cfg := config.LoadConfig()
	// 1. 初始化持久化层
	sessionStore, err := sqlite.NewSessionStore(cfg.Storage.SQLiteDBPath)
	if err != nil {
		logger.Fatalf("初始化 SQLite 会话仓储失败: %v", err)
	}
	profileRepo := file.NewFileProfileRepository(cfg.Storage.ProfileDir)
	userStore, err := sqlite.NewUserStore(cfg.Storage.SQLiteDBPath)
	if err != nil {
		logger.Fatalf("初始化 SQLite 用户仓储失败: %v", err)
	}

	// 2. 初始化核心事件总线
	eventBus := infraevent.NewInMemoryEventBus(100, 2, nil)
	defer func() {
		_ = eventBus.Shutdown(context.Background())
	}()

	// 3. 根据环境变量加载大模型和提取器
	var chatModel chat.ChatModel
	var llmExtractor profile.Extractor

	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" || apiKey == "mock-api-key" {
		logger.Println("=========================================================================")
		logger.Println("【警告】未检测到有效的 DEEPSEEK_API_KEY 环境变量。")
		logger.Println("系统已自动启用「本地 MockChatModel 服务」进行流畅调试，支持工具调用与审批阻断模拟。")
		logger.Println("若需启用真实大模型，请运行: export DEEPSEEK_API_KEY=你的真实Key 后重新启动。")
		logger.Println("=========================================================================")

		chatModel = &mockChatModel{}
		llmExtractor = &mockExtractor{}
	} else {
		logger.Println("成功加载 DEEPSEEK_API_KEY，启用真实 DeepSeek 接入。")
		dsClient := deepseek.NewClient(apiKey)
		chatModel = chat_model.NewDeepSeekAdapter(dsClient)
		llmExtractor = extractor.NewLLMExtractor(chatModel)
	}

	// 4. 初始化领域服务
	sessionSvc := session.NewSessionService(sessionStore)
	evolutionSvc := profile.NewEvolutionService(llmExtractor)
	userDomainSvc := user.NewUserService(userStore)

	// 5. 初始化应用层服务
	agentSvc := agent.NewService(chatModel, sessionSvc)
	interactionSvc := agent.NewInteractionService(agentSvc, sessionSvc)
	evolutionAppSvc := memory_app.NewEvolutionAppService(sessionSvc, profileRepo, evolutionSvc)
	userAppSvc := user_app.NewUserAppService(userDomainSvc)

	// 6. 构造 API 控制器与注册路由
	appTools := []tool.Tool{
		tools.NewWeatherTool(),
		tools.NewCurrentCityTool(),
	}
	handler := api.NewAPIHandler(agentSvc, interactionSvc, sessionSvc, profileRepo, evolutionAppSvc, userAppSvc, appTools, logger)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// 带请求日志的中间件
	loggingMux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		logger.Printf("--> %s %s", r.Method, r.URL.Path)
		mux.ServeHTTP(w, r)
		logger.Printf("<-- %s %s (elapsed %v)", r.Method, r.URL.Path, time.Since(start))
	})

	port := cfg.Server.Port
	if port != "" && !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	if port == "" {
		port = ":8080"
	}
	logger.Printf("HTTP 接口服务成功启动，监听端口 %s ...\n", port)
	if err := http.ListenAndServe(port, loggingMux); err != nil {
		logger.Fatalf("HTTP 服务运行异常退出: %v", err)
	}
}

// =========================================================================
// 零依赖本地 Mock 智能体引擎 (当没有 API KEY 时自动回退启用，供前端无缝调试)
// =========================================================================

type mockChatModel struct{}

func (m *mockChatModel) Generate(ctx context.Context, messages []message.Message, opts ...chat.OptionFunc) (*message.Message, error) {
	return &message.Message{
		Role:    message.RoleAssistant,
		Content: "您好！这是本地 Mock 响应。请通过配置环境变量 DEEPSEEK_API_KEY 来启动真实大模型。",
	}, nil
}

func (m *mockChatModel) Stream(ctx context.Context, messages []message.Message, opts ...chat.OptionFunc) (message.StreamMessageReader, error) {
	if len(messages) == 0 {
		return nil, errors.New("empty messages")
	}

	lastMsg := messages[len(messages)-1]

	ch := make(chan *message.StreamMessage, 50)
	reader := &mockStreamReader{ch: ch}

	go func() {
		defer close(ch)

		// 1. 模拟思考流 (Reasoning Delta)
		ch <- &message.StreamMessage{Type: message.StreamMessageReasoningDelta, Content: "系统检测到输入... 开始推理意图\n"}
		time.Sleep(200 * time.Millisecond)
		ch <- &message.StreamMessage{Type: message.StreamMessageReasoningDelta, Content: "正在评估是否调用已声明的外部工具\n"}
		time.Sleep(250 * time.Millisecond)

		// 2. 如果是工具返回值返回的后续多轮对话，执行总结并回复
		if lastMsg.Role == message.RoleTool {
			replyText := fmt.Sprintf("我已经代表您执行了对应的工具。工具反馈回来的内容是: 「%s」。所有操作均已妥善处理，还有什么我可以帮您的？", lastMsg.Content)
			for _, r := range []rune(replyText) {
				ch <- &message.StreamMessage{
					Type:    message.StreamMessageTextDelta,
					Content: string(r),
				}
				time.Sleep(20 * time.Millisecond)
			}
			return
		}

		userText := lastMsg.Content

		// 3. 检查是否触发敏感需要人工确认的操作 (如 "删除数据"、"清空画像" 等)
		if strings.Contains(userText, "删除数据") || strings.Contains(userText, "清空画像") || strings.Contains(userText, "敏感操作") {
			ch <- &message.StreamMessage{Type: message.StreamMessageReasoningDelta, Content: "⚠ 警告: 触发敏感方法 [delete_user_data]，申请进行人工审批拦截！\n"}
			time.Sleep(300 * time.Millisecond)

			ch <- &message.StreamMessage{
				Type: message.StreamMessageToolCall,
				ToolCall: &message.ToolCall{
					ID:   "call_delete_mock_" + fmt.Sprintf("%d", time.Now().Unix()),
					Type: "function",
					Function: message.FunctionCall{
						Name:      "delete_user_data",
						Arguments: `{"user_id":"user_test_999"}`,
					},
				},
			}
			return
		}

		// 4. 检查是否触发一般操作 (如 "天气" 接口)
		if strings.Contains(userText, "天气") {
			ch <- &message.StreamMessage{Type: message.StreamMessageReasoningDelta, Content: "▶ 识别到需要获取天气指标。路由至外部工具 [get_weather]...\n"}
			time.Sleep(300 * time.Millisecond)

			ch <- &message.StreamMessage{
				Type: message.StreamMessageToolCall,
				ToolCall: &message.ToolCall{
					ID:   "call_weather_mock_" + fmt.Sprintf("%d", time.Now().Unix()),
					Type: "function",
					Function: message.FunctionCall{
						Name:      "get_weather",
						Arguments: `{"location":"杭州"}`,
					},
				},
			}
			return
		}

		// 5. 普通对话文本输出
		ch <- &message.StreamMessage{Type: message.StreamMessageReasoningDelta, Content: "▶ 无需调用工具，直接进行文本回复。\n"}
		time.Sleep(100 * time.Millisecond)

		welcomeText := "您好！我是您的智能助手（当前正运行在「本地调试 MockChatModel」模式下）。\n\n您可以对我进行以下专项交互测试：\n" +
			"1. 输入「**杭州的天气如何？**」以触发一般的外部工具调用查看回显。\n" +
			"2. 输入「**我想删除我的敏感数据**」以触发敏感工具拦截和「人工确认审批卡片」，体验中止与恢复流。\n" +
			"3. 对话结束后，在右侧面板点击「**Sync & Evolve**」按钮，系统会解析增量对话，并提取长期记忆 Facts 或者是 Preferences。"

		for _, r := range []rune(welcomeText) {
			ch <- &message.StreamMessage{
				Type:    message.StreamMessageTextDelta,
				Content: string(r),
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	return reader, nil
}

type mockStreamReader struct {
	ch chan *message.StreamMessage
}

func (r *mockStreamReader) Recv() (*message.StreamMessage, error) {
	msg, ok := <-r.ch
	if !ok {
		return nil, io.EOF
	}
	return msg, nil
}

func (r *mockStreamReader) Close() error {
	return nil
}

func (r *mockStreamReader) Interrupt() error {
	return r.Close()
}

// Mock 记忆提炼提取器 (无 API KEY 时的调试降级)
type mockExtractor struct{}

func (m *mockExtractor) Extract(ctx context.Context, messages []message.Message, existingPrefs []string, existingFacts []string) ([]string, []string, error) {
	newPrefs := append([]string{}, existingPrefs...)
	newFacts := append([]string{}, existingFacts...)

	// 从多轮消息中进行基础的关键词正则模拟提炼
	for _, msg := range messages {
		if msg.Role == message.RoleUser {
			text := msg.Content
			if strings.Contains(text, "喜欢") || strings.Contains(text, "爱吃") || strings.Contains(text, "平时特别") {
				newPrefs = append(newPrefs, fmt.Sprintf("对“%s”表达了明确的兴趣或个人喜好", text))
			} else if strings.Contains(text, "我是") || strings.Contains(text, "叫") {
				newFacts = append(newFacts, fmt.Sprintf("用户在对话中透露了身份属性: %s", text))
			} else if strings.Contains(text, "Go") || strings.Contains(text, "前端") || strings.Contains(text, "开发") {
				newFacts = append(newFacts, fmt.Sprintf("用户正在研发涉及技术栈的项目: %s", text))
			}
		}
	}

	// 保证每一次演化都有看得见的变化
	if len(newPrefs) == len(existingPrefs) && len(newFacts) == len(existingFacts) {
		newPrefs = append(newPrefs, fmt.Sprintf("测试偏好：%s 喜欢在前端页面上测试 Agent 的记忆反馈", time.Now().Format("15:04:05")))
		newFacts = append(newFacts, fmt.Sprintf("提取事实：用户在 %s 触发了画像演化流程", time.Now().Format("2006-01-02 15:04:05")))
	}

	return newPrefs, newFacts, nil
}
