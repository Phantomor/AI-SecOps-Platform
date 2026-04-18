// internal/repository/redis.go
package repository

import (
	"context"
	"log"

	"sentinel-agent-go/internal/config"

	"github.com/redis/go-redis/v9"
)

// RedisClient 全局 Redis 客户端实例
var RedisClient *redis.Client

// InitRedis 初始化 Redis 连接
func InitRedis() {
	cfg := config.Global.Redis

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 测试连接
	_, err := RedisClient.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("❌ Redis 连接失败: %v", err)
	}

	log.Println("✅ Redis 缓存中枢连接成功！")
}
