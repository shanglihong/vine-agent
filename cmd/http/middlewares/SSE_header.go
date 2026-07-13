package middlewares

import "github.com/gin-gonic/gin"

// SSEHeaders 统一注入 Server-Sent Events 标准响应头
func SSEHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 必须在 c.Next() 之前注入，因为流式传输一旦开始，Header 就会立刻发出去
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("Transfer-Encoding", "chunked")
		c.Header("X-Accel-Buffering", "no") // 核心：防止 Nginx 缓存流式数据
		c.Next()
	}
}
