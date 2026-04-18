// internal/mcp/client.go
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"sentinel-agent-go/internal/agent"

	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// ExternalMCPTool 包装器：将外部的 MCP 工具转化为我们自己系统的 agent.Tool
type ExternalMCPTool struct {
	mcpClient client.MCPClient
	mcpTool   mcp.Tool
	info      *schema.ToolInfo
}

// Info 返回给我们大模型的工具说明书
func (t *ExternalMCPTool) Info() *schema.ToolInfo {
	return t.info
}

// Execute 拦截大模型的调用，转发给外部的 MCP Server
func (t *ExternalMCPTool) Execute(args string) (string, error) {
	// 1. 大模型传过来的是包含了 args_json 的包装层
	var wrapper struct {
		ArgsJson string `json:"args_json"`
	}
	if err := json.Unmarshal([]byte(args), &wrapper); err != nil {
		return "", fmt.Errorf("参数解析失败: %v", err)
	}

	// 2. 解析出真正的参数字典
	var realParams map[string]interface{}
	if err := json.Unmarshal([]byte(wrapper.ArgsJson), &realParams); err != nil {
		// 兜底：如果大模型直接传了平铺的 JSON
		json.Unmarshal([]byte(args), &realParams)
	}

	// 3. 组装 MCP 标准请求并发送给外部 Server
	req := mcp.CallToolRequest{}
	req.Params.Name = t.mcpTool.Name
	req.Params.Arguments = realParams

	resp, err := t.mcpClient.CallTool(context.Background(), req)
	if err != nil {
		return "", fmt.Errorf("外部 MCP 工具调用崩溃: %v", err)
	}

	if resp.IsError {
		return fmt.Sprintf("⚠️ 外部工具执行报错: %v", resp.Content), nil
	}

	// 4. 将外部执行结果返回给大模型
	resultBytes, _ := json.Marshal(resp.Content)
	return string(resultBytes), nil
}

// StartMCPClient 启动客户端，连接外部 Server 并把工具挂载到本地注册表
func StartMCPClient() {
	ctx := context.Background()

	// 1. 通过 Stdio (标准输入输出) 连接官方的 SQLite MCP Server
	// 命令等价于：npx -y @modelcontextprotocol/server-sqlite /tmp/test.db

	mcpClient, err := client.NewStdioMCPClient(
		"/home/pvr1sc/.local/bin/mcp-server-sqlite", // ⚠️ 务必替换为你自己真实的绝对路径！
		[]string{"--db-path", "/tmp/test.db"},
	)
	if err != nil {
		log.Printf("❌ 启动外部 MCP Client 失败: %v", err)
		return
	}

	// 2. 初始化握手
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = "2024-11-05"
	initReq.Params.ClientInfo = mcp.Implementation{Name: "sentinel-agent-go", Version: "1.0.0"}

	_, err = mcpClient.Initialize(ctx, initReq)
	if err != nil {
		log.Printf("❌ MCP Client 握手失败: %v", err)
		return
	}

	// 3. 获取外部 Server 所有的工具！
	toolsResp, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Printf("❌ 获取外部工具列表失败: %v", err)
		return
	}

	// 4. 遍历并转化为本地插件，注册进 GlobalRegistry
	for _, externalTool := range toolsResp.Tools {
		schemaBytes, _ := json.MarshalIndent(externalTool.InputSchema, "", "  ")

		// 极其关键：通过 Prompt 让大模型理解外部工具的参数格式
		desc := fmt.Sprintf("%s\n【重要参数规范】请严格按照以下 JSON Schema 构造参数，并将构造好的 JSON 字符串放入 args_json 字段中:\n%s", externalTool.Description, string(schemaBytes))

		info := &schema.ToolInfo{
			Name: "ext_sqlite_" + externalTool.Name, // 加上前缀防止重名
			Desc: desc,
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"args_json": {Type: schema.String, Desc: "将需要的真实参数序列化为 JSON 字符串填入这里", Required: true},
			}),
		}

		wrapper := &ExternalMCPTool{
			mcpClient: mcpClient,
			mcpTool:   externalTool,
			info:      info,
		}

		// 动态打入你写好的全局注册表！
		agent.GlobalRegistry.Register(wrapper)
		log.Printf("🔗 成功吸收外部 MCP 工具: [%s]", info.Name)
	}
}
