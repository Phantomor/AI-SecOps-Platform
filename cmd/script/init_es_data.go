package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"sentinel-agent-go/internal/config"
	"sentinel-agent-go/internal/pkg/llm"
	"sentinel-agent-go/internal/repository"

	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// 配置文件路径和知识库目录
const (
	configPath   = "configs/application.yaml"
	knowledgeDir = "assets/knowledge" // 确保这个目录存在且里面有 .md 文件
)

func main() {
	// 1. 初始化配置和 ES 连接
	config.LoadConfig(configPath)
	repository.InitES()
	ctx := context.Background()

	indexName := "sec_knowledge_base"
	es := repository.ESClient

	log.Println("🚀 开始初始化 Elasticsearch 向量知识库 (本地文件直读模式)...")

	// 2. 删除旧索引（防止反复测试时因为 Schema 冲突报错）
	es.Indices.Delete([]string{indexName}, es.Indices.Delete.WithIgnoreUnavailable(true))

	// 3. 创建带有 dense_vector (稠密向量) 的索引 Mapping
	// ⚠️ 注意：这里配置的 dims 必须和你使用的 Embedding 模型维度一致 (千问 v2 是 1536)
	mapping := `{
		"mappings": {
			"properties": {
				"title": { "type": "keyword" },
				"content": { "type": "text" },
				"vector": {
					"type": "dense_vector",
					"dims": 1536,
					"index": true,
					"similarity": "cosine"
				}
			}
		}
	}`
	res, err := es.Indices.Create(indexName, es.Indices.Create.WithBody(strings.NewReader(mapping)))
	if err != nil || res.IsError() {
		log.Fatalf("❌ 创建索引失败: %v", err)
	}
	log.Println("✅ 索引 sec_knowledge_base 创建成功！")

	// 4. 遍历本地目录，读取知识库文件
	files, err := os.ReadDir(knowledgeDir)
	if err != nil {
		log.Fatalf("❌ 读取知识库目录 [%s] 失败，请确保目录存在: %v", knowledgeDir, err)
	}

	docCount := 0
	for _, file := range files {
		// 跳过文件夹和非 Markdown 文件 (可选)
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(knowledgeDir, file.Name())
		contentBytes, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("⚠️ 读取文件 %s 失败: %v", file.Name(), err)
			continue
		}
		contentStr := string(contentBytes)

		// ⚠️ 简易 Chunking (切片)：在企业级项目中，通常会按特定段落长度切分。
		// 这里为了演示，我们假设每个文件都是一个合理的独立 Chunk。

		log.Printf("🧠 正在调用千问 API 向量化文档: [%s]...", file.Name())

		// 调用 Embedding 接口把文本变向量
		vector, err := llm.GenerateEmbedding(contentStr)
		if err != nil {
			log.Printf("❌ 文件 %s 向量化失败，跳过: %v", file.Name(), err)
			continue
		}

		// 组装要插入 ES 的数据体 (包含文件名 title，内容 content，和向量 vector)
		docData := map[string]interface{}{
			"title":   file.Name(),
			"content": contentStr,
			"vector":  vector,
		}
		docBytes, _ := json.Marshal(docData)

		req := esapi.IndexRequest{
			Index:      indexName,
			DocumentID: fmt.Sprintf("doc_%s", file.Name()), // 用文件名作 ID 避免重复
			Body:       bytes.NewReader(docBytes),
			Refresh:    "true", // 强制刷新，确保存入后立刻可查
		}

		res, err := req.Do(ctx, es)
		if err != nil || res.IsError() {
			log.Printf("❌ 文件 %s 写入 ES 失败: %v", file.Name(), err)
			continue
		}

		docCount++
		log.Printf("✅ 文件 [%s] 成功写入 Elasticsearch！", file.Name())
	}

	log.Printf("🎉 知识库初始化大功告成！共成功处理并存储了 %d 篇文档。", docCount)
}
