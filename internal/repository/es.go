// internal/repository/es.go
package repository

import (
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
