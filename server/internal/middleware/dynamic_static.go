package middleware

import (
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// DynamicStaticConfig 动态静态文件配置
type DynamicStaticConfig struct {
	mu        sync.RWMutex
	urlPrefix string
	basePath  string
	enabled   bool
}

// NewDynamicStaticConfig 创建动态静态文件配置
func NewDynamicStaticConfig() *DynamicStaticConfig {
	return &DynamicStaticConfig{
		enabled: false,
	}
}

// Update 更新配置
func (c *DynamicStaticConfig) Update(urlPrefix, basePath string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 规范化 URL 前缀
	if !strings.HasPrefix(urlPrefix, "/") {
		urlPrefix = "/" + urlPrefix
	}
	urlPrefix = strings.TrimSuffix(urlPrefix, "/")

	c.urlPrefix = urlPrefix
	c.basePath = basePath
	c.enabled = urlPrefix != "" && basePath != ""
}

// Disable 禁用静态文件服务
func (c *DynamicStaticConfig) Disable() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = false
}

// Enable 启用静态文件服务
func (c *DynamicStaticConfig) Enable() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.urlPrefix != "" && c.basePath != "" {
		c.enabled = true
	}
}

// IsEnabled 检查是否启用
func (c *DynamicStaticConfig) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

// GetConfig 获取当前配置
func (c *DynamicStaticConfig) GetConfig() (urlPrefix, basePath string, enabled bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.urlPrefix, c.basePath, c.enabled
}

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
