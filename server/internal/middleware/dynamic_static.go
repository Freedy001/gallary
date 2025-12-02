package middleware

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// DynamicStaticMiddleware 动态静态文件中间件
func DynamicStaticMiddleware(config *DynamicStaticConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		config.mu.RLock()
		enabled := config.enabled
		urlPrefix := config.urlPrefix
		basePath := config.basePath
		config.mu.RUnlock()

		// 未启用则跳过
		if !enabled {
			c.Next()
			return
		}

		requestPath := c.Request.URL.Path

		// 检查请求路径是否匹配 URL 前缀
		if !strings.HasPrefix(requestPath, urlPrefix+"/") && requestPath != urlPrefix {
			c.Next()
			return
		}

		// 提取相对路径
		relativePath := strings.TrimPrefix(requestPath, urlPrefix)
		if relativePath == "" {
			relativePath = "/"
		}

		// 清理路径防止目录遍历攻击
		relativePath = filepath.Clean(relativePath)
		if strings.HasPrefix(relativePath, "..") {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		// 构建完整文件路径
		filePath := filepath.Join(basePath, relativePath)

		// 使用 http.ServeFile 提供文件
		http.ServeFile(c.Writer, c.Request, filePath)
		c.Abort()
	}
}
