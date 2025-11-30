package handler

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"gallary/server/config"
	"gallary/server/internal/utils"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	cfg *config.Config
}

// NewAuthHandler 创建认证处理器实例
func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应数据
type LoginResponse struct {
	Token     string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ExpiresIn int    `json:"expires_in" example:"86400"`
}

// AuthCheckResponse 认证检查响应数据
type AuthCheckResponse struct {
	Authenticated bool `json:"authenticated" example:"true"`
}

// Login 管理员登录
//
//	@Summary		管理员登录
//	@Description	使用密码登录获取JWT token
//	@Tags			认证
//	@Accept			json
//	@Produce		json
//	@Param			request	body		LoginRequest						true	"登录信息"
//	@Success		200		{object}	utils.Response{data=LoginResponse}	"登录成功"
//	@Failure		400		{object}	utils.Response						"请求参数错误"
//	@Failure		401		{object}	utils.Response						"密码错误"
//	@Failure		500		{object}	utils.Response						"生成token失败"
//	@Router			/api/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	// 如果未设置密码，则不需要认证
	if !h.cfg.Admin.IsAuthEnabled() {
		utils.Error(c, 400, "系统未启用认证")
		return
	}

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求参数错误")
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(h.cfg.Admin.Password), []byte(req.Password)); err != nil {
		utils.Unauthorized(c, "密码错误")
		return
	}

	// 生成JWT token，包含密码版本号
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(h.cfg.JWT.GetExpireDuration()).Unix(),
		"iat": time.Now().Unix(),
		"pv":  h.cfg.Admin.PasswordVersion, // 密码版本号
	})

	tokenString, err := token.SignedString([]byte(h.cfg.JWT.Secret))
	if err != nil {
		utils.InternalServerError(c, "生成token失败")
		return
	}

	utils.Success(c, gin.H{
		"token":      tokenString,
		"expires_in": int(h.cfg.JWT.GetExpireDuration().Seconds()),
	})
}

// Check 检查认证状态
//
//	@Summary		检查认证状态
//	@Description	检查当前token是否有效
//	@Tags			认证
//	@Produce		json
//	@Success		200	{object}	utils.Response{data=AuthCheckResponse}	"认证状态"
//	@Router			/api/auth/check [get]
func (h *AuthHandler) Check(c *gin.Context) {
	utils.Success(c, gin.H{
		"authenticated": h.hasAuth(c),
	})
}

func (h *AuthHandler) hasAuth(c *gin.Context) bool {
	// 如果没有设置管理员密码，则不需要认证
	if !h.cfg.Admin.IsAuthEnabled() {
		return true
	}

	// 获取Authorization头
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return false
	}

	// 解析Bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return false
	}

	// 验证token
	token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
		return []byte(h.cfg.JWT.Secret), nil
	})

	if err != nil || !token.Valid {
		return false
	}

	return true
}
