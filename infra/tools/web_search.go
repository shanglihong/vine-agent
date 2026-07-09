package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"vine-agent/domain/tool"

	"golang.org/x/net/html"
)

// SearchResult 搜索结果实体
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

const (
	// DefaultSearchURL 默认搜索引擎的查询 Base URL
	DefaultSearchURL = "https://html.duckduckgo.com/html/"
)

// WebSearchTool 网页检索工具，用于获取互联网最新资讯
type WebSearchTool struct {
	client    *http.Client
	searchURL string
}

// NewWebSearchTool 创建 WebSearchTool 实例
func NewWebSearchTool() tool.Tool {
	return &WebSearchTool{
		client: &http.Client{
			Timeout: 8 * time.Second,
		},
		searchURL: DefaultSearchURL,
	}
}

// SetSearchURL 允许自定义修改搜索引擎 URL 路径，便于配置管理与测试
func (t *WebSearchTool) SetSearchURL(url string) {
	t.searchURL = url
}

// Info 返回工具元数据定义
func (t *WebSearchTool) Info() tool.Definition {
	return tool.Definition{
		Name:        "web_search",
		Description: "在互联网上搜索与指定查询词相关的最新资讯、网页和文章链接。返回结果为 JSON 格式的网页数组，包含 title, url, snippet 字段。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "搜索查询词，例如：Go 1.25 新特性，DeepSeek 介绍",
				},
			},
			"required": []any{"query"},
		},
		RequiresConfirmation: false,
	}
}

// Execute 执行搜索逻辑
func (t *WebSearchTool) Execute(ctx context.Context, args string) (string, error) {
	// 1. 验证和解析参数
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	params.Query = strings.TrimSpace(params.Query)
	if params.Query == "" {
		return "", fmt.Errorf("搜索查询词不能为空")
	}

	// 2. 执行网络检索
	results, err := t.search(ctx, params.Query)
	if err != nil {
		return "", fmt.Errorf("执行网络检索失败: %w", err)
	}

	// 3. 格式化结果输出
	if len(results) == 0 {
		return "[]", nil
	}

	resBytes, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("序列化搜索结果失败: %w", err)
	}
	return string(resBytes), nil
}

// search 内部通过搜索引擎搜索
func (t *WebSearchTool) search(ctx context.Context, query string) ([]SearchResult, error) {
	searchURL := fmt.Sprintf("%s?q=%s", t.searchURL, url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	// 设置 User-Agent 模拟浏览器，防止被拦截
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("搜索服务返回异常状态码: %d", resp.StatusCode)
	}

	return parseDuckDuckGoHTML(resp.Body)
}

// parseDuckDuckGoHTML 解析 DuckDuckGo HTML 页面
func parseDuckDuckGoHTML(r io.Reader) ([]SearchResult, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "result") {
			// 在 result 节点子树中寻找标题、链接和摘要
			var title, href, snippet string

			var searchSubtree func(*html.Node)
			searchSubtree = func(sn *html.Node) {
				if sn.Type == html.ElementNode {
					if sn.Data == "a" && hasClass(sn, "result__a") {
						title = getInnerText(sn)
						href = getAttr(sn, "href")
					} else if sn.Data == "div" && hasClass(sn, "result__snippet") {
						snippet = getInnerText(sn)
					}
				}
				for child := sn.FirstChild; child != nil; child = child.NextSibling {
					searchSubtree(child)
				}
			}
			searchSubtree(n)

			// 清理并保存有效结果
			title = strings.TrimSpace(title)
			snippet = strings.TrimSpace(snippet)
			if title != "" && href != "" {
				// 处理相对路径的 href（虽然 DDG HTML 里的链接通常是经过处理的）
				if strings.HasPrefix(href, "//") {
					href = "https:" + href
				} else if strings.HasPrefix(href, "/uddg/") {
					// 解析 DuckDuckGo 重定向的真实链接
					if u, err := url.Parse(href); err == nil {
						if q := u.Query().Get("uddg"); q != "" {
							href = q
						} else {
							href = "https://duckduckgo.com" + href
						}
					}
				}
				results = append(results, SearchResult{
					Title:   title,
					URL:     href,
					Snippet: snippet,
				})
			}
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}
	traverse(doc)

	return results, nil
}

// 辅助 HTML 解析工具函数
func hasClass(n *html.Node, className string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			classes := strings.Fields(attr.Val)
			for _, c := range classes {
				if c == className {
					return true
				}
			}
		}
	}
	return false
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func getInnerText(n *html.Node) string {
	var sb strings.Builder
	var dfs func(*html.Node)
	dfs = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			dfs(child)
		}
	}
	dfs(n)
	return sb.String()
}
