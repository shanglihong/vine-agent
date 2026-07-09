package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"vine-agent/domain/tool"

	"golang.org/x/net/html"
)

// WebCrawlTool 网页爬取工具，用于爬取指定 URL 的网页纯文本内容
type WebCrawlTool struct {
	client *http.Client
}

// NewWebCrawlTool 创建 WebCrawlTool 实例
func NewWebCrawlTool() tool.Tool {
	return &WebCrawlTool{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Info 返回工具元数据定义
func (t *WebCrawlTool) Info() tool.Definition {
	return tool.Definition{
		Name:        "fetch_webpage",
		Description: "获取指定 URL 网页的纯文本内容，过滤掉 HTML 标签、脚本和样式，主要用于阅读详细的文章、文档或资讯正文。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{
					"type":        "string",
					"description": "网页的完整 URL，例如：https://go.dev/blog/go1.25",
				},
			},
			"required": []any{"url"},
		},
		RequiresConfirmation: false,
	}
}

// Execute 执行网页爬取与纯文本提取逻辑
func (t *WebCrawlTool) Execute(ctx context.Context, args string) (string, error) {
	// 1. 验证和解析参数
	var params struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	params.URL = strings.TrimSpace(params.URL)
	if params.URL == "" {
		return "", fmt.Errorf("网页 URL 不能为空")
	}

	// 2. 执行网络请求抓取
	htmlContent, err := t.fetchHTML(ctx, params.URL)
	if err != nil {
		return "", fmt.Errorf("执行网页抓取失败: %w", err)
	}

	// 3. 解析并清理 HTML 提取纯文本
	plainText, err := extractPlainText(htmlContent)
	if err != nil {
		return "", fmt.Errorf("提取网页纯文本失败: %w", err)
	}

	// 4. 对文本做截断，防止 Token 溢出（最大 8000 字符）
	const maxTextLength = 8000
	runes := []rune(plainText)
	if len(runes) > maxTextLength {
		plainText = string(runes[:maxTextLength]) + "\n\n(内容过长，已截断...)"
	}

	return plainText, nil
}

// fetchHTML 发送 HTTP GET 请求抓取 HTML
func (t *WebCrawlTool) fetchHTML(ctx context.Context, targetURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return "", err
	}

	// 设置防爬头部
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("服务器返回状态码异常: %d", resp.StatusCode)
	}

	// 限制读取大小为 2MB，防止内存溢出
	limitedReader := io.LimitReader(resp.Body, 2*1024*1024)
	bytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// extractPlainText 过滤无用 HTML 元素并提取纯文本
func extractPlainText(htmlStr string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	// 块级元素定义，用于在这些元素之间插入换行以保持基本排版
	blockTags := map[string]bool{
		"p": true, "div": true, "h1": true, "h2": true, "h3": true, "h4": true,
		"h5": true, "h6": true, "li": true, "br": true, "tr": true, "section": true,
		"article": true, "aside": true,
	}

	// 过滤无用的渲染/交互脚本及非核心结构标签
	ignoreTags := map[string]bool{
		"script": true, "style": true, "noscript": true, "iframe": true,
		"svg": true, "canvas": true, "nav": true, "footer": true, "header": true,
		"head": true,
	}

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			tagName := strings.ToLower(n.Data)
			if ignoreTags[tagName] {
				return // 忽略整个子树
			}
			if blockTags[tagName] {
				sb.WriteString("\n")
			}
		} else if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				sb.WriteString(n.Data)
			}
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}

		if n.Type == html.ElementNode {
			tagName := strings.ToLower(n.Data)
			if blockTags[tagName] {
				sb.WriteString("\n")
			}
		}
	}
	traverse(doc)

	// 进一步格式化：合并多余的空行和连续的空白字符
	res := sb.String()
	// 合并 3 个及以上的换行为 2 个
	reNewlines := regexp.MustCompile(`\n{3,}`)
	res = reNewlines.ReplaceAllString(res, "\n\n")

	// 合并连续的水平空白字符
	reSpaces := regexp.MustCompile(`[ \t]+`)
	res = reSpaces.ReplaceAllString(res, " ")

	return strings.TrimSpace(res), nil
}
