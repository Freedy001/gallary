package router

import (
	"github.com/gin-gonic/gin"

	"gallary/server/config"
	"gallary/server/internal/handler"
	"gallary/server/internal/middleware"
)

// SetupRouter 配置路由
func SetupRouter(
	cfg *config.Config,
	authHandler *handler.AuthHandler,
	imageHandler *handler.ImageHandler,
	shareHandler *handler.ShareHandler,
	settingHandler *handler.SettingHandler,
	storageHandler *handler.StorageHandler,
	migrationHandler *handler.MigrationHandler,
	dynamicStaticConfig *middleware.DynamicStaticConfig,
) *gin.Engine {
	// 设置运行模式
	gin.SetMode(cfg.Server.Mode)

	r := gin.New()

	// 使用中间件
	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware(&cfg.CORS))

	// 动态静态文件中间件
	r.Use(middleware.DynamicStaticMiddleware(dynamicStaticConfig))

	// API路由组
	api := r.Group("/api")
	{
		// 认证相关路由（无需认证）
		auth := api.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.GET("/check", authHandler.Check)
		}

		// 图片相关路由（需要认证）
		images := api.Group("/images")
		images.Use(middleware.AuthMiddleware(cfg))
		{
			images.POST("/upload", imageHandler.Upload)
			images.POST("/batch-delete", imageHandler.BatchDelete)
			images.POST("/batch-download", imageHandler.BatchDownload)
			images.PUT("/metadata", imageHandler.BatchUpdateMetadata)
			images.GET("", imageHandler.List)
			images.GET("/clusters", imageHandler.GetClusters)
			images.GET("/clusters/images", imageHandler.GetClusterImages)
			images.GET("/geo-bounds", imageHandler.GetGeoBounds)
			images.GET("/:id", imageHandler.GetByID)
			images.DELETE("/:id", imageHandler.Delete)
			images.GET("/:id/download", imageHandler.Download)
			images.GET("/:id/file", imageHandler.ProxyFile)

			// 回收站相关路由
			images.GET("/trash", imageHandler.ListDeleted)
			images.POST("/trash/restore", imageHandler.RestoreImages)
			images.POST("/trash/delete", imageHandler.PermanentlyDelete)
		}

		// 搜索路由（需要认证）
		api.GET("/search", middleware.AuthMiddleware(cfg), imageHandler.Search)

		// 分享管理路由（需要认证）
		shares := api.Group("/shares")
		shares.Use(middleware.AuthMiddleware(cfg))
		{
			shares.POST("", shareHandler.Create)
			shares.GET("", shareHandler.List)
			shares.DELETE("/:id", shareHandler.Delete)
		}

		// 公开分享访问路由（无需认证，或单独认证）
		publicShares := api.Group("/s")
		{
			publicShares.GET("/:code/info", shareHandler.GetPublicInfo)
			publicShares.POST("/:code/images", shareHandler.SharedImages)
		}

		// 设置路由（无需认证）
		settings := api.Group("/settings")
		{
			settings.GET("/:category", settingHandler.GetByCategory)
			settings.GET("/password/status", settingHandler.GetPasswordStatus)
			settings.PUT("/password", settingHandler.UpdatePassword)
			settings.PUT("/storage", settingHandler.UpdateStorage)
			settings.PUT("/cleanup", settingHandler.UpdateCleanup)
		}

		// 存储统计路由（需要认证）
		storageGroup := api.Group("/storage")
		storageGroup.Use(middleware.AuthMiddleware(cfg))
		{
			storageGroup.GET("/stats", storageHandler.GetStats)

			// 阿里云盘登录相关路由
			aliyunpan := storageGroup.Group("/aliyunpan")
			{
				aliyunpan.POST("/qrcode", storageHandler.GenerateAliyunPanQRCode)
				aliyunpan.GET("/qrcode/status", storageHandler.CheckAliyunPanQRCodeStatus)
				aliyunpan.POST("/logout", storageHandler.LogoutAliyunPan)
			}

			// 迁移相关路由
			migration := storageGroup.Group("/migration")
			{
				migration.GET("/active", migrationHandler.GetActive)
				migration.GET("/:id", migrationHandler.GetByID)
			}
		}
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	return r
}
