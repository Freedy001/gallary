package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"gallary/server/config"
	"gallary/server/internal/utils"
)

// AuthMiddleware JWT认证中间件
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果没有设置管理员密码，则不需要认证
		if !cfg.Admin.IsAuthEnabled() {
			c.Next()
			return
		}

		var tokenString string

		// 1. 优先从 Authorization 头获取
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		// 2. 如果头部没有，尝试从表单参数获取（支持表单下载等场景）
		if tokenString == "" {
			tokenString = c.PostForm("token")
		}

		// 3. 如果还没有，尝试从 URL 查询参数获取
		if tokenString == "" {
			tokenString = c.Query("token")
		}

		if tokenString == "" {
			utils.Unauthorized(c, "未提供认证token")
			c.Abort()
			return
		}

		// 验证token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWT.Secret), nil
		})

		if err != nil || !token.Valid {
			utils.Unauthorized(c, "无效的token")
			c.Abort()
			return
		}

		// 验证密码版本号
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			// 获取token中的密码版本号
			tokenPV := int64(0)
			if pv, exists := claims["pv"]; exists {
				switch v := pv.(type) {
				case float64:
					tokenPV = int64(v)
				case int64:
					tokenPV = v
				}
			}

			// 比较密码版本号，如果token的版本号小于当前版本号，则token已失效
			if tokenPV < cfg.Admin.PasswordVersion {
				utils.Unauthorized(c, "密码已更改，请重新登录")
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
