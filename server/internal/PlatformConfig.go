package internal

import (
	"gallary/server/internal/middleware"
	"gallary/server/internal/model"
)

type PlatformConfig struct {
	*middleware.AdminConfig
	*middleware.DynamicStaticConfig
	*model.CleanupPO
}
