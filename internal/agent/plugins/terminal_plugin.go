// internal/agent/plugins/terminal_plugin.go
package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"sentinel-agent-go/internal/agent"

	"github.com/cloudwego/eino/schema"
)

type TerminalOperationTool struct{}

var _ agent.Tool = (*TerminalOperationTool)(nil)

func (t *TerminalOperationTool) Info() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "execute_go_code",
		// 🌟 优化1：在描述中明确限制大模型只能使用标准库，防止它引用第三方包导致编译失败
		Desc: "在终端中编译并运行候选人提交的 Go 语言代码。返回标准输出(stdout)或错误信息(stderr)。注意：当前沙箱环境无 go.mod，仅允许使用 Go 原生标准库，禁止 import 任何第三方依赖！",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"code": {Type: schema.String, Desc: "完整的 Go 语言源代码文本", Required: true},
		}),
	}
}

func (t *TerminalOperationTool) Execute(args string) (string, error) {
	var params struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("解析参数失败: %v", err)
	}

	// 1. 创建临时执行目录和文件
	tmpDir, err := os.MkdirTemp("", "interviewer_code_*")
	if err != nil {
		return "", fmt.Errorf("创建沙盒目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir) // 运行完自动清理

	codePath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(codePath, []byte(params.Code), 0644); err != nil {
		return "", fmt.Errorf("写入代码文件失败: %v", err)
	}

	// 2. 准备执行命令 (go run main.go)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", "main.go") // 这里直接写 main.go 即可，因为下面切换了执行目录

	// 🌟 优化2：将工作目录切换到临时目录！防止大模型代码越权访问你主项目的文件
	cmd.Dir = tmpDir

	// 🌟 优化3：清理环境变量！只保留 PATH 和基本的 Go 环境变量，防止主程序的 API Key/数据库密码被恶意读取
	safeEnv := []string{
		"PATH=" + os.Getenv("PATH"),
		"GOPATH=" + os.Getenv("GOPATH"),
		"GOCACHE=" + os.Getenv("GOCACHE"),
		"HOME=" + os.Getenv("HOME"),
		"GO111MODULE=auto", // 确保在没有 go.mod 的情况下单文件能顺利编译
	}
	cmd.Env = safeEnv

	// 捕获输出
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 3. 执行代码
	err = cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return "⚠️ 代码运行超时 (超过 5 秒被强制终止)，可能存在死循环，请检查逻辑。", nil
	}

	if err != nil {
		// 🌟 优化4：有时候 panic 信息不仅在 stderr，也可能在 stdout。合并输出能给大模型提供更完整的报错上下文
		return fmt.Errorf("代码运行报错:\n%s\n部分标准输出:\n%s", stderr.String(), stdout.String()).Error(), nil
	}

	// 如果输出为空（比如大模型忘了写 fmt.Println），给一个友好的提示
	result := stdout.String()
	if result == "" {
		result = "(代码执行成功，但没有产生任何标准输出，请确认是否忘记添加 fmt.Println)"
	}

	// 4. 返回标准输出给大模型
	return fmt.Sprintf("代码运行成功，输出结果:\n%s", result), nil
}

func init() {
	agent.GlobalRegistry.Register(&TerminalOperationTool{})
}
