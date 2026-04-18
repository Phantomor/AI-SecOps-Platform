// internal/repository/kafka_producer.go
package repository

import (
	"context"
	"log"
	"time"

	"sentinel-agent-go/internal/config"

	"github.com/segmentio/kafka-go"
)

var KafkaWriter *kafka.Writer

func InitKafkaProducer() {
	cfg := config.Global.Kafka

	// 初始化 Writer，配置重试机制和异步批量发送策略
	KafkaWriter = &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topics["document_parse"], // 对应 yaml 中的 doc-parse-topic
		Balancer:     &kafka.LeastBytes{},          // 负载均衡策略
		BatchTimeout: 10 * time.Millisecond,
	}
	log.Println("✅ Kafka 生产者初始化成功！")
}

// SendDocParseTask 发送文档解析任务
func SendDocParseTask(ctx context.Context, key string, payload []byte) error {
	msg := kafka.Message{
		Key:   []byte(key),
		Value: payload,
	}
	return KafkaWriter.WriteMessages(ctx, msg)
}
