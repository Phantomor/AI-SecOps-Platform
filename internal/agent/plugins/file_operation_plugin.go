// internal/agent/plugins/file_operation_plugin.go
package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sentinel-agent-go/internal/agent"

	"github.com/cloudwego/eino/schema"
)

type FileOperationTool struct{}

var _ agent.Tool = (*FileOperationTool)(nil)

func (t *FileOperationTool) Info() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "file_operation",
		Desc: "用于读取或写入本地文件内容。支持 'read' 和 'write' 两种操作。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action":    {Type: schema.String, Desc: "操作类型: 'read' 或 'write'", Required: true},
			"file_path": {Type: schema.String, Desc: "文件的绝对路径或相对路径", Required: true},
			"content":   {Type: schema.String, Desc: "仅在 write 操作时需要，表示要写入的文件内容", Required: false},
		}),
	}
}

func (t *FileOperationTool) Execute(args string) (string, error) {
	var params struct {
		Action   string `json:"action"`
		FilePath string `json:"file_path"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("参数解析失败: %v", err)
	}

	// 🔒 安全防御：禁止访问系统级敏感目录
	forbiddenPaths := []string{"/etc", "/root", "/var", "C:\\Windows"}
	for _, fp := range forbiddenPaths {
		if strings.HasPrefix(filepath.ToSlash(params.FilePath), filepath.ToSlash(fp)) {
			return "", fmt.Errorf("安全拦截：禁止访问系统级目录")
		}
	}

	switch strings.ToLower(params.Action) {
	case "read":
		data, err := os.ReadFile(params.FilePath)
		if err != nil {
			return "", fmt.Errorf("读取文件失败: %v", err)
		}
		// 同样做截断，防止大文件撑爆 Token
		contentStr := string(data)
		if len(contentStr) > 5000 {
			contentStr = contentStr[:5000] + "\n...(文件过长已截断)"
		}
		return fmt.Sprintf("文件读取成功:\n%s", contentStr), nil

	case "write":
		// 如果目录不存在，自动创建
		dir := filepath.Dir(params.FilePath)
		os.MkdirAll(dir, 0755)

		err := os.WriteFile(params.FilePath, []byte(params.Content), 0644)
		if err != nil {
			return "", fmt.Errorf("写入文件失败: %v", err)
		}
		return fmt.Sprintf("✅ 成功将内容写入文件: %s", params.FilePath), nil

	default:
		return "", fmt.Errorf("不支持的 action 类型: %s，请使用 read 或 write", params.Action)
	}
}

func init() {
	agent.GlobalRegistry.Register(&FileOperationTool{})
}
