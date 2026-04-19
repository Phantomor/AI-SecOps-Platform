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
	"sentinel-agent-go/internal/config"
	"sentinel-agent-go/internal/dto"
	"sentinel-agent-go/internal/pkg"
	"sentinel-agent-go/internal/repository"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
)

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

	// [ MySQL 持久化] 从 JWT 中间件获取 UserID，如果没有则默认为 0 (匿名)
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

	// [ MySQL 持久化] 保存用户发送的消息到数据库 (使用 goroutine 异步执行，不阻塞主流程)
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

			// [ MySQL 持久化] 对话结束时：异步保存 AI 的完整回答到 MySQL
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
		// 修改 1：使用统一的 Error 格式
		pkg.Error(c, 400, "参数错误，需提供 session_id, resume 和 jd")
		return
	}

	ctx := context.Background()
	// 修改 2：对齐隔离后的 Redis Key！(根据你给超级智能体定义的前缀来)
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

	// 修改 3：增强 Markdown 清理容错能力
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
	// 修改 4：使用统一的 Success 格式返回
	pkg.Success(c, report)
}
