// internal/config/config.go
package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// AppConfig 全局配置结构体
type AppConfig struct {
	Server        ServerConfig        `yaml:"server"`
	MySQL         MySQLConfig         `yaml:"mysql"`
	Redis         RedisConfig         `yaml:"redis"`
	Minio         MinioConfig         `yaml:"minio"`
	Elasticsearch ElasticsearchConfig `yaml:"elasticsearch"`
	Kafka         KafkaConfig         `yaml:"kafka"`
	AI            AIConfig            `yaml:"ai"`
}

type ServerConfig struct {
	Port      int    `yaml:"port"`
	Mode      string `yaml:"mode"`
	JwtSecret string `yaml:"jwt_secret"`
}

type MySQLConfig struct {
	DSN          string `yaml:"dsn"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
	MaxOpenConns int    `yaml:"max_open_conns"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type MinioConfig struct {
	Endpoint   string `yaml:"endpoint"`
	AccessKey  string `yaml:"access_key"`
	SecretKey  string `yaml:"secret_key"`
	UseSSL     bool   `yaml:"use_ssl"`
	BucketName string `yaml:"bucket_name"`
}

type ElasticsearchConfig struct {
	Addresses []string `yaml:"addresses"`
	IndexName string   `yaml:"index_name"`
}

type KafkaConfig struct {
	Brokers []string          `yaml:"brokers"`
	Topics  map[string]string `yaml:"topics"`
}

type AIConfig struct {
	OpenAI struct {
		APIKey  string `yaml:"api_key"`
		BaseURL string `yaml:"base_url"`
		Model   string `yaml:"model"`
	} `yaml:"openai"`
}

// Global 暴露在外的全局配置实例
var Global *AppConfig

// LoadConfig 从指定路径加载 YAML 文件
func LoadConfig(path string) {
	file, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}

	Global = &AppConfig{}
	err = yaml.Unmarshal(file, Global)
	if err != nil {
		log.Fatalf("解析配置文件失败: %v", err)
	}

	log.Println("✅ 配置文件加载成功！")
}
