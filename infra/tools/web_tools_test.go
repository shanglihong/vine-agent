package tools_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"vine-agent/infra/tools"

	"github.com/stretchr/testify/assert"
)

// mockRoundTripper 模拟 HTTP 传输层，用于单元测试
type mockRoundTripper struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

func TestWebSearchTool_Execute(t *testing.T) {
	t.Run("成功从 DuckDuckGo 页面解析出搜索结果", func(t *testing.T) {
		mockHTML := `
		<html>
		<body>
			<div class="result results_links results_links_deep web-result">
				<div class="links_main links_deep result__body">
					<h2 class="result__title">
						<a class="result__a" href="https://go.dev/blog/go1.25">Go 1.25 is out</a>
					</h2>
					<div class="result__snippet">Go 1.25 is officially released with exciting features.</div>
				</div>
			</div>
			<div class="result results_links results_links_deep web-result">
				<div class="links_main links_deep result__body">
					<h2 class="result__title">
						<a class="result__a" href="https://github.com/golang/go">Go GitHub Repository</a>
					</h2>
					<div class="result__snippet">The Go programming language repository.</div>
				</div>
			</div>
		</body>
		</html>
		`

		toolInstance := tools.NewWebSearchTool()
		searchTool, ok := toolInstance.(*tools.WebSearchTool)
		assert.True(t, ok)

		mockClient := &http.Client{
			Transport: &mockRoundTripper{
				roundTripFunc: func(req *http.Request) (*http.Response, error) {
					assert.Contains(t, req.URL.String(), "duckduckgo.com")
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(mockHTML)),
						Header:     make(http.Header),
					}, nil
				},
			},
		}
		searchTool.SetClient(mockClient)

		// 执行
		res, err := searchTool.Execute(context.Background(), `{"query": "Go 1.25"}`)
		assert.NoError(t, err)

		// 校验包含解析出的结果
		assert.Contains(t, res, "Go 1.25 is out")
		assert.Contains(t, res, "https://go.dev/blog/go1.25")
		assert.Contains(t, res, "Go 1.25 is officially released")
		assert.Contains(t, res, "Go GitHub Repository")
	})

	t.Run("搜索请求发生网络错误时，直接提示失败", func(t *testing.T) {
		toolInstance := tools.NewWebSearchTool()
		searchTool, ok := toolInstance.(*tools.WebSearchTool)
		assert.True(t, ok)

		mockClient := &http.Client{
			Transport: &mockRoundTripper{
				roundTripFunc: func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("network timeout")
				},
			},
		}
		searchTool.SetClient(mockClient)

		res, err := searchTool.Execute(context.Background(), `{"query": "golang features"}`)
		assert.Error(t, err)
		assert.Equal(t, "", res)
		assert.Contains(t, err.Error(), "执行网络检索失败")
		assert.Contains(t, err.Error(), "network timeout")
	})

	t.Run("入参校验失败", func(t *testing.T) {
		toolInstance := tools.NewWebSearchTool()
		_, err := toolInstance.Execute(context.Background(), `{"query": ""}`)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "搜索查询词不能为空")

		_, err = toolInstance.Execute(context.Background(), `{invalid-json}`)
		assert.Error(t, err)
	})
}

func TestWebCrawlTool_Execute(t *testing.T) {
	t.Run("成功爬取网页并过滤 HTML 提取纯文本", func(t *testing.T) {
		mockHTML := `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Ignore Head</title>
			<style>body { color: red; }</style>
		</head>
		<body>
			<nav>
				<a href="/home">Home</a>
			</nav>
			<article>
				<h1>Welcome to Go 1.25</h1>
				<p>This is a paragraph of text explaining Go 1.25 features.</p>
				<script>console.log("ignore script");</script>
				<div>Some extra text in div.</div>
			</article>
			<footer>
				<p>Copyright 2026</p>
			</footer>
		</body>
		</html>
		`

		toolInstance := tools.NewWebCrawlTool()
		crawlTool, ok := toolInstance.(*tools.WebCrawlTool)
		assert.True(t, ok)

		mockClient := &http.Client{
			Transport: &mockRoundTripper{
				roundTripFunc: func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, "https://example.com/go125", req.URL.String())
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(mockHTML)),
						Header:     make(http.Header),
					}, nil
				},
			},
		}
		crawlTool.SetClient(mockClient)

		// 执行
		res, err := crawlTool.Execute(context.Background(), `{"url": "https://example.com/go125"}`)
		assert.NoError(t, err)

		// 检查纯文本内容
		assert.Contains(t, res, "Welcome to Go 1.25")
		assert.Contains(t, res, "This is a paragraph of text explaining Go 1.25 features.")
		assert.Contains(t, res, "Some extra text in div.")

		// 检查过滤的内容（不应包含导航栏、页脚、脚本、样式）
		assert.NotContains(t, res, "Home")
		assert.NotContains(t, res, "Copyright")
		assert.NotContains(t, res, "console.log")
		assert.NotContains(t, res, "color: red")
	})

	t.Run("爬取请求发生网络错误时，直接提示失败", func(t *testing.T) {
		toolInstance := tools.NewWebCrawlTool()
		crawlTool, ok := toolInstance.(*tools.WebCrawlTool)
		assert.True(t, ok)

		mockClient := &http.Client{
			Transport: &mockRoundTripper{
				roundTripFunc: func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("connection refused")
				},
			},
		}
		crawlTool.SetClient(mockClient)

		res, err := crawlTool.Execute(context.Background(), `{"url": "https://go.dev/blog/go1.25"}`)
		assert.Error(t, err)
		assert.Equal(t, "", res)
		assert.Contains(t, err.Error(), "执行网页抓取失败")
		assert.Contains(t, err.Error(), "connection refused")
	})

	t.Run("网页内容过长被自动截断", func(t *testing.T) {
		// 生成超长网页内容
		var htmlBuilder strings.Builder
		htmlBuilder.WriteString("<html><body><p>")
		for i := 0; i < 9000; i++ {
			htmlBuilder.WriteString("a")
		}
		htmlBuilder.WriteString("</p></body></html>")

		toolInstance := tools.NewWebCrawlTool()
		crawlTool, ok := toolInstance.(*tools.WebCrawlTool)
		assert.True(t, ok)

		mockClient := &http.Client{
			Transport: &mockRoundTripper{
				roundTripFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(htmlBuilder.String())),
						Header:     make(http.Header),
					}, nil
				},
			},
		}
		crawlTool.SetClient(mockClient)

		res, err := crawlTool.Execute(context.Background(), `{"url": "https://example.com/long"}`)
		assert.NoError(t, err)

		// 验证是否包含截断标记，且长度受限
		assert.Contains(t, res, "已截断...")
		// 8000 + 截断说明长度
		assert.LessOrEqual(t, len(res), 8200)
	})
}
