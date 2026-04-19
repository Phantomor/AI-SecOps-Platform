// cmd/server/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// 注意：这里的 "sentinel-agent-go" 需要和你 go.mod 里的 module 名称保持一致
	_ "sentinel-agent-go/internal/agent/plugins"
	"sentinel-agent-go/internal/config"
	"sentinel-agent-go/internal/controller"
	"sentinel-agent-go/internal/mcp"
	"sentinel-agent-go/internal/repository"
	"sentinel-agent-go/internal/worker"

	_ "sentinel-agent-go/docs"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-gonic/gin"
)

// CorsMiddleware 企业级动态跨域中间件
func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 动态获取前端的 Origin，而不是写死 "*"
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		// 拦截浏览器的 OPTIONS 预检请求，直接放行
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// @title           ZeroTrust Sentinel API
// @version         1.0
// @description     基于 Go + MCP 协议的企业级零信任安全智能体平台 (AI SecOps)
// @termsOfService  http://swagger.io/terms/

// @contact.name   Phantomor
// @contact.url    https://github.com/Phantomor

// @host      localhost:8080
// @BasePath  /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// 1. 加载配置
	// 注意：我们在项目根目录执行 go run 时，相对路径是 configs/application.yaml
	config.LoadConfig("configs/application.yaml")

	// === 初始化 Redis, Elasticsearch ===
	repository.InitMySQL()
	repository.InitRedis()
	repository.InitES()
	repository.InitMinIO()
	repository.InitKafkaProducer()

	// 启动后台消费者监听协程
	worker.StartDocParserWorker()
	// ：启动 MCP Client，疯狂吸收外部工具！
	mcp.StartMCPClient()

	// 2. 设置 Gin 运行模式 (从配置读取 debug 或 release)
	gin.SetMode(config.Global.Server.Mode)

	// 3. 初始化 Gin 引擎
	// Default() 会默认包含 Logger 和 Recovery 中间件，保证程序崩溃时能自动恢复并记录日志
	r := gin.Default()

	// 注册 MCP Server
	mcp.RegisterMCPServer()
	// 注册 Swagger 路由
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Use(CorsMiddleware()) // 应用跨域中间件

	// 4. 注册健康检查路由 (探针)
	// 这个接口后续会被 Docker/K8s 用来检测你的服务是否存活
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "pong",
			"data":    "GoAI 面试官核心网关已成功启动！",
		})
	})

	// 注册用户模块路由
	userCtrl := controller.NewUserController()
	userCtrl.RegisterRoutes(r)

	// TODO: 预留位置，后续在这里注册大模型聊天、用户登录等路由
	// registerRoutes(r)
	aiCtrl := controller.NewAIController()
	aiCtrl.RegisterRoutes(r)

	// 5. 启动服务器
	addr := fmt.Sprintf(":%d", config.Global.Server.Port)

	// 1. 构造 http.Server 实例
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// 2. 开启一个 goroutine 启动服务，避免阻塞主线程
	go func() {
		log.Printf("🚀 Starting server on http://127.0.0.1%s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 3. 等待中断信号以优雅地关闭服务器
	// 监听 SIGINT (Ctrl+C) 和 SIGTERM (Docker/K8s 停止容器时的默认信号)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞，直到接收到信号
	<-quit
	log.Println("⚠️ 接收到停机信号，准备关闭服务...")

	// 4. 创建一个 5 秒超时的上下文
	// 这给正在处理的请求（比如还没发完的 SSE 流）5秒钟的时间完成收尾
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 5. 优雅关闭
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("❌ 服务器强制关闭:", err)
	}

	// 可以在这里补充清理逻辑，例如关闭数据库连接、清理缓存等
	// sqlDB, _ := repository.DB.DB()
	// sqlDB.Close()

	log.Println("✅ GoAI 面试官服务已平滑退出")
}
