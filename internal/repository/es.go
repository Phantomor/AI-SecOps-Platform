// internal/repository/es.go
package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"sentinel-agent-go/internal/config"

	"github.com/elastic/go-elasticsearch/v8"
)

// ESClient 全局 Elasticsearch 客户端实例
var ESClient *elasticsearch.Client

// InitES 初始化 Elasticsearch 连接
func InitES() {
	// 从 application.yaml 中读取 ES 的地址 (比如 http://127.0.0.1:9200)
	cfg := elasticsearch.Config{
		Addresses: config.Global.Elasticsearch.Addresses,
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("❌ Elasticsearch 客户端创建失败: %v", err)
	}

	// Ping 一下 ES 看看是否真正连通
	res, err := client.Info()
	if err != nil {
		log.Fatalf("❌ Elasticsearch 连接失败，请检查 Docker 容器是否启动: %v", err)
	}
	defer res.Body.Close()

	log.Println("✅ Elasticsearch 知识库引擎连接成功！")
	ESClient = client
}

// SearchKnowledge 真实混合检索：去 ES 中进行 kNN 搜索并解析结果
func SearchKnowledge(ctx context.Context, queryVector []float32, topK int) ([]string, error) {
	// 1. 构建 ES 8.x 的 kNN 搜索 DSL
	query := map[string]interface{}{
		"knn": map[string]interface{}{
			"field":          "vector", // 必须与建表时的 dense_vector 字段名一致
			"query_vector":   queryVector,
			"k":              topK,
			"num_candidates": 100,
		},
		"_source": []string{"content", "metadata"}, // 我们只需要文本内容
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("构建查询 DSL 失败: %v", err)
	}

	// 2. 发起真实请求
	res, err := ESClient.Search(
		ESClient.Search.WithContext(ctx),
		ESClient.Search.WithIndex("sec_knowledge_base"), // 确保你的索引名是这个
		ESClient.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, fmt.Errorf("ES 查询执行失败: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("ES 返回错误状态: %s", res.String())
	}

	// 3. 解析 ES 复杂的嵌套 JSON 响应
	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("解析 ES 响应失败: %v", err)
	}

	var docs []string
	// 提取 hits.hits 数组
	if hitsObj, ok := r["hits"].(map[string]interface{}); ok {
		if hitsArr, ok := hitsObj["hits"].([]interface{}); ok {
			for _, hit := range hitsArr {
				source := hit.(map[string]interface{})["_source"].(map[string]interface{})
				if content, exists := source["content"].(string); exists {
					docs = append(docs, content)
				}
			}
		}
	}

	// 4. 如果没查到，兜底返回
	if len(docs) == 0 {
		return []string{"知识库中未检索到相关安全规范。"}, nil
	}

	return docs, nil
}
