package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"gallary/server/config"
)

// CORSMiddleware CORS中间件
func CORSMiddleware(cfg *config.CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置CORS头
		if len(cfg.AllowOrigins) > 0 {
			origin := c.GetHeader("Origin")
			for _, allowOrigin := range cfg.AllowOrigins {
				if allowOrigin == "*" || allowOrigin == origin {
					c.Header("Access-Control-Allow-Origin", allowOrigin)
					break
				}
			}
		}

		if len(cfg.AllowMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", strings.Join(cfg.AllowMethods, ", "))
		}

		if len(cfg.AllowHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", strings.Join(cfg.AllowHeaders, ", "))
		}

		if len(cfg.ExposeHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", strings.Join(cfg.ExposeHeaders, ", "))
		}

		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if cfg.MaxAge > 0 {
			c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", cfg.MaxAge))
		}

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
