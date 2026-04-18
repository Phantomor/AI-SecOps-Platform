// internal/repository/mysql.go
package repository

import (
	"log"
	"time"

	"sentinel-agent-go/internal/config"
	"sentinel-agent-go/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitMySQL() {
	cfg := config.Global.MySQL

	// 配置 GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 打印 SQL 语句，方便调试
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN), gormConfig)
	if err != nil {
		log.Fatalf("❌ MySQL 连接失败: %v", err)
	}

	// 获取底层的 sql.DB 对象以设置连接池
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("❌ 获取底层 SQL DB 失败: %v", err)
	}

	// 设置连接池参数，提升并发性能
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = db
	log.Println("✅ MySQL 数据库连接成功！")

	// 自动迁移表结构 (生产环境通常用 Flyway/Goose，项目初期用 AutoMigrate 很方便)
	err = DB.AutoMigrate(&model.User{}, &model.ChatLog{})
	if err != nil {
		log.Fatalf("❌ 表结构自动迁移失败: %v", err)
	}
}
