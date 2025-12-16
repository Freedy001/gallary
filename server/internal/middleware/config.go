package middleware

import (
	"gallary/server/config"
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
	mu       sync.RWMutex
	basePath string
}

// NewDynamicStaticConfig 创建动态静态文件配置
func NewDynamicStaticConfig() *DynamicStaticConfig {
	return &DynamicStaticConfig{}
}

// Update 更新配置
func (c *DynamicStaticConfig) Update(basePath string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.basePath = basePath
}
