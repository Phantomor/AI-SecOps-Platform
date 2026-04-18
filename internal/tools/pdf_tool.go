// internal/tools/pdf_tool.go
package tools

import (
	"fmt"
	"log"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// GenerateInterviewPDF 是实际执行 PDF 生成的 Go 函数
// candidateName: 候选人姓名（用于文件名）
// comments: 面试评价内容
func GenerateInterviewPDF(candidateName, comments string) (string, error) {
	// 1. 初始化 A4 纸张的 PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// ⚠️ 避坑指南：gofpdf 默认不支持中文字体。
	// 在真实的生产环境中，你需要下载一个中文字体库（如 simhei.ttf）并使用 pdf.AddUTF8Font 引入。
	// 为了演示 MVP 流程，我们这里暂时使用系统默认字体，并用英文模板拼装内容。
	pdf.AddUTF8Font("SongTi", "", "assets/fonts/songti.ttf")
	pdf.SetFont("Arial", "B", 20)
	pdf.Cell(40, 10, "GoAI Interview Assessment Report")

	pdf.Ln(15)
	pdf.SetFont("SongTi", "", 12)

	// 2. 拼装报告内容
	content := fmt.Sprintf("Candidate Name: %s\nDate: %s\n\nDetailed Evaluation:\n%s",
		candidateName,
		time.Now().Format("2006-01-02 15:04:05"),
		comments,
	)

	// MultiCell 能够自动处理长文本换行
	pdf.MultiCell(0, 8, content, "", "", false)

	// 3. 保存到本地磁盘
	fileName := fmt.Sprintf("Interview_Report_%s_%d.pdf", candidateName, time.Now().Unix())
	err := pdf.OutputFileAndClose(fileName)
	if err != nil {
		log.Printf("❌ PDF 生成失败: %v", err)
		return "", err
	}

	log.Printf("✅ PDF 报告生成成功，已保存至: %s", fileName)
	// 返回文件的本地相对路径给大模型
	return fileName, nil
}
