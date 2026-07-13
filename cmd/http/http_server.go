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

	engine.Group("/api/sessions/", middlewares.ExtractSessionID())
	engine.GET("/api/sessions", handle(router.GetSessionHandler().ListSessions))
	engine.POST("/api/sessions", handle(router.GetSessionHandler().CreateSession))
	engine.GET("/api/sessions/:id/messages", handle(router.GetSessionHandler().GetSessionMessages))
	engine.DELETE("/api/sessions/:id", handle(router.GetSessionHandler().DeleteSession))
	engine.POST("/api/sessions/:id/rename", handle(router.GetSessionHandler().RenameSession))
	engine.POST("/api/sessions/:id/chat", middlewares.SSEHeaders(), router.GetSessionChatHandler().Chat)      // 流式
	engine.POST("/api/sessions/:id/confirm", middlewares.SSEHeaders(), router.GetSessionChatHandler().Resume) // 流式
	engine.POST("/api/sessions/:id/cancel", router.GetSessionChatHandler().Cancel)

	engine.GET("/api/user", handle(router.GetUserHandler().GetUser))
	engine.GET("/api/users/:id/profile", handle(router.GetUserHandler().GetUserProfile))
	engine.POST("/api/users/:id/evolve", handle(router.GetUserHandler().Evolve))

	engine.POST("/api/projects", handle(router.GetProjectHandler().CreateProject))
	engine.GET("/api/projects", handle(router.GetProjectHandler().ListProjects))
	engine.GET("/api/projects/{id}", handle(router.GetProjectHandler().GetProject))
	engine.PUT("/api/projects/{id}", handle(router.GetProjectHandler().UpdateProject))
	engine.DELETE("/api/projects/{id}", handle(router.GetProjectHandler().DeleteProject))
	engine.GET("/api/projects/{id}/sessions", handle(router.GetProjectHandler().ListProjectSessions))

	return s
}

func handle[Req any, Resp any](bizFunc func(ctx context.Context, req Req) (Resp, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req Req
		if err := c.ShouldBindUri(&req); err != nil {
			_ = c.Error(err)
			return
		}
		if err := c.ShouldBind(&req); err != nil {
			_ = c.Error(err)
			return
		}

		resp, err := bizFunc(c.Request.Context(), req)

		// 4. 统一处理业务层返回的 error 或 成功数据
		if err != nil {
			_ = c.Error(err)
			return
		}

		c.JSON(http.StatusOK, dto.NewSuccessResp(resp))
	}
}
