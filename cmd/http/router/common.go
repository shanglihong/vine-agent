package router

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// Bind 通用的多源自动绑定器
func Bind(c *gin.Context, obj any) error {
	// 1. 如果路由有路径参数，自动解析 URI
	if len(c.Params) > 0 {
		m := make(map[string][]string)
		for _, param := range c.Params {
			m[param.Key] = []string{param.Value}
		}
		_ = binding.Uri.BindUri(m, obj)
	}

	// 2. 自动解析 URL 问号参数 (Query)
	_ = binding.Query.Bind(c.Request, obj)

	// 3. 自动解析 Body (JSON) -> 采用 With 缓存机制防止多级中间件 EOF
	if c.Request.Body != nil && c.Request.ContentLength > 0 {
		_ = c.ShouldBindBodyWith(obj, binding.JSON)
	}

	// 4. 【核心】所有数据装填完毕，最后统一触发唯一的全局结构体验证
	return binding.Validator.ValidateStruct(obj)
}
