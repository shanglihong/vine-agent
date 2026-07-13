package main

import (
	"log"
	"os"
	"vine-agent/cmd/bootstrap"
	"vine-agent/cmd/http"
	"vine-agent/cmd/scheduler"
	"vine-agent/domain/chat"
	"vine-agent/domain/chat/chat_model"
	"vine-agent/infra/client/deepseek"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetPrefix("[vine-agent] ")
	log.SetOutput(os.Stdout)

	// 依赖注入，在cmd层级可以通过GetXxxContainer快速获取已经注入的服务
	chatModel := getChatModel()
	repoContainer := bootstrap.InitRepoContainer()
	domainContainer := bootstrap.InitDomainContainer(repoContainer, chatModel)
	_ = bootstrap.InitAppContainer(domainContainer, chatModel)

	// 定时任务
	jobScheduler := scheduler.New().
		Register().
		Start()
	defer jobScheduler.Stop()

	// http
	http.New().Register().Run()
}

func getChatModel() chat.ChatModel {
	var chatModel chat.ChatModel
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	dsClient := deepseek.NewClient(apiKey)
	chatModel = chat_model.NewDeepSeekAdapter(dsClient)
	return chatModel
}
