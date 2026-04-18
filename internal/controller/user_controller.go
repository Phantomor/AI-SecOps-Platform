// internal/controller/user_controller.go
package controller

import (
	"errors"
	"sentinel-agent-go/internal/middleware"
	"sentinel-agent-go/internal/model"
	"sentinel-agent-go/internal/pkg"
	"sentinel-agent-go/internal/repository"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserController struct{}

func NewUserController() *UserController {
	return &UserController{}
}

func (uc *UserController) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/user")
	{
		api.POST("/login", uc.Login)
		api.POST("/register", uc.Register)

		// 需要登录后才能访问的接口
		auth := api.Group("")
		auth.Use(middleware.AuthMiddleware()) // 挂载中间件
		{
			auth.GET("/get/login", uc.GetLoginUser)
			auth.POST("/logout", uc.Logout)
		}
	}
}

// Login 用户登录
// @Summary      用户登录接口
// @Description  通过账号密码登录并返回 JWT Token
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        request body     dto.LoginRequest  true  "登录参数"
// @Success      200  {object}  pkg.Response{data=dto.LoginResponse} "成功"
// @Failure      401  {object}  pkg.Response "账号或密码错误"
// @Router       /user/login [post]
func (uc *UserController) Login(c *gin.Context) {
	var req struct {
		UserAccount  string `json:"userAccount" binding:"required"`
		UserPassword string `json:"userPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.Error(c, 400, "参数错误")
		return
	}

	// 1. 查库找人
	var user model.User
	if err := repository.DB.Where("user_account = ?", req.UserAccount).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			pkg.Error(c, 401, "账号或密码错误") // 防御性提示，不暴露具体是账号错还是密码错
			return
		}
		pkg.Error(c, 500, "数据库查询异常")
		return
	}

	// 2. 校验密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.UserPassword), []byte(req.UserPassword)); err != nil {
		pkg.Error(c, 401, "账号或密码错误")
		return
	}

	// 3. 登录成功，下发 Token 或 Cookie (此处暂时保留你的 Cookie 逻辑，后续可升级为 JWT)
	// 将用户的自增 ID 转为字符串存入 cookie
	// import "strconv" -> strconv.Itoa(int(user.ID))
	// c.SetCookie("login_token", string(rune(user.ID)), 86400, "/", "", false, true)

	token, err := pkg.GenerateToken(user.ID, user.UserAccount, user.UserRole)
	if err != nil {
		pkg.Error(c, 500, "生成 Token 失败")
		return
	}
	c.Header("Authorization", "Bearer "+token)

	pkg.Success(c, user) // 返回用户信息（密码字段因为加了 json:"-" 会自动隐藏）
}

// GetLoginUser 改为从上下文获取
// @Summary 获取当前登录用户信息
// @Description 从上下文中获取当前登录用户的 ID 和账号信息（需要登录凭证）
// @Tags 用户模块 (User)
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "{"code": 0, "data": {"id": 1, "userAccount": "xxx"}}"
// @Router /user/get/login [get]
// @Security ApiKeyAuth
func (uc *UserController) GetLoginUser(c *gin.Context) {
	userID, _ := c.Get("currentUserID")
	userAccount, _ := c.Get("currentUserAccount")

	pkg.Success(c, gin.H{
		"id":          userID,
		"userAccount": userAccount,
	})
}

// Register 用户注册
// @Summary 用户注册
// @Description 注册新用户，账号至少4位，密码至少6位。密码将在入库前通过 Bcrypt 加密。
// @Tags 用户模块 (User)
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "用户注册参数"
// @Success 200 {object} map[string]interface{} "{"code": 0, "data": "注册成功"}"
// @Failure 400 {object} map[string]interface{} "{"code": 400, "message": "账号已存在或参数校验失败"}"
// @Failure 500 {object} map[string]interface{} "{"code": 500, "message": "服务端异常(如加密失败或数据库写入失败)"}"
// @Router /user/register [post]
func (uc *UserController) Register(c *gin.Context) {
	var req struct {
		UserAccount  string `json:"userAccount" binding:"required,min=4"`
		UserPassword string `json:"userPassword" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.Error(c, 400, "账号至少4位，密码至少6位")
		return
	}

	// 1. 检查账号是否已存在
	var count int64
	repository.DB.Model(&model.User{}).Where("user_account = ?", req.UserAccount).Count(&count)
	if count > 0 {
		pkg.Error(c, 400, "账号已存在")
		return
	}

	// 2. 密码加密 (Bcrypt)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.UserPassword), bcrypt.DefaultCost)
	if err != nil {
		pkg.Error(c, 500, "密码加密失败")
		return
	}

	// 3. 存入数据库
	newUser := model.User{
		UserAccount:  req.UserAccount,
		UserPassword: string(hashedPassword),
		UserName:     "用户_" + req.UserAccount,
		UserRole:     "user",
	}

	if err := repository.DB.Create(&newUser).Error; err != nil {
		pkg.Error(c, 500, "注册失败: "+err.Error())
		return
	}

	pkg.Success(c, "注册成功")
}

// @Summary 用户退出登录
// @Description 清除客户端的 login_token Cookie
// @Tags 用户模块 (User)
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "{"code": 0, "data": "退出成功"}"
// @Router /api/user/logout [post]
// @Security ApiKeyAuth
func (uc *UserController) Logout(c *gin.Context) {
	// 清除 Cookie
	c.SetCookie("login_token", "", -1, "/", "", false, true)
	pkg.Success(c, "退出成功")
}
