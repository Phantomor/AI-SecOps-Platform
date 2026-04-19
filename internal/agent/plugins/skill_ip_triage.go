package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	// ⚠️ 替换为你的真实路径
	"sentinel-agent-go/internal/agent"
	"sentinel-agent-go/internal/pkg/llm"
	"sentinel-agent-go/internal/repository"
)

type IPTriageSkill struct{}

// 编译期接口实现检查
var _ agent.Tool = (*IPTriageSkill)(nil)

// Info 定义工具的元数据
func (t *IPTriageSkill) Info() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "skill_ip_triage",
		Desc: `【SOP 级高级技能】用于对可疑/恶意的外部 IP 地址进行全自动综合研判。
当用户要求“分析 IP”、“研判某个外部地址”或“查一下这个 IP 的底细”时，必须调用此技能。
它将在底层自动执行：1. 威胁情报信誉查询 2. 内网资产关联 3. 内部封禁规范 (RAG) 检索。`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"target_ip": {Type: schema.String, Desc: "需要进行研判的目标 IP 地址", Required: true},
		}),
	}
}

// Execute 实际执行 SOP 工作流
func (t *IPTriageSkill) Execute(args string) (string, error) {
	var params struct {
		TargetIP string `json:"target_ip"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("参数解析失败: %v", err)
	}

	if params.TargetIP == "" {
		return "错误：target_ip 不能为空", nil
	}

	log.Printf("[AUDIT LOG] ⚡ 启动自动化研判技能流 (SOP)，目标 IP: %s", params.TargetIP)

	// ==========================================
	// 🚀 SOP 步骤 1：调用外部威胁情报 (模拟 VirusTotal/微步在线 API)
	// ==========================================
	time.Sleep(500 * time.Millisecond) // 模拟网络延迟
	tiReport := fmt.Sprintf(`[微步情报源] IP: %s
- 信誉评分: 危险 (Score: 85/100)
- 标签: [僵尸网络] [C2节点] [Cerberus变种]
- 最近活跃时间: 24小时内`, params.TargetIP)

	// ==========================================
	// 🚀 SOP 步骤 2：调用 ES 向量数据库查询企业内部阻断规范 (RAG 联动)
	// ==========================================
	kbDocs := "未获取到相关规范"
	// 我们把一个固定的查询意图转成向量，去搜 ES 里的规范
	queryVector, err := llm.GenerateEmbedding("发现高危恶意 IP 时的网络层阻断规范和处置流程")
	if err == nil {
		docs, searchErr := repository.SearchKnowledge(context.Background(), queryVector, 1) // 取 Top 1
		if searchErr == nil && len(docs) > 0 {
			kbDocs = docs[0]
		}
	}

	// ==========================================
	// 🚀 SOP 步骤 3：模拟内部 CMDB/WAF 关联查询
	// ==========================================
	wafLog := "过去 1 小时内，WAF 拦截了该 IP 对 /api/v1/payment 的 342 次高频扫描尝试。"

	// ==========================================
	// 🚀 SOP 步骤 4：组装并返回标准化研判报告
	// ==========================================
	var report strings.Builder
	report.WriteString(fmt.Sprintf("✅ 对 IP [%s] 的自动化研判已完成。以下是多维聚合结果：\n\n", params.TargetIP))
	report.WriteString("### 1. 外部威胁情报 (TI)\n" + tiReport + "\n\n")
	report.WriteString("### 2. 内网告警关联\n" + wafLog + "\n\n")
	report.WriteString("### 3. 企业合规行动建议 (知识库检索)\n" + kbDocs + "\n")

	return report.String(), nil
}

// init 自动注册到大模型的武器库中
func init() {
	agent.GlobalRegistry.Register(&IPTriageSkill{})
}
