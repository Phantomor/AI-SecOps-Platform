package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	// ⚠️ 请根据你的实际项目路径替换这里的包名
	"sentinel-agent-go/internal/agent"
	"sentinel-agent-go/internal/pkg/llm"
	"sentinel-agent-go/internal/repository"

	"github.com/cloudwego/eino/schema"
	// sentinel "sentinel-agent-go/internal/schema"
	// "your_project/internal/repository" // 稍后用于连接 Elasticsearch
)

type KnowledgeSearchTool struct{}

// 编译期接口实现检查
var _ agent.Tool = (*KnowledgeSearchTool)(nil)

// Info 定义工具的元数据与参数结构，供大模型识别
func (t *KnowledgeSearchTool) Info() *schema.ToolInfo {
	return &schema.ToolInfo{
		Name: "ext_knowledge_search",
		Desc: "用于检索企业内部《零信任安全规范》、架构标准和应急预案文档。当用户询问公司内部规定、安全合规要求或审查代码规范时，必须使用此工具获取上下文。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {Type: schema.String, Desc: "用于在向量数据库(ES)中进行语义检索的查询语句", Required: true},
		}),
	}
}

func (t *KnowledgeSearchTool) Execute(args string) (string, error) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("参数解析失败: %v", err)
	}

	if params.Query == "" {
		return "错误：query 参数不能为空", nil
	}

	log.Printf("[AUDIT LOG] 正在挂载知识库检索模块，目标向量库: Elasticsearch, 检索词: %s", params.Query)

	// ==========================================
	// 🚀 真实的 RAG 逻辑
	// ==========================================

	// 步骤 1: 调用你封装好的大模型 API，将用户的搜索词转为向量
	// ⚠️ 这里的 llm.GenerateEmbedding 需要你自己实现！
	// 如果你还没实现，请告诉我你用的是哪家的大模型。
	queryVector, err := llm.GenerateEmbedding(params.Query)
	if err != nil {
		return "", fmt.Errorf("生成 Embedding 失败: %v", err)
	}

	// 步骤 2: 去 Elasticsearch 检索最相似的 3 个片段
	docs, err := repository.SearchKnowledge(context.Background(), queryVector, 3)
	if err != nil {
		return "", fmt.Errorf("知识库检索失败: %v", err)
	}

	// 步骤 3: 拼装检索到的真实文档，返回给 Agent
	return fmt.Sprintf("检索成功。以下是相关的企业安全规范：\n\n%s", strings.Join(docs, "\n\n---\n\n")), nil
}

// init 自动将工具注册到全局总线
func init() {
	// 因为你在 WebSearchTool 里用的是 agent.GlobalRegistry.Register
	// 这里保持一致，这样服务启动时会自动加载它
	agent.GlobalRegistry.Register(&KnowledgeSearchTool{})
}
