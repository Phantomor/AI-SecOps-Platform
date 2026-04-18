// internal/worker/doc_parser.go
package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"sentinel-agent-go/internal/config"

	"github.com/segmentio/kafka-go"
)

// DocParseMessage 约定 Kafka 消息体的 JSON 结构
type DocParseMessage struct {
	UserID     uint   `json:"user_id"`
	ObjectName string `json:"object_name"`
	Status     string `json:"status"`
}

func StartDocParserWorker() {
	cfg := config.Global.Kafka
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          cfg.Topics["document_parse"],
		GroupID:        "goai-doc-parser-group", // 消费者组，保证一条消息只被消费一次
		CommitInterval: time.Second,
	})

	log.Println("🚀 后台 Worker [文档解析] 已启动，正在监听 Kafka...")

	go func() {
		defer reader.Close()
		for {
			msg, err := reader.ReadMessage(context.Background())
			if err != nil {
				log.Printf("⚠️ 读取 Kafka 消息失败: %v", err)
				continue
			}

			var task DocParseMessage
			if err := json.Unmarshal(msg.Value, &task); err != nil {
				log.Printf("解析消息体失败: %v", err)
				continue
			}

			// ==========================================
			// 核心业务流：
			// 1. 根据 ObjectName 去 MinIO 下载该 PDF 文件
			// 2. 提取 PDF 文本内容
			// 3. 调用 Embedder 将文本向量化
			// 4. 将向量存入 ES
			// 5. 更新 MySQL 中该记录的状态为 "解析完成"
			// ==========================================
			log.Printf("✅ [Worker 接收任务] 开始处理用户 [%d] 的简历: %s", task.UserID, task.ObjectName)

			// 模拟耗时操作 (后续接入真实的 RAG 逻辑)
			time.Sleep(2 * time.Second)
			log.Printf("🎉 [Worker 完成任务] 简历 [%s] 向量化并入库成功！", task.ObjectName)
		}
	}()
}
