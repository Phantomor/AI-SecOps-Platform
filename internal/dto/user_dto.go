package dto

// RegisterRequest 定义注册请求的参数结构体 (供 Swagger 解析和参数绑定)
type RegisterRequest struct {
	UserAccount  string `json:"userAccount" binding:"required,min=4" example:"testuser" validate:"required"`
	UserPassword string `json:"userPassword" binding:"required,min=6" example:"123456" validate:"required"`
}

// LoginRequest 用户登录请求参数
type LoginRequest struct {
	UserAccount  string `json:"userAccount" binding:"required" example:"admin"`
	UserPassword string `json:"userPassword" binding:"required" example:"123456"`
}

// LoginResponse 用户登录返回结果 (仅作 Swagger 文档展示用)
type LoginResponse struct {
	ID          uint   `json:"id" example:"1"`
	UserAccount string `json:"userAccount" example:"admin"`
	UserName    string `json:"userName" example:"管理员"`
	UserRole    string `json:"userRole" example:"admin"`
	// 注意：密码字段不要放在 Response 中，或者在数据库 Model 中加上 `json:"-"`
}
