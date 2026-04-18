// internal/controller/ai_controller.go
package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
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
		// 新增：高大上的 SSE 流式接口
		api.POST("/chat/stream", ac.HandleStreamChat)
		api.POST("/interview/report", ac.GenerateReport)

		// 1. AI 恋爱大师专用路由
		api.POST("/secops/compliance", ac.HandleComplianceChat)

		// 2. 超级智能体 (带工具调用) 专用路由
		api.POST("/secops/threat_hunter", ac.HandleThreatHunterChat)
	}
}

// HandleChat 处理基础对话请求
// @Summary 基础对话 (非流式)
// @Description 处理基础的聊天请求，返回完整的字符串响应。
// @Tags AI 对话模块 (AI Chat)
// @Accept json
// @Produce json
// @Param request body dto.ChatRequest true "对话内容"
// @Success 200 {object} map[string]interface{} "{"code": 200, "data": "AI回答的内容..."}"
// @Failure 400 {object} map[string]interface{} "{"error": "无效的请求参数"}"
// @Failure 500 {object} map[string]interface{} "{"error": "大脑宕机了: ..."}"
// @Router /chat [post]
func (ac *AIController) HandleChat(c *gin.Context) {
	// 1. 定义接收前端请求的结构体
	var req struct {
		Message string `json:"message" binding:"required"`
	}

	// 2. 解析前端传来的 JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 3. 构建发给大模型的消息 (eino 的标准 Schema)
	messages := []*schema.Message{
		// 赋予 AI 面试官的人设 (系统提示词)
		schema.SystemMessage("你是一个资深的 Go 语言后端面试官。请用严谨、专业的语气回答用户的问题。"),
		// 用户的提问
		schema.UserMessage(req.Message),
	}

	// 4. 调用大模型生成回答
	resp, err := ac.chatModel.Generate(c.Request.Context(), messages)
	if err != nil {
		fmt.Println("❌ 阿里云 API 详细报错:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "大脑宕机了: " + err.Error()})
		return
	}

	// 5. 将 AI 的回答返回给前端
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": resp.Content, // 提取 AI 回答的文本内容
	})
}

// 定义存入 Redis 的消息结构
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// HandleStreamChat 核心：SSE 流式对话处理 (已整合 MySQL 持久化)
// @Summary 对话 (SSE 流式)
// @Description 带有 RAG 知识库检索、上下文记忆、MySQL持久化的流式对话接口。使用 Server-Sent Events (SSE) 协议返回数据。
// @Tags AI 对话模块 (AI Chat)
// @Accept json
// @Produce text/event-stream
// @Param request body dto.StreamChatRequest true "包含 SessionID 的对话内容"
// @Success 200 {string} string "data: {"content": "..."}\n\ndata: [DONE]\n\n"
// @Failure 400 {object} map[string]interface{} "{"error": "无效的请求参数"}"
// @Failure 500 {object} map[string]interface{} "{"error": "流式连接失败..."}"
// @Router /chat/stream [post]
func (ac *AIController) HandleStreamChat(c *gin.Context) {
	// 1. 接收参数
	var req struct {
		// session_id 用于区分不同用户的会话
		SessionID string `json:"session_id" binding:"required"`
		Message   string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// [新增 MySQL 持久化] 从 JWT 中间件获取 UserID，如果没有则默认为 0 (匿名)
	userIDVal, _ := c.Get("currentUserID")
	userID, _ := userIDVal.(uint)

	ctx := context.Background()

	// ==================== [ RAG 模块：检索知识库] ====================
	// 1. 将用户的新问题变成向量
	vectors, err := ac.embedder.EmbedStrings(ctx, []string{req.Message})
	var ragContext string
	if err == nil && len(vectors) > 0 {
		userVector := vectors[0]

		// 2. 构建 ES 的 KNN 向量检索请求 (寻找最相似的 2 条记录)
		esQuery := map[string]interface{}{
			"knn": map[string]interface{}{
				"field":          "embedding",
				"query_vector":   userVector,
				"k":              2,
				"num_candidates": 10,
			},
			"_source": []string{"question", "answer"}, // 我们只需要查出问题和答案，不需要把巨大的向量数字查回来
		}
		queryBytes, _ := json.Marshal(esQuery)

		// 3. 执行 ES 搜索
		res, err := repository.ESClient.Search(
			repository.ESClient.Search.WithContext(ctx),
			repository.ESClient.Search.WithIndex(config.Global.Elasticsearch.IndexName),
			repository.ESClient.Search.WithBody(bytes.NewReader(queryBytes)),
		)

		if err == nil && !res.IsError() {
			defer res.Body.Close()
			var esResult map[string]interface{}
			json.NewDecoder(res.Body).Decode(&esResult)

			// 4. 解析检索到的标准答案，拼接到 ragContext 中
			hits := esResult["hits"].(map[string]interface{})["hits"].([]interface{})
			if len(hits) > 0 {
				ragContext = "【系统提供的标准参考答案如下：】\n"
				for i, hit := range hits {
					source := hit.(map[string]interface{})["_source"].(map[string]interface{})
					ragContext += fmt.Sprintf("%d. 问题：%s\n答案：%s\n", i+1, source["question"], source["answer"])
				}
			}
		}
	}
	// 如果知识库里查不到，打印个日志（不阻断流程，让模型用自己的知识强答）
	if ragContext == "" {
		log.Println("⚠️ 知识库未命中相关考点")
	} else {
		log.Println("🔍 知识库成功命中！注入上下文：\n", ragContext)
	}

	redisKey := "chat_history:" + req.SessionID

	// ==================== [记忆模块 1：提取历史对话] ====================
	// 基础人设（永远在最前面）
	messages := []*schema.Message{
		schema.SystemMessage("你是一个资深的 Go 语言后端面试官。请根据上下文，连贯地回答用户的问题。"),
	}

	// 从 Redis 的 List 中拉取最近的 10 条对话记录（防止 Token 爆表，这里做了一个简单的滑动窗口）
	historyStrList, _ := repository.RedisClient.LRange(ctx, redisKey, -10, -1).Result()
	for _, histStr := range historyStrList {
		var msg ChatMessage
		if err := json.Unmarshal([]byte(histStr), &msg); err == nil {
			if msg.Role == "user" {
				messages = append(messages, schema.UserMessage(msg.Content))
			} else if msg.Role == "assistant" {
				messages = append(messages, schema.AssistantMessage(msg.Content, nil))
			}
		}
	}

	// 巧妙的一步：把 RAG 的标准答案和用户的问题合并，一起当做 UserMessage 发过去
	finalUserMessage := req.Message
	if ragContext != "" {
		finalUserMessage = ragContext + "\n\n【用户当前的提问/回答】：\n" + req.Message
	}

	// 把用户当前的新问题加到最后
	messages = append(messages, schema.UserMessage(finalUserMessage))

	// ==================== [记忆模块 2：保存用户新问题] ====================
	userMsgBytes, _ := json.Marshal(ChatMessage{Role: "user", Content: req.Message})
	repository.RedisClient.RPush(ctx, redisKey, userMsgBytes)
	// 设置过期时间，比如 24 小时后自动清除这轮面试记录
	repository.RedisClient.Expire(ctx, redisKey, 24*3600*1000*1000*1000)

	// [新增 MySQL 持久化] 保存用户发送的消息到数据库 (使用 goroutine 异步执行，不阻塞主流程)
	go ac.SaveMessageToDB(req.SessionID, userID, "user", req.Message)

	// 2. 调用大模型（流式）
	streamReader, err := ac.chatModel.Stream(c.Request.Context(), messages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "流式连接失败: " + err.Error()})
		return
	}
	defer streamReader.Close()

	// 3. SSE 头部设置
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "您的环境不支持流式传输"})
		return
	}

	// 准备一个变量，用来拼接 AI 断断续续吐出来的完整回答
	var fullAIResponse strings.Builder

	// 4. 流式推送循环
	for {
		chunk, err := streamReader.Recv()
		if err == io.EOF {
			// ==================== [记忆模块 3：对话结束，保存 AI 的完整回答] ====================
			aiMsgBytes, _ := json.Marshal(ChatMessage{Role: "assistant", Content: fullAIResponse.String()})
			repository.RedisClient.RPush(ctx, redisKey, aiMsgBytes)

			// [新增 MySQL 持久化] 对话结束时：异步保存 AI 的完整回答到 MySQL
			// 使用 goroutine 避免阻塞 SSE 连接的关闭
			go ac.SaveMessageToDB(req.SessionID, userID, "assistant", fullAIResponse.String())

			c.Writer.Write([]byte("data: [DONE]\n\n"))
			flusher.Flush()
			break
		}
		if err != nil {
			break
		}

		// 拼接字符串
		fullAIResponse.WriteString(chunk.Content)

		// 推给前端
		chunkData, _ := json.Marshal(gin.H{"content": chunk.Content})
		c.Writer.Write([]byte("data: " + string(chunkData) + "\n\n"))
		flusher.Flush()
	}
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
	var req struct {
		// session_id 用于区分不同用户的会话
		SessionID string `json:"session_id" binding:"required"`
		Message   string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	ctx := context.Background()

	// 🌟 修改 1：隔离 Redis Key，加上专属的前缀 love_chat_history:
	redisKey := "love_chat_history:" + req.SessionID

	// 🌟 修改 2：初始切片里只放 System 人设，绝对不要在这里放当前的 req.Message
	messages := []*schema.Message{
		schema.SystemMessage(`你现在的身份是「ZeroTrust Sentinel」系统中的 DevSecOps 合规顾问。
    你的职责是协助研发和运维团队，解释企业零信任安全规范，提供安全编码建议，并辅助进行告警日志的初步研判。

    【工作流规范】
    1. 接收到用户的报错代码或异常日志后，指出可能存在的安全隐患（如 SQL 注入、XSS、越权访问）。
    2. 提供符合大厂规范的修复代码片段。
    3. 语气保持专业、严谨且富有指导性。`),
	}

	// 从 Redis 取出历史记录拼接到后面
	historyStrList, _ := repository.RedisClient.LRange(ctx, redisKey, -10, -1).Result()
	for _, histStr := range historyStrList {
		var msg ChatMessage
		if err := json.Unmarshal([]byte(histStr), &msg); err == nil {
			if msg.Role == "user" {
				messages = append(messages, schema.UserMessage(msg.Content))
			} else if msg.Role == "assistant" {
				messages = append(messages, schema.AssistantMessage(msg.Content, nil))
			}
		}
	}

	// 🌟 修改 3：遍历完历史记录后，最后再把用户当前的提问压入切片
	messages = append(messages, schema.UserMessage(req.Message))

	// ==================== [记忆模块：保存用户新问题] ====================
	userMsgBytes, _ := json.Marshal(ChatMessage{Role: "user", Content: req.Message})
	repository.RedisClient.RPush(ctx, redisKey, userMsgBytes)

	// 🌟 修改 4：使用 time.Hour 规范化过期时间
	repository.RedisClient.Expire(ctx, redisKey, 24*time.Hour)

	// 调用大模型（流式）
	streamReader, err := ac.chatModel.Stream(c.Request.Context(), messages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "流式连接失败: " + err.Error()})
		return
	}
	defer streamReader.Close()

	// SSE 头部设置
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "您的环境不支持流式传输"})
		return
	}

	var fullAIResponse strings.Builder

	// 流式推送循环
	for {
		chunk, err := streamReader.Recv()
		if err == io.EOF {
			// 对话结束，保存 AI 的完整回答
			aiMsgBytes, _ := json.Marshal(ChatMessage{Role: "assistant", Content: fullAIResponse.String()})
			repository.RedisClient.RPush(ctx, redisKey, aiMsgBytes)

			c.Writer.Write([]byte("data: [DONE]\n\n"))
			flusher.Flush()
			break
		}
		if err != nil {
			break
		}

		fullAIResponse.WriteString(chunk.Content)

		chunkData, _ := json.Marshal(gin.H{"content": chunk.Content})
		c.Writer.Write([]byte("data: " + string(chunkData) + "\n\n"))
		flusher.Flush()
	}
}

// GenerateReport 生成结构化评估报告
// @Summary 生成面试报告
// @Description 根据候选人历史面试记录、简历和 JD，调用大模型生成结构化的面试评估 JSON 报告。
// @Tags AI 对话模块 (AI Chat)
// @Accept json
// @Produce json
// @Param request body dto.ReportRequest true "报告生成请求参数"
// @Success 200 {object} map[string]interface{} "{"code": 0, "data": {"score": 85, "strengths": [...], "hire_conclusion": "Hire"}}"
// @Failure 400 {object} map[string]interface{} "{"code": 400, "message": "参数错误或没有找到面试记录"}"
// @Failure 500 {object} map[string]interface{} "{"code": 500, "message": "生成报告失败"}"
// @Router /interview/report [post]
func (ac *AIController) GenerateReport(c *gin.Context) {
	var req dto.ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 🌟 修改 1：使用统一的 Error 格式
		pkg.Error(c, 400, "参数错误，需提供 session_id, resume 和 jd")
		return
	}

	ctx := context.Background()
	// 🌟 修改 2：对齐隔离后的 Redis Key！(根据你给超级智能体定义的前缀来)
	redisKey := "agent_chat_history:" + req.SessionID

	// 1. 从 Redis 提取完整的面试记录记录（作为评判依据）
	historyStrList, _ := repository.RedisClient.LRange(ctx, redisKey, 0, -1).Result()
	if len(historyStrList) == 0 {
		pkg.Error(c, 400, "没有找到该候选人的面试记录")
		return
	}

	// 将历史记录拼接成纯文本供大模型阅读
	var chatHistoryBuilder strings.Builder
	for _, histStr := range historyStrList {
		var msg ChatMessage
		json.Unmarshal([]byte(histStr), &msg)
		chatHistoryBuilder.WriteString(msg.Role + ": " + msg.Content + "\n")
	}

	// 2. 核心：Go 原生 Template 特性动态注入数据
	const promptTpl = `
你是一位严厉且专业的研发总监。请根据以下信息，对候选人进行综合评估。

【岗位要求 (JD)】
{{.JD}}

【候选人简历片段】
{{.Resume}}

【面试问答记录】
{{.ChatHistory}}

【输出要求 (结构化输出)】
请务必严格按照以下 JSON 格式输出你的评估结果，不要输出任何额外的 Markdown 标记（如 ` + "```json" + `）或解释性文字，只输出纯 JSON 对象：
{
  "score": 85,
  "strengths": ["熟悉 Go 并发", "回答逻辑清晰"],
  "weaknesses": ["对底层调度理解不深"],
  "hire_conclusion": "Hire",
  "comments": "该候选人基础扎实，符合岗位要求..."
}
`
	// 解析模板并注入动态变量
	t, _ := template.New("report").Parse(promptTpl)
	var finalPrompt bytes.Buffer
	t.Execute(&finalPrompt, map[string]interface{}{
		"JD":          req.JD,
		"Resume":      req.Resume,
		"ChatHistory": chatHistoryBuilder.String(),
	})

	// 3. 构建发给大模型的消息
	messages := []*schema.Message{
		schema.UserMessage(finalPrompt.String()),
	}

	// 4. 调用大模型 (注意这里用普通的 Generate，不需要流式)
	resp, err := ac.chatModel.Generate(c.Request.Context(), messages)
	if err != nil {
		pkg.Error(c, 500, "生成报告失败: "+err.Error())
		return
	}

	// 5. 校验结构化输出：尝试将大模型返回的文本反序列化为我们的 Go 结构体
	var report dto.InterviewReport

	// 🌟 修改 3：增强 Markdown 清理容错能力
	cleanJSON := strings.TrimSpace(resp.Content)
	cleanJSON = strings.TrimPrefix(cleanJSON, "```json\n")
	cleanJSON = strings.TrimPrefix(cleanJSON, "```\n")
	cleanJSON = strings.TrimSuffix(cleanJSON, "\n```")
	cleanJSON = strings.TrimSuffix(cleanJSON, "```")
	cleanJSON = strings.TrimSpace(cleanJSON)

	if err := json.Unmarshal([]byte(cleanJSON), &report); err != nil {
		// 如果大模型不听话，返回的不是标准 JSON，则走降级处理
		pkg.Success(c, gin.H{
			"msg":      "报告生成成功(非标准格式)",
			"raw_data": resp.Content,
		})
		return
	}

	// 6. 成功返回严格的 JSON 数据给前端
	// 🌟 修改 4：使用统一的 Success 格式返回
	pkg.Success(c, report)
}

// HandleThreatHunterChat 威胁狩猎专家接口
// @Summary      ThreatHunter 核心调度接口
// @Description  基于 MCP 协议和 ReAct 引擎，提供自动化安全研判与本地沙箱执行能力
// @Tags         ZeroTrust AI 引擎
// @Accept json
// @Produce text/event-stream
// @Param request body dto.ThreatHunterRequest true "智能体对话请求参数"
// @Success 200 {string} string "data: {"content": "..."}\n\ndata: [DONE]\n\n"
// @Failure 400 {object} map[string]interface{} "{"error": "参数错误"}"
// @Failure 500 {object} map[string]interface{} "{"error": "不支持流式或推理中断"}"
// @Router /secops/threat_hunter [post]
func (ac *AIController) HandleThreatHunterChat(c *gin.Context) {
	var req struct {
		SessionID string `json:"session_id" binding:"required"`
		Message   string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	ctx := context.Background()
	redisKey := "agent_chat_history:" + req.SessionID

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
	toolsJson, _ := json.MarshalIndent(availableTools, "", "  ")
	log.Println("传给大模型的工具详情:", string(toolsJson))

	// 🌟 核心修复 1：终极破冰 System Prompt，杜绝大模型幻觉与拒绝服务
	messages := []*schema.Message{
		schema.SystemMessage(`你现在的身份是「ZeroTrust Sentinel」系统中的核心大模型节点：ThreatHunter（威胁狩猎专家）。
你是一位精通网络安全、Android 恶意软件分析（TA-HCL 概念漂移）和数据库底层操作的资深安全架构师。

【核心能力与工作流】
1. 当遇到安全分析请求时，你必须优先考虑使用你挂载的 MCP 工具（如 ext_sqlite_read_query）去本地数据库提取恶意软件特征或网络日志。
2. 对于未知漏洞或安全动态，请使用 web_search 工具检索 CVE 库或最新安全播报。
3. 如果需要分析混淆代码或测试加密算法，请使用 execute_go_code 工具在沙箱中运行验证。

【输出规范】
1. 语气必须极其专业、冷峻、客观，严禁使用任何轻浮、口语化或拟人化的表达（如“亲爱的”、“我觉得”）。
2. 在展示 IP 地址、Hash 值、恶意样本特征或数据库查询结果时，必须使用 Markdown 表格进行格式化。
3. 你的分析报告必须包含：[事件概述]、[数据取证/调用记录]、[威胁定性]、[处置建议] 四个模块。
`),
	}

	// 从 Redis 拉取历史消息
	historyStrList, _ := repository.RedisClient.LRange(ctx, redisKey, -10, -1).Result()
	for _, histStr := range historyStrList {
		var msg ChatMessage
		if err := json.Unmarshal([]byte(histStr), &msg); err == nil {
			if msg.Role == "user" {
				messages = append(messages, schema.UserMessage(msg.Content))
			} else if msg.Role == "assistant" {
				messages = append(messages, schema.AssistantMessage(msg.Content, nil))
			}
		}
	}

	// 压入用户当前新问题
	messages = append(messages, schema.UserMessage(req.Message))

	// 记录用户当前新消息
	userMsgBytes, _ := json.Marshal(ChatMessage{Role: "user", Content: req.Message})
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

		// 🌟 核心修复 2：实时播报 AI 的思考过程 (Thought)，解决前端假死问题
		// 如果大模型在调用工具前输出了思考文本，我们立刻通过流式打字机推给前端
		log.Printf("\n========== [ReAct Step %d] ==========", step)
		log.Printf("大模型输出文本长度: %d 字符", len(resp.Content))
		if len(resp.Content) > 0 {
			log.Printf("大模型输出正文截取: %s...", resp.Content)
		}
		log.Printf("大模型请求调用工具数量: %d", len(resp.ToolCalls))
		log.Println("======================================")

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
			// ⚠️ 必须把 AI 的调用意图压入上下文
			messages = append(messages, resp)

			for _, toolCall := range resp.ToolCalls {
				toolName := toolCall.Function.Name
				toolArgs := toolCall.Function.Arguments

				plugin, err := agent.GlobalRegistry.GetTool(toolName)
				if err != nil {
					// 🌟 核心修复 1：把 AI 调错工具的行为暴露出来！
					errMsg := fmt.Sprintf("⚠️ 警告: AI 试图调用未注册的工具 [%s]", toolName)
					log.Println(errMsg) // 后端打印

					// 前端播报
					progressMsg, _ := json.Marshal(gin.H{"content": fmt.Sprintf("\n*(Step %d) %s...*\n", step, errMsg)})
					c.Writer.Write([]byte("data: " + string(progressMsg) + "\n\n"))
					flusher.Flush()

					messages = append(messages, schema.ToolMessage("Error: 你调用的工具不存在，请检查工具名称是否正确，或者更换其他可用工具。", toolCall.ID))
					continue
				}

				// 向前端实时播报插件调度状态
				progressMsg, _ := json.Marshal(gin.H{"content": fmt.Sprintf("\n*(Step %d) ⚙️ 正在调用插件 [%s]...*\n", step, toolName)})
				c.Writer.Write([]byte("data: " + string(progressMsg) + "\n\n"))
				flusher.Flush()

				// 执行工具逻辑 (Observation)
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
