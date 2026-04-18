// internal/agent/plugins/web_search_plugin.go
package plugins

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"sentinel-agent-go/internal/agent"

	"github.com/PuerkitoBio/goquery"
	"github.com/cloudwego/eino/schema"
)

type WebSearchTool struct{}

var _ agent.Tool = (*WebSearchTool)(nil)

func (t *WebSearchTool) Info() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "web_search",
		Desc: "查询实时信息时调用此工具。它会返回搜索引擎的前几条结果，包含【标题】、【链接(URL)】和【摘要】。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {Type: schema.String, Desc: "要搜索的关键词", Required: true},
		}),
	}
}

func (t *WebSearchTool) Execute(args string) (string, error) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("参数解析失败: %v", err)
	}

	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(params.Query))
	req, _ := http.NewRequest("GET", searchURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// 使用代理，确保网络畅通
	proxyURL, _ := url.Parse("http://127.0.0.1:7890")
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 15 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("网络请求失败: %v", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("HTML 解析失败: %v", err)
	}

	var results strings.Builder
	results.WriteString(fmt.Sprintf("关于 '%s' 的搜索结果：\n\n", params.Query))

	count := 0
	// 🌟 核心修复：解析 DuckDuckGo 的整体 result 块，提取出 标题、摘要 和 URL
	doc.Find(".result").Each(func(i int, s *goquery.Selection) {
		if count < 4 { // 提取 4 条，给大模型更多选择
			title := strings.TrimSpace(s.Find(".result__title").Text())
			snippet := strings.TrimSpace(s.Find(".result__snippet").Text())

			// 获取链接
			rawURL, exists := s.Find("a.result__url").Attr("href")
			if !exists {
				rawURL, exists = s.Find(".result__title a").Attr("href")
			}

			if exists && snippet != "" {
				realURL := rawURL
				// 清洗 DuckDuckGo 的重定向伪装链接 (如 //duckduckgo.com/l/?uddg=https://...)
				if strings.Contains(rawURL, "uddg=") {
					u, parseErr := url.Parse(rawURL)
					if parseErr == nil {
						realURL = u.Query().Get("uddg")
					}
				}

				// 格式化输出给大模型
				results.WriteString(fmt.Sprintf("%d. 标题: %s\n   URL: %s\n   摘要: %s\n\n", count+1, title, realURL, snippet))
				count++
			}
		}
	})

	if results.Len() < 50 {
		return "未能提取到有效的搜索结果，请更换关键词重试。", nil
	}

	return results.String(), nil
}

func init() {
	agent.GlobalRegistry.Register(&WebSearchTool{})
}
