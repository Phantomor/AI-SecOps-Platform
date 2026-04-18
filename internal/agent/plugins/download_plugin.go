// internal/agent/plugins/download_plugin.go
package plugins

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url" // 新增引入
	"os"
	"path/filepath"
	"time"

	"sentinel-agent-go/internal/agent"

	"github.com/cloudwego/eino/schema"
)

type ResourceDownloadTool struct{}

var _ agent.Tool = (*ResourceDownloadTool)(nil)

func (t *ResourceDownloadTool) Info() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "download_resource",
		Desc: "从指定的网络 URL 下载文件到本地临时目录。返回文件保存的绝对路径。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"url":       {Type: schema.String, Desc: "文件的网络下载地址", Required: true},
			"file_name": {Type: schema.String, Desc: "期望保存的文件名，包含后缀（例如 data.json, image.png）", Required: true},
		}),
	}
}

func (t *ResourceDownloadTool) Execute(args string) (string, error) {
	var params struct {
		URL      string `json:"url"`
		FileName string `json:"file_name"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("参数解析失败: %v", err)
	}

	// 🌟 核心修复 1：创建带有真实 User-Agent 的请求，防止被目标服务器拒绝
	req, err := http.NewRequest("GET", params.URL, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// 🌟 核心修复 2：配置代理和超时时间 (下载文件给 30 秒)
	proxyURL, _ := url.Parse("http://127.0.0.1:7890")
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 30 * time.Second,
	}

	// 1. 发起下载请求
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载请求失败 (可能是网络或代理问题): %v", err)
	}
	defer resp.Body.Close()

	// 🌟 核心修复 3：优雅处理错误状态码，把 404/403 等信息返回给大模型，让它知道换个链接
	if resp.StatusCode != 200 {
		return fmt.Sprintf("下载失败：目标 URL 返回了 HTTP 状态码 %d。请检查该文件链接是否有效或尝试寻找其他下载地址。", resp.StatusCode), nil
	}

	// 2. 准备本地保存路径 (放到操作系统的临时目录)
	saveDir := filepath.Join(os.TempDir(), "goai_downloads")
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %v", err)
	}

	// 拼接时间戳防止文件名冲突
	safeFileName := fmt.Sprintf("%d_%s", time.Now().Unix(), params.FileName)
	savePath := filepath.Join(saveDir, safeFileName)

	// 3. 写入文件
	out, err := os.Create(savePath)
	if err != nil {
		return "", fmt.Errorf("创建本地文件失败: %v", err)
	}
	defer out.Close()

	if _, err = io.Copy(out, resp.Body); err != nil {
		return "", fmt.Errorf("保存文件数据失败: %v", err)
	}

	return fmt.Sprintf("✅ 资源下载成功，已保存至本地路径: %s", savePath), nil
}

func init() {
	agent.GlobalRegistry.Register(&ResourceDownloadTool{})
}
