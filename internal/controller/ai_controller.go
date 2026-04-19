// internal/controller/ai_controller.go
package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"sentinel-agent-go/internal/agent"
	"sentinel-agent-go/internal/config"
	"sentinel-agent-go/internal/dto"
	modeldb "sentinel-agent-go/internal/model"
	"sentinel-agent-go/internal/pkg"
	"sentinel-agent-go/internal/repository"
	"sentinel-agent-go/internal/worker"

	"github.com/cloudwego/eino-ext/components/embedding/openai" // 引入 embedding 包
	modelopenai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"

	// 引入 Eino 的基础模型包
	"github.com/cloudwego/eino/components/model"
)

// AIController 负责处理 AI 相关的 HTTP 请求
type AIController struct {
	chatModel *modelopenai.ChatModel
	embedder  *openai.Embedder // 向量化工具
}

// NewAIController 初始化 AI 控制器并构建模型客户端
func NewAIController() *AIController {
	// 从我们之前写好的配置中心读取 AI 配置
	aiConf := config.Global.AI.OpenAI

	// 1. 初始化对话模型
	chatModel, err := modelopenai.NewChatModel(context.Background(), &modelopenai.ChatModelConfig{
		APIKey:  aiConf.APIKey,
		BaseURL: aiConf.BaseURL,
		Model:   aiConf.Model,
	})

	if err != nil {
		panic("AI 对话模型初始化失败: " + err.Error())
	}

	// 2. 初始化向量化模型
	embedder, err := openai.NewEmbedder(context.Background(), &openai.EmbeddingConfig{
		APIKey:  aiConf.APIKey,
		BaseURL: aiConf.BaseURL,
		Model:   "text-embedding-v1", // 保持与刷数据脚本一致
	})
	if err != nil {
		panic("AI 向量模型初始化失败: " + err.Error())
	}

	return &AIController{
		chatModel: chatModel,
		embedder:  embedder,
	}
}

// RegisterRoutes 注册 AI 相关的路由
func (ac *AIController) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		// 保留老接口作为对比
		api.POST("/chat", ac.HandleChat)
		// 高大上的 SSE 流式接口
		api.POST("/chat/stream", ac.HandleStreamChat)
		api.POST("/interview/report", ac.GenerateReport)

		// 1. 合规审查专用路由
		api.POST("/secops/compliance", ac.HandleComplianceChat)

		// 2. 威胁狩猎专家专用路由
		api.POST("/secops/threat_hunter", ac.HandleThreatHunterChat)
	}
}

// 定义存入 Redis 的消息结构
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// UploadResume 异步处理简历上传
// @Summary 异步处理简历上传
// @Description 上传简历文件(PDF/Word等)至 MinIO，并异步投递文档解析任务到 Kafka 中转。
// @Tags 简历模块 (Resume)
// @Accept multipart/form-data
// @Produce json
// @Param resume formData file true "需要上传的简历文件"
// @Success 200 {object} map[string]interface{} "{"code": 0, "data": {"message": "简历上传成功...", "object_name": "resumes/..."}}"
// @Failure 400 {object} map[string]interface{} "{"code": 400, "message": "请上传简历文件"}"
// @Failure 500 {object} map[string]interface{} "{"code": 500, "message": "文件存储失败或任务投递失败"}"
// @Security ApiKeyAuth
// @Router /upload/resume [post]
func (ac *AIController) UploadResume(c *gin.Context) {
	// 假设我们在中间件里拿到了当前用户 ID (结合刚才阶段一的 JWT)
	userID, exists := c.Get("currentUserID")
	if !exists {
		userID = uint(1001) // mock
	}

	file, err := c.FormFile("resume")
	if err != nil {
		pkg.Error(c, 400, "请上传简历文件")
		return
	}

	fileContent, _ := file.Open()
	defer fileContent.Close()

	// 1. 生成唯一文件名存入 MinIO
	objectName := fmt.Sprintf("resumes/user_%v_%d_%s", userID, time.Now().Unix(), file.Filename)
	err = repository.UploadFile(c.Request.Context(), objectName, fileContent, file.Size, file.Header.Get("Content-Type"))
	if err != nil {
		pkg.Error(c, 500, "文件存储失败: "+err.Error())
		return
	}

	// 2. 构建消息体发送到 Kafka
	msgBody := worker.DocParseMessage{
		UserID:     userID.(uint),
		ObjectName: objectName,
		Status:     "pending",
	}
	payload, _ := json.Marshal(msgBody)

	err = repository.SendDocParseTask(c.Request.Context(), string(rune(userID.(uint))), payload)
	if err != nil {
		pkg.Error(c, 500, "任务投递失败")
		return
	}

	// 3. 立即响应前端，不让前端长时间等待
	pkg.Success(c, gin.H{
		"message":     "简历上传成功，AI 正在后台急速解析中...",
		"object_name": objectName,
	})
}

// SaveMessageToDB 封装一个辅助函数用于保存记录
func (ac *AIController) SaveMessageToDB(sessionID string, userID uint, role, content string) {
	logEntry := modeldb.ChatLog{
		SessionID: sessionID,
		UserID:    userID,
		Role:      role,
		Content:   content,
	}
	// 这里直接使用 GORM 写入
	if err := repository.DB.Create(&logEntry).Error; err != nil {
		fmt.Printf("❌ 消息持久化失败: %v\n", err)
	}
}

const ThreatHunterPrompt = `你现在的身份是「ZeroTrust Sentinel」系统中的核心大模型节点：ThreatHunter（威胁狩猎专家）。
你是一位精通网络安全、Android 恶意软件分析（TA-HCL 概念漂移）和数据库底层操作的资深安全架构师。

【核心能力与工作流】
1. 当遇到安全分析请求时，你必须优先考虑使用你挂载的 MCP 工具（如 ext_sqlite_read_query）去本地数据库提取恶意软件特征或网络日志。
2. 对于未知漏洞或安全动态，请使用 web_search 工具检索 CVE 库或最新安全播报。
3. 如果需要分析混淆代码或测试加密算法，请使用 execute_go_code 工具在沙箱中运行验证。

【输出规范】
1. 语气必须极其专业、冷峻、客观，严禁使用任何轻浮、口语化或拟人化的表达。
2. 在展示 IP 地址、Hash 值、恶意样本特征或数据库查询结果时，必须使用 Markdown 表格进行格式化。
3. 你的分析报告必须包含：[事件概述]、[数据取证/调用记录]、[威胁定性]、[处置建议] 四个模块。`

const ComplianceCopilotPrompt = `[SYSTEM BOOT] 你现在的身份是「ZeroTrust Sentinel」系统中的 DevSecOps 合规顾问。
你的职责是协助研发团队审查代码，解释企业零信任安全规范，并提供漏洞修复建议。

【核心能力与工作流】
1. 当用户询问内部安全标准、最佳实践或合规性要求时，你 **必须优先调用 [ext_knowledge_search] 工具**。
2. 将用户的意图提炼为精确的搜索词汇传入 query 参数中，检索内部 Elasticsearch 知识库。
3. 严格基于检索到的知识片段进行回答。如果检索不到，明确告知“知识库中未找到相关规范”，绝不能凭空捏造。

【输出规范】
1. 语气保持严谨、客观，体现大厂安全专家的素养。
2. 在引用规范时，请标明数据来源（如：“根据《公司内部安全规范》...”）。`

// HandleThreatHunterChat 威胁狩猎专家接口
// @Summary      ThreatHunter 核心调度接口
// @Description  基于 MCP 协议和 ReAct 引擎，提供自动化安全研判与本地沙箱执行能力
// @Tags         ZeroTrust AI 引擎
// @Accept json
// @Produce text/event-stream
// @Param request body dto.ThreatHunterRequest true "智能体对话请求参数"
// @Success 200 {string} string "data: {\"content\": \"...\"}\n\ndata: [DONE]\n\n"
// @Failure 400 {object} map[string]interface{} "{\"error\": \"参数错误\"}"
// @Failure 500 {object} map[string]interface{} "{\"error\": \"不支持流式或推理中断\"}"
// @Router /secops/threat_hunter [post]
func (ac *AIController) HandleThreatHunterChat(c *gin.Context) {
	// 配合你的 Swagger 注释，使用 dto 层的结构体
	var req dto.ThreatHunterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 启动核心 ReAct 引擎，注入 ThreatHunter 灵魂
	ac.RunReAct(c, req.SessionID, req.Message, ThreatHunterPrompt)
}

// HandleComplianceChat DevSecOps 合规顾问处理逻辑
// @Summary DevSecOps 合规审查对话 (SSE 流式)
// @Description AI 合规顾问专用接口，基于企业零信任规范提供安全编码审查、漏洞修复建议与告警初步研判。具备独立的历史会话隔离 (Redis)。使用 Server-Sent Events (SSE) 协议实时返回审查分析结果。
// @Tags ZeroTrust AI 引擎
// @Accept json
// @Produce text/event-stream
// @Param request body dto.StreamChatRequest true "包含 SessionID 的待审计代码片段、架构描述或告警日志"
// @Success 200 {string} string "data: {\"content\": \"...\"}\n\ndata: [DONE]\n\n"
// @Failure 400 {object} map[string]interface{} "{\"error\": \"无效的安全审计请求参数\"}"
// @Failure 500 {object} map[string]interface{} "{\"error\": \"流式安全连接建立失败...\"}"
// @Router /secops/compliance [post]
func (ac *AIController) HandleComplianceChat(c *gin.Context) {
	// 配合你的 Swagger 注释，使用 dto 层的结构体
	var req dto.StreamChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的安全审计请求参数: " + err.Error()})
		return
	}

	// 启动核心 ReAct 引擎，注入 Compliance Copilot 灵魂
	ac.RunReAct(c, req.SessionID, req.Message, ComplianceCopilotPrompt)
}

// RunReAct 核心智能体调度引擎 (剥离出来的核心逻辑)
func (ac *AIController) RunReAct(c *gin.Context, sessionID, message, systemPrompt string) {
	ctx := context.Background()
	redisKey := "agent_chat_history:" + sessionID

	// ==================== [步骤 0：开启 SSE 流式通道] ====================
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "不支持流式"})
		return
	}

	// ==================== [步骤 1：加载上下文与动态插件库] ====================
	availableTools := agent.GlobalRegistry.GetAllToolInfos()
	log.Printf("当前已加载工具数量: %d", len(availableTools))

	// 注入传入的 System Prompt
	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
	}

	// 从 Redis 拉取历史消息
	historyStrList, _ := repository.RedisClient.LRange(ctx, redisKey, -10, -1).Result()
	for _, histStr := range historyStrList {
		var msg ChatMessage // 确保你的文件里定义了 ChatMessage 结构体
		if err := json.Unmarshal([]byte(histStr), &msg); err == nil {
			if msg.Role == "user" {
				messages = append(messages, schema.UserMessage(msg.Content))
			} else if msg.Role == "assistant" {
				messages = append(messages, schema.AssistantMessage(msg.Content, nil))
			}
		}
	}

	// 压入用户当前新问题
	messages = append(messages, schema.UserMessage(message))

	// 记录用户当前新消息
	userMsgBytes, _ := json.Marshal(ChatMessage{Role: "user", Content: message})
	repository.RedisClient.RPush(ctx, redisKey, userMsgBytes)
	repository.RedisClient.Expire(ctx, redisKey, 24*time.Hour)

	initialPrompt, _ := json.Marshal(gin.H{"content": "🧠 *Agent 开始推演...*\n\n"})
	c.Writer.Write([]byte("data: " + string(initialPrompt) + "\n\n"))
	flusher.Flush()

	// ==================== [步骤 2：核心 ReAct 引擎 (改良流式版)] ====================
	maxSteps := 10
	var fullChatHistory strings.Builder // 记录完整的聊天历史存入 Redis

	for step := 1; step <= maxSteps; step++ {
		// 调用大模型进行推理决策
		resp, err := ac.chatModel.Generate(ctx, messages, model.WithTools(availableTools))
		if err != nil {
			errData, _ := json.Marshal(gin.H{"content": "\n❌ 推理中断: " + err.Error()})
			c.Writer.Write([]byte("data: " + string(errData) + "\n\ndata: [DONE]\n\n"))
			flusher.Flush()
			return
		}

		// 实时播报 AI 的思考过程 (Thought)
		log.Printf("\n========== [ReAct Step %d] ==========", step)
		if len(resp.Content) > 0 {
			log.Printf("大模型输出正文截取: %s...", resp.Content)
		}

		if resp.Content != "" {
			runes := []rune(resp.Content)
			for _, r := range runes {
				chunkData, _ := json.Marshal(gin.H{"content": string(r)})
				c.Writer.Write([]byte("data: " + string(chunkData) + "\n\n"))
				flusher.Flush()
				time.Sleep(10 * time.Millisecond) // 流畅打字机效果
			}
			fullChatHistory.WriteString(resp.Content + "\n")
		}

		// 场景 A：AI 触发了真正的原生工具调用 (Action)
		if len(resp.ToolCalls) > 0 {
			messages = append(messages, resp) // 把 AI 的调用意图压入上下文

			for _, toolCall := range resp.ToolCalls {
				toolName := toolCall.Function.Name
				toolArgs := toolCall.Function.Arguments

				plugin, err := agent.GlobalRegistry.GetTool(toolName)
				if err != nil {
					errMsg := fmt.Sprintf("⚠️ 警告: AI 试图调用未注册的工具 [%s]", toolName)
					log.Println(errMsg)
					progressMsg, _ := json.Marshal(gin.H{"content": fmt.Sprintf("\n*(Step %d) %s...*\n", step, errMsg)})
					c.Writer.Write([]byte("data: " + string(progressMsg) + "\n\n"))
					flusher.Flush()

					messages = append(messages, schema.ToolMessage("Error: 你调用的工具不存在。", toolCall.ID))
					continue
				}

				// 播报插件调度状态
				progressMsg, _ := json.Marshal(gin.H{"content": fmt.Sprintf("\n*(Step %d) ⚙️ 正在调用插件 [%s]...*\n", step, toolName)})
				c.Writer.Write([]byte("data: " + string(progressMsg) + "\n\n"))
				flusher.Flush()

				// 执行工具逻辑
				toolResultMsg, toolErr := plugin.Execute(toolArgs)
				if toolErr != nil {
					toolResultMsg = "工具执行失败：" + toolErr.Error()
				}
				log.Printf("🛠️ 插件 [%s] 返回给 AI 的结果: %s", toolName, toolResultMsg)
				messages = append(messages, schema.ToolMessage(toolResultMsg, toolCall.ID))
			}

			// 继续下一轮思考
			continue
		}

		// 场景 B：没有工具调用，说明任务全部圆满完成 (Final Answer)
		break
	}

	if fullChatHistory.Len() == 0 {
		fullChatHistory.WriteString("\n*(系统提示：Agent 思考超过最大轮数，强制终止)*")
		errData, _ := json.Marshal(gin.H{"content": fullChatHistory.String()})
		c.Writer.Write([]byte("data: " + string(errData) + "\n\n"))
	}

	// ==================== [步骤 3：保存记忆并关闭通道] ====================
	aiMsgBytes, _ := json.Marshal(ChatMessage{Role: "assistant", Content: fullChatHistory.String()})
	repository.RedisClient.RPush(ctx, redisKey, aiMsgBytes)

	c.Writer.Write([]byte("data: [DONE]\n\n"))
	flusher.Flush()
}
