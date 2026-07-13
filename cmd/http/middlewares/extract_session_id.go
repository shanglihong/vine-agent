package middlewares

import (
	"github.com/gin-gonic/gin"
	"vine-agent/app/agent"
	"vine-agent/cmd/http/dto"
)

func ExtractSessionID() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ext dto.SessIdReq
		_ = c.ShouldBindUri(&ext)
		_ = c.ShouldBind(&ext)

		ctx := c.Request.Context()
		if ext.ProjectID != "" {
			newCtx := agent.WithProjectID(ctx, ext.ProjectID)
			c.Request = c.Request.WithContext(newCtx)
		}
		if ext.SessionID != "" {
			newCtx := agent.WithSessionID(ctx, ext.SessionID)
			c.Request = c.Request.WithContext(newCtx)
		}
		if ext.UserID != "" {
			newCtx := agent.WithUserID(ctx, ext.UserID)
			c.Request = c.Request.WithContext(newCtx)
		}

		c.Next()
	}
}
