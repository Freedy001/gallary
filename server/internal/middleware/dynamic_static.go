package middleware

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// DynamicStaticMiddleware 动态静态文件中间件
func DynamicStaticMiddleware(config *DynamicStaticConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		config.mu.RLock()
		basePath := config.basePath
		config.mu.RUnlock()

		requestPath := c.Request.URL.Path

		// 检查请求路径是否匹配 URL 前缀
		if !strings.HasPrefix(requestPath, "/static/") && requestPath != "/static/" {
			c.Next()
			return
		}

		// 提取相对路径
		relativePath := strings.TrimPrefix(requestPath, "/static")
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

		// 检查文件是否存在
		info, err := os.Stat(filePath)
		if err != nil || info.IsDir() {
			c.Next()
			return
		}

		// 设置缓存头 - 图片文件使用长期缓存
		c.Header("Cache-Control", "public, max-age=31536000, immutable")

		// 使用 http.ServeFile 提供文件
		http.ServeFile(c.Writer, c.Request, filePath)
		c.Abort()
	}
}
