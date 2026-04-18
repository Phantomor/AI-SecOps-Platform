// internal/pkg/response.go
package pkg

import "github.com/gin-gonic/gin"

// 统一的返回结构
func Success(c *gin.Context, data interface{}) {
	c.JSON(200, gin.H{
		"code":    200, // 前端拦截器认准的成功状态码
		"data":    data,
		"message": "success",
	})
}

func Error(c *gin.Context, code int, msg string) {
	c.JSON(200, gin.H{
		"code":    code,
		"data":    nil,
		"message": msg,
	})
}

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"` // 这个字段名或 JSON tag 必须叫 data
}
