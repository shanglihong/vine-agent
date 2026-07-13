package middlewares

import (
	"vine-agent/app/agent"
	"vine-agent/cmd/http/dto"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func ExtractSessionID() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ext dto.SessIdReq

		fill(c, &ext)

		// 注入上下文
		ctx := c.Request.Context()
		if ext.ProjectID != "" {
			ctx = agent.WithProjectID(ctx, ext.ProjectID)
		}
		if ext.SessionID != "" {
			ctx = agent.WithSessionID(ctx, ext.SessionID)
		}
		if ext.UserID != "" {
			ctx = agent.WithUserID(ctx, ext.UserID)
		}
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func fill(c *gin.Context, obj any) {
	if len(c.Params) > 0 {
		m := make(map[string][]string)
		for _, param := range c.Params {
			m[param.Key] = []string{param.Value}
		}
		_ = binding.Uri.BindUri(m, obj)
	}
	_ = binding.Query.Bind(c.Request, obj)
	if c.Request.Body != nil && c.Request.ContentLength > 0 {
		_ = c.ShouldBindBodyWith(obj, binding.JSON)
	}
}
