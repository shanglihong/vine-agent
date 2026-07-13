package middlewares

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func ExtractBizID() gin.HandlerFunc {
	// 定义一个临时的内部结构体，利用 Gin 的多源绑定能力
	type sessionExtractor struct {
		SessionID string `form:"session_id" json:"session_id" uri:"session_id"`
	}

	return func(c *gin.Context) {
		var ext sessionExtractor

		// 1. 优先尝试从路径参数解析 (如 /api/sessions/:session_id)
		_ = c.ShouldBindUri(&ext)

		// 2. 如果路径里没有，再从 Query 或 JSON Body 中解析
		if ext.SessionID == "" {
			_ = c.ShouldBind(&ext)
		}

		// 3. 校验 session_id 是否存在
		if ext.SessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "msg": "缺少必要参数 session_id"})
			c.Abort() // 拦截，不执行后续业务
			return
		}

		// 4. 【核心点】：将 session_id 注入到 c.Request.Context() 中
		// 这样后续业务层通过 ctx.Value(SessionIDKey) 就能直接拿到，彻底脱离 gin.Context
		ctx := context.WithValue(c.Request.Context(), SessionIDKey, ext.SessionID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
