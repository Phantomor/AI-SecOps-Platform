// internal/repository/minio.go
package repository

import (
	"context"
	"io"
	"log"

	"sentinel-agent-go/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var MinioClient *minio.Client

func InitMinIO() {
	cfg := config.Global.Minio
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		log.Fatalf("❌ MinIO 客户端初始化失败: %v", err)
	}

	// 检查 Bucket 是否存在，不存在则创建
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.BucketName)
	if err == nil && !exists {
		err = client.MakeBucket(ctx, cfg.BucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Fatalf("❌ 创建 MinIO Bucket 失败: %v", err)
		}
	}

	MinioClient = client
	log.Println("✅ MinIO 对象存储连接成功！")
}

// UploadFile 上传文件到 MinIO
func UploadFile(ctx context.Context, objectName string, reader io.Reader, objectSize int64, contentType string) error {
	_, err := MinioClient.PutObject(ctx, config.Global.Minio.BucketName, objectName, reader, objectSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}
