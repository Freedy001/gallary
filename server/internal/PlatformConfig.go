package internal

import (
	"gallary/server/internal/middleware"
	"gallary/server/internal/model"
)

var PlatConfig = &PlatformConfig{}

type PlatformConfig struct {
	*middleware.AdminConfig
	*middleware.DynamicStaticConfig
	*model.CleanupPO
	*model.AIPo
}
