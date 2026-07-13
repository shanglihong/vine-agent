package http

import (
	"context"
	"net/http"
	"vine-agent/cmd/http/dto"
	"vine-agent/cmd/http/middlewares"
	"vine-agent/cmd/http/router"
	"vine-agent/config"
	"vine-agent/utils"

	"github.com/gin-gonic/gin"
)

type Server struct {
	engine *gin.Engine
}

func New() *Server {
	engine := gin.Default()
	newServer := &Server{
		engine: engine,
	}
	return newServer
}

func (s *Server) Run() {
	cfg := config.LoadConfig()
	err := s.engine.Run(cfg.Server.Port)
	utils.Panic(err)
}

func (s *Server) Register() *Server {
	engine := s.engine
	engine.Use(middlewares.Cors())
	engine.Use(middlewares.ErrorHandler())
	engine.Use(middlewares.ErrorHandler())

	engine.GET("/api/sessions", handle(router.GetSessionHandler().ListSessions))
	engine.POST("/api/sessions", handle(router.GetSessionHandler().CreateSession))
	sessionGroup := engine.Group("/api/sessions/", middlewares.ExtractSessionID())
	sessionGroup.GET(":id/messages", handle(router.GetSessionHandler().GetSessionMessages))
	sessionGroup.DELETE(":id", handle(router.GetSessionHandler().DeleteSession))
	sessionGroup.POST(":id/rename", handle(router.GetSessionHandler().RenameSession))
	sessionGroup.POST(":id/chat", middlewares.SSEHeaders(), router.GetSessionChatHandler().Chat)      // 流式
	sessionGroup.POST(":id/confirm", middlewares.SSEHeaders(), router.GetSessionChatHandler().Resume) // 流式
	sessionGroup.POST(":id/cancel", router.GetSessionChatHandler().Cancel)

	engine.GET("/api/user", handle(router.GetUserHandler().GetUser))
	engine.GET("/api/users/:id/profile", handle(router.GetUserHandler().GetUserProfile))
	engine.POST("/api/users/:id/evolve", handle(router.GetUserHandler().Evolve))

	engine.POST("/api/projects", handle(router.GetProjectHandler().CreateProject))
	engine.GET("/api/projects", handle(router.GetProjectHandler().ListProjects))
	engine.GET("/api/projects/:id", handle(router.GetProjectHandler().GetProject))
	engine.PUT("/api/projects/:id", handle(router.GetProjectHandler().UpdateProject))
	engine.DELETE("/api/projects/:id", handle(router.GetProjectHandler().DeleteProject))
	engine.GET("/api/projects/:id/sessions", handle(router.GetProjectHandler().ListProjectSessions))

	return s
}

func handle[Req any, Resp any](bizFunc func(ctx context.Context, req Req) (Resp, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req Req
		if err := router.Bind(c, &req); err != nil {
			_ = c.Error(err)
			return
		}

		// 执行核心业务
		resp, err := bizFunc(c.Request.Context(), req)
		if err != nil {
			_ = c.Error(err)
			return
		}

		c.JSON(http.StatusOK, dto.NewSuccessResp(resp))
	}
}
