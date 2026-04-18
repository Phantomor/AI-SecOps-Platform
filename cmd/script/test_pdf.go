// cmd/script/test_pdf.go
package main

import (
	"log"

	"sentinel-agent-go/internal/tools" // 确保这里的包名和你的项目 go.mod 一致
)

func main() {
	log.Println("🚀 开始测试 PDF 生成工具...")

	// 模拟大模型提取出来的参数（注意：暂时使用纯英文防止乱码）
	candidateName := "John_Doe"
	comments := "The candidate demonstrated a solid understanding of Go concurrency, including the G-P-M model. Strong communication skills. Recommended for the Senior Backend Engineer position."

	// 真正调用我们的工具类
	filePath, err := tools.GenerateInterviewPDF(candidateName, comments)

	if err != nil {
		log.Fatalf("❌ 工具调用失败: %v", err)
	}

	log.Printf("🎉 测试大成功！请在项目根目录下查看文件: %s", filePath)
}
