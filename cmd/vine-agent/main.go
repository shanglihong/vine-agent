package main

import (
	"context"
	"fmt"
	"log"
	"os"

	memory_app "vine-agent/app/memory"
	"vine-agent/config"
	"vine-agent/domain/chat/chat_model"
	"vine-agent/domain/memory/profile"
	"vine-agent/domain/memory/session"
	"vine-agent/domain/message"
	"vine-agent/infra/client/deepseek"
	"vine-agent/infra/extractor"
	"vine-agent/infra/persistence/file"
	"vine-agent/infra/persistence/sqlite"
)

func main() {
	logger := log.New(os.Stdout, "[vine-agent] ", log.LstdFlags)
	logger.Println("开始装配长期记忆演进系统...")
	ctx := context.Background()

	// 1. 加载配置
	cfg := config.DefaultConfig()

	// 2. 初始化持久化层仓储
	sessionStore, err := sqlite.NewSessionStore(cfg.SQLiteDBPath)
	if err != nil {
		logger.Fatalf("初始化 SQLite 仓储失败: %v", err)
	}
	profileRepo := file.NewFileProfileRepository(cfg.ProfileDir)

	// 3. 初始化大模型客户端与提取器
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		apiKey = "mock-api-key"
	}
	dsClient := deepseek.NewClient(apiKey)
	chatModel := chat_model.NewDeepSeekAdapter(dsClient)
	llmExtractor := extractor.NewLLMExtractor(chatModel)

	// 4. 初始化领域服务
	sessionSvc := session.NewSessionService(sessionStore)
	evolutionSvc := profile.NewEvolutionService(llmExtractor)

	// 5. 初始化应用层服务
	evolutionAppSvc := memory_app.NewEvolutionAppService(sessionSvc, profileRepo, evolutionSvc)

	logger.Println("系统各层组件依赖注入组装成功。")

	// ==================== 演示运行记忆进化流程 ====================
	userID := "user_test_999"
	sessionID := "sess_chat_001"

	// A. 模拟创建一个包含未演进消息的对话 Session
	sess := session.NewSession(sessionID, userID, nil)
	sess.Messages = append(sess.Messages, message.Message{
		Role:    message.RoleUser,
		Content: "你好！我最近正准备用 Go 语言和 DDD 架构来重构我的 Vine-Agent 智能体项目，感觉这样项目结构更加清晰！",
	})
	sess.Messages = append(sess.Messages, message.Message{
		Role:    message.RoleAssistant,
		Content: "你好！Go 语言和 DDD 的配合非常适合大型和复杂的 Agent 系统，它可以为你提供坚实的包解耦与业务内聚保护。我们可以一起讨论细节！",
	})
	sess.Messages = append(sess.Messages, message.Message{
		Role:    message.RoleUser,
		Content: "对了，我非常不喜欢吃海鲜，平时特别钟爱辣味火锅。",
	})

	// B. 保存该 Session 到 SQLite 数据库中
	logger.Println("【步骤1】保存新模拟的会话数据到 SQLite 数据库中...")
	if err := sessionSvc.Save(ctx, sess); err != nil {
		logger.Fatalf("保存会话失败: %v", err)
	}

	// C. 触发记忆进化应用层编排
	logger.Println("【步骤2】触发记忆演进编排 TriggerEvolution...")
	// 该方法会：
	// 1. 从 sessionSvc 抓取最新增量消息
	// 2. 读取当前用户已有的 profile 列表 (如存在)
	// 3. 渲染就近存放的 prompts/memory_extractor.md 并调用大模型进行提炼与覆盖
	// 4. 将提炼后的偏好及事实分别存入 data/profile/ 目录下对应的两个 md 文件中
	// 5. 更新 Session 中的已演进消息进度计数，并写回 SQLite
	err = evolutionAppSvc.TriggerEvolution(ctx, sessionID)
	if err != nil {
		logger.Printf("[演示运行错误] 记忆进化失败: %v (如果您的 API Key 无效，这符合预期)\n", err)
		logger.Println("提示：若要体验真实的大模型提炼进化，请在控制台 export DEEPSEEK_API_KEY=您的真实Key，并设置环境变量 APP_ENV=dev (启用项目本地data目录) 再次运行。")
	} else {
		logger.Println("【步骤3】记忆演进编排成功结束！")

		// D. 读取并展示进化后的长期记忆文件内容
		prof, err := profileRepo.GetByUserID(ctx, userID)
		if err != nil {
			logger.Fatalf("读取用户长期记忆失败: %v", err)
		}
		if prof != nil {
			fmt.Println("\n========================================")
			fmt.Printf("【用户长期记忆画像】UserID: %s\n", prof.UserID)
			fmt.Println("----------------------------------------")
			fmt.Println("★ 个人偏好 (Preferences):")
			for _, p := range prof.Preferences {
				fmt.Printf("- %s\n", p)
			}
			fmt.Println("\n★ 静态事实 (Facts):")
			for _, f := range prof.Facts {
				fmt.Printf("- %s\n", f)
			}
			fmt.Println("========================================")
		}
	}
}
