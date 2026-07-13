package middlewares

import (
	"net/http"
	"vine-agent/cmd/http/dto"

	"github.com/gin-gonic/gin"
)

// ErrorHandler 自定义全局错误处理中间件
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // 执行后续的路由处理函数

		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			c.JSON(http.StatusBadRequest, dto.NewErrorResp(err))
			return
		}

	}
}
