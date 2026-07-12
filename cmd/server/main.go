package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"vine-agent/cmd/api"
	"vine-agent/cmd/bootstrap"
	"vine-agent/cmd/scheduler"
	"vine-agent/cmd/scheduler/job"

	"vine-agent/domain/chat"
	"vine-agent/domain/chat/chat_model"
	"vine-agent/infra/client/deepseek"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetPrefix("[vine-agent] ")
	log.SetOutput(os.Stdout)

	// 依赖注入
	chatModel := getChatModel()
	repoContainer := bootstrap.GetRepoContainer()
	domainContainer := bootstrap.GetDomainContainer(repoContainer, chatModel)
	appContainer := bootstrap.GetAppContainer(domainContainer, chatModel)

	// 定时任务
	cronScheduler := scheduler.NewCronScheduler()
	cronScheduler.Register("*/1 * * * *", job.NewEvolutionJob(domainContainer, appContainer).Run)
	cronScheduler.Start()
	defer cronScheduler.Stop()

	// http-api
	handler := api.NewAPIHandler(domainContainer, appContainer)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// 带请求日志的中间件
	loggingMux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("--> %s %s", r.Method, r.URL.Path)
		mux.ServeHTTP(w, r)
		log.Printf("<-- %s %s (elapsed %v)", r.Method, r.URL.Path, time.Since(start))
	})

	port := bootstrap.GetConfig().Server.Port
	if port != "" && !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	if port == "" {
		port = ":8080"
	}
	log.Printf("HTTP 接口服务成功启动，监听端口 %s ...\n", port)
	if err := http.ListenAndServe(port, loggingMux); err != nil {
		log.Fatalf("HTTP 服务运行异常退出: %v", err)
	}
}

func getChatModel() chat.ChatModel {
	var chatModel chat.ChatModel
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	dsClient := deepseek.NewClient(apiKey)
	chatModel = chat_model.NewDeepSeekAdapter(dsClient)
	return chatModel
}
