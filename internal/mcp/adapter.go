// internal/mcp/adapter.go
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"sentinel-agent-go/internal/agent"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterMCPServer 初始化并独立启动 MCP 服务端
func RegisterMCPServer() {
	// 1. 创建一个新的 MCP Server 实例
	s := server.NewMCPServer(
		"sentinel-agent-go-tools", // Server 名称
		"1.0.0",                   // 版本号
	)

	// 2. 动态加载我们自己写的 GlobalRegistry 里的所有工具
	tools := agent.GlobalRegistry.GetAllToolInfos()

	for _, tInfo := range tools {
		toolName := tInfo.Name

		// 构建 MCP 标准工具定义
		mcpTool := mcp.NewTool(toolName,
			mcp.WithDescription(tInfo.Desc),
			mcp.WithString("params_json", mcp.Description("请将工具所需的所有参数组合成一个标准的 JSON 字符串传入此字段。")),
		)

		// 3. 将工具挂载到 MCP Server 上
		s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			log.Printf("🔗 [MCP Server] 收到外部调用请求，目标工具: %s", request.Params.Name)

			plugin, err := agent.GlobalRegistry.GetTool(request.Params.Name)
			if err != nil {
				return mcp.NewToolResultError("工具未找到: " + request.Params.Name), nil
			}

			// 🌟 修复 1：安全地处理 any 类型的类型断言
			var argsStr string
			if argsMap, ok := request.Params.Arguments.(map[string]interface{}); ok {
				// 尝试提取 params_json
				if val, exists := argsMap["params_json"].(string); exists && val != "" {
					argsStr = val
				} else {
					// 兜底：直接序列化整个 map
					argsBytes, _ := json.Marshal(argsMap)
					argsStr = string(argsBytes)
				}
			} else {
				// 如果连 map 都不是，直接暴力序列化
				argsBytes, _ := json.Marshal(request.Params.Arguments)
				argsStr = string(argsBytes)
			}

			// 🚀 执行我们自己写的底层插件
			result, err := plugin.Execute(argsStr)
			if err != nil {
				log.Printf("❌ [MCP Server] 工具执行失败: %v", err)
				return mcp.NewToolResultError(fmt.Sprintf("执行失败: %v", err)), nil
			}

			log.Printf("✅ [MCP Server] 工具执行成功，返回结果长度: %d", len(result))
			return mcp.NewToolResultText(result), nil
		})
	}

	// 🌟 修复 2 & 3：使用库自带的方法独立启动，避免与 Gin 框架的路由冲突
	// WithBaseURL 必须配置，这是客户端(Cursor)发 POST 消息的目标地址
	sseServer := server.NewSSEServer(s, server.WithBaseURL("http://localhost:8081"))

	// 放到后台 Goroutine 运行，绝不阻塞主线程的 Gin
	go func() {
		log.Printf("✅ MCP Server (SSE) 已独立启动，请在 Cursor 中连接: http://localhost:8081/sse")
		if err := sseServer.Start(":8081"); err != nil {
			log.Fatalf("MCP Server 启动失败: %v", err)
		}
	}()
}
