// internal/agent/plugins/web_scraping_plugin.go
package plugins

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url" // 引入
	"strings"
	"time" // 引入

	"sentinel-agent-go/internal/agent"

	"github.com/PuerkitoBio/goquery"
	"github.com/cloudwego/eino/schema"
)

type WebScrapingTool struct{}

var _ agent.Tool = (*WebScrapingTool)(nil)

func (t *WebScrapingTool) Info() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "web_scrape",
		Desc: "用于联网搜索最新的网络安全资讯、CVE 漏洞情报或黑客攻击分析报告。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"url": {Type: schema.String, Desc: "需要抓取的网页完整 URL，例如 https://xxx.com", Required: true},
		}),
	}
}

func (t *WebScrapingTool) Execute(args string) (string, error) {
	var params struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("参数解析失败: %v", err)
	}

	req, err := http.NewRequest("GET", params.URL, nil)
	if err != nil {
		return "", err
	}
	// 使用更逼真的 User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// 1：为网页抓取也加上代理和超时时间
	proxyURL, _ := url.Parse("http://127.0.0.1:7890")
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 15 * time.Second, // 防止网页卡死导致整个 Agent 卡住
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("网页请求失败 (可能是网络或代理问题): %v", err)
	}
	defer resp.Body.Close()

	// 2：优雅处理 404 等错误，把结果告诉 AI，让 AI 自己决定下一步
	if resp.StatusCode != 200 {
		return fmt.Sprintf("抓取失败：目标网页返回了 HTTP 状态码 %d。该链接可能失效或存在反爬策略，请尝试搜索并抓取其他相关链接。", resp.StatusCode), nil
	}

	// 使用 goquery 解析 DOM 树
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("HTML 解析失败: %v", err)
	}

	var content strings.Builder
	title := doc.Find("title").Text()
	content.WriteString(fmt.Sprintf("网页标题: %s\n正文内容:\n", strings.TrimSpace(title)))

	// 提取所有的段落 <p> 标签文字
	doc.Find("p").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			content.WriteString(text + "\n")
		}
	})

	// 截断过长的文本，防止 Token 爆表
	finalText := content.String()
	if len(finalText) > 4000 {
		finalText = finalText[:4000] + "\n...(内容过长已截断)"
	}

	if len(finalText) < 50 {
		return "网页抓取成功，但未能提取到有效的正文文本。这可能是一个需要执行 JavaScript 的动态网页。", nil
	}

	return finalText, nil
}

func init() {
	agent.GlobalRegistry.Register(&WebScrapingTool{})
}
