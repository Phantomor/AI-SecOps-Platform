// cmd/script/init_es_data.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"sentinel-agent-go/internal/config"
	"sentinel-agent-go/internal/repository"

	"github.com/cloudwego/eino-ext/components/embedding/openai"
)

// InterviewQA 定义要存入 ES 的数据结构
type InterviewQA struct {
	Question  string    `json:"question"`
	Answer    string    `json:"answer"`
	Embedding []float64 `json:"embedding"` // 核心：存储大模型生成的向量
}

func main() {
	// 1. 加载配置并初始化 ES
	config.LoadConfig("configs/application.yaml")
	repository.InitES()

	ctx := context.Background()
	indexName := config.Global.Elasticsearch.IndexName // "interview_knowledge_base"

	// 2. 初始化 Eino 的 Embedding 客户端 (把文字变成向量)
	// 注意：阿里云兼容 OpenAI 的 Embedding 接口
	embedder, err := openai.NewEmbedder(ctx, &openai.EmbeddingConfig{
		APIKey:  config.Global.AI.OpenAI.APIKey,
		BaseURL: config.Global.AI.OpenAI.BaseURL,
		Model:   "text-embedding-v1", // 阿里云的文本向量模型，输出维度是 1536
	})
	if err != nil {
		log.Fatalf("❌ Embedder 初始化失败: %v", err)
	}

	// 3. 在 ES 中创建“表结构” (Mapping)
	// 如果索引已存在，先删除它（方便我们反复测试运行）
	repository.ESClient.Indices.Delete([]string{indexName}, repository.ESClient.Indices.Delete.WithIgnoreUnavailable(true))

	// 定义 ES 的 Mapping：告诉 ES 我们要存向量，并且使用余弦相似度(cosine)来计算
	mapping := `{
		"mappings": {
			"properties": {
				"question": { "type": "text" },
				"answer": { "type": "text" },
				"embedding": {
					"type": "dense_vector",
					"dims": 1536,
					"index": true,
					"similarity": "cosine"
				}
			}
		}
	}`
	res, err := repository.ESClient.Indices.Create(
		indexName,
		repository.ESClient.Indices.Create.WithBody(strings.NewReader(mapping)),
	)
	if err != nil || res.IsError() {
		log.Fatalf("❌ 创建 ES 索引失败: %v", res)
	}
	log.Println("✅ 成功在 ES 中创建向量索引:", indexName)

	// 4. 准备我们的“初始面试题库”
	mockData := []struct {
		Q string
		A string
	}{
		{"Go语言的 Goroutine 和操作系统的线程有什么区别？", "Goroutine 是 Go 语言层面实现的轻量级用户态线程。它的初始内存占用极小（仅 2KB），而且由 Go 运行时的 G-P-M 调度器负责调度，上下文切换成本远低于操作系统线程。"},
		{"Go 的 Slice 扩容机制是怎样的？", "在 Go 1.18 之前，当容量小于 1024 时每次扩容翻倍，大于等于 1024 时每次增加 25%。Go 1.18 之后引入了平滑过渡的扩容公式，避免了容量阈值处的跃变。核心目的是平摊内存分配的开销。"},
		{"MySQL 的 B+ 树索引为什么比 B 树更适合做数据库索引？", "B+ 树的所有数据都存在叶子节点，非叶子节点只存索引键，这样每个磁盘页能装下更多的索引，降低了树的高度，从而减少了磁盘 IO 次数。同时叶子节点之间用双向链表连接，非常适合范围查询。"},
	}

	// 5. 遍历题库 -> 调用大模型生成向量 -> 存入 ES
	for i, item := range mockData {
		log.Printf("⏳ 正在处理第 %d 题: %s\n", i+1, item.Q)

		// 调用 Embedding 接口，将问题和答案组合起来进行向量化
		textToEmbed := "问题：" + item.Q + "\n答案：" + item.A
		vectors, err := embedder.EmbedStrings(ctx, []string{textToEmbed})
		if err != nil || len(vectors) == 0 {
			log.Fatalf("❌ 生成向量失败: %v", err)
		}

		// 组装要存入 ES 的 JSON 文档
		qaDoc := InterviewQA{
			Question:  item.Q,
			Answer:    item.A,
			Embedding: vectors[0], // 获取浮点数数组
		}
		docBytes, _ := json.Marshal(qaDoc)

		// 写入 ES
		docID := fmt.Sprintf("qa_%d", i+1)
		res, err = repository.ESClient.Index(
			indexName,
			bytes.NewReader(docBytes),
			repository.ESClient.Index.WithDocumentID(docID),
			repository.ESClient.Index.WithRefresh("true"), // 强制刷新，保证写入后立即可查
		)
		if err != nil || res.IsError() {
			log.Printf("❌ 写入 ES 失败: %v", res)
		} else {
			log.Printf("✅ 写入成功！DocID: %s\n", docID)
		}
	}

	log.Println("🎉 知识库初始化完毕！你的 AI 现在有题库了！")
}
