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
			resp := dto.NewErrorResp(err)
			if bindErr := c.ShouldBind(&resp); bindErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			}
			return
		}

	}
}
