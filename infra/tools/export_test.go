package tools

import "net/http"

// SetClient 允许在单元测试中注入自定义的 http.Client
func (t *WebSearchTool) SetClient(c *http.Client) {
	t.client = c
}

// SetClient 允许在单元测试中注入自定义的 http.Client
func (t *WebCrawlTool) SetClient(c *http.Client) {
	t.client = c
}
