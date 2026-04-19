// internal/agent/plugins/pdf_plugin.go
package plugins

import (
	"encoding/json"
	"fmt"

	"sentinel-agent-go/internal/agent"
	"sentinel-agent-go/internal/tools"

	"github.com/cloudwego/eino/schema"
)

// PDFReportTool 实现了 agent.Tool 接口
type PDFReportTool struct{}

// 确保编译时检查接口实现
var _ agent.Tool = (*PDFReportTool)(nil)

func (t *PDFReportTool) Info() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "generate_pdf_report",
		Desc: "用于生成《ZeroTrust 终端安全研判报告》(PDF格式)。当用户要求“生成安全报告”、“导出审计PDF”或分析流程结束时，必须调用此工具。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"candidate_name": {Type: schema.String, Desc: "候选人的名字", Required: true},
			"comments":       {Type: schema.String, Desc: "面试评价、打分和总结", Required: true},
		}),
	}
}

func (t *PDFReportTool) Execute(args string) (string, error) {
	// 1. 解析大模型传过来的参数
	var params struct {
		CandidateName string `json:"candidate_name"`
		Comments      string `json:"comments"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("参数解析失败: %v", err)
	}

	// 2. 调用底层的业务逻辑
	filePath, err := tools.GenerateInterviewPDF(params.CandidateName, params.Comments)
	if err != nil {
		return "", err
	}

	// 3. 返回执行结果给大模型
	return fmt.Sprintf("PDF 生成成功，文件保存在：%s", filePath), nil
}

// 初始化时自动注册到全局 Registry
func init() {
	agent.GlobalRegistry.Register(&PDFReportTool{})
}
