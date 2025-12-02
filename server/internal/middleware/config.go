package middleware

import (
	"gallary/server/config"
	"strings"
	"sync"
)

type AdminConfig struct {
	config.JWTConfig
	Password        string
	PasswordVersion int64
}

// IsAuthEnabled 检查是否启用认证
func (c *AdminConfig) IsAuthEnabled() bool {
	return c.Password != ""
}

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
