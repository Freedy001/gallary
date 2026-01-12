package router

import (
	"gallary/server/internal"

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
	albumHandler *handler.AlbumHandler,
	settingHandler *handler.SettingHandler,
	storageHandler *handler.StorageHandler,
	migrationHandler *handler.MigrationHandler,
	aiHandler *handler.AIHandler,
	wsHandler *handler.WebSocketHandler,
) *gin.Engine {
	configCompose := internal.PlatConfig
	// 设置运行模式
	gin.SetMode(cfg.Server.Mode)

	r := gin.New()

	// 使用中间件
	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware(&cfg.CORS))

	// 动态静态文件中间件
	r.Use(middleware.DynamicStaticMiddleware(configCompose.DynamicStaticConfig))
	r.GET("/resouse/:hash/file", imageHandler.ProxyFile)

	// API路由组
	api := r.Group("/api")
	{
		// WebSocket 连接（在路由组内，但独立路由）
		api.GET("/ws", wsHandler.HandleWebSocket)

		// 认证相关路由（无需认证）
		auth := api.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.GET("/check", authHandler.Check)
		}

		// 公开分享访问路由（无需认证，或单独认证）
		publicShares := api.Group("/s")
		{
			publicShares.GET("/:code/info", shareHandler.GetPublicInfo)
			publicShares.POST("/:code/images", shareHandler.SharedImages)
		}

		// 图片相关路由（需要认证）
		images := api.Group("/images")
		images.Use(middleware.AuthMiddleware(configCompose.AdminConfig))
		{
			images.POST("/upload", imageHandler.Upload)
			// 新上传流程
			images.POST("/prepare-upload", imageHandler.PrepareUpload)
			images.POST("/confirm-upload", imageHandler.ConfirmUpload)
			images.PUT("/upload-direct/:uploadId", imageHandler.UploadDirect)
			images.PUT("/upload-thumbnail/:uploadId", imageHandler.UploadThumbnail)
			images.POST("/batch-delete", imageHandler.BatchDelete)
			images.POST("/batch-download", imageHandler.BatchDownload)
			images.POST("/batch", imageHandler.GetByIDs)
			images.PUT("/metadata", imageHandler.BatchUpdateMetadata)
			images.GET("", imageHandler.List)
			images.GET("/clusters", imageHandler.GetClusters)
			images.GET("/clusters/images", imageHandler.GetClusterImages)
			images.GET("/geo-bounds", imageHandler.GetGeoBounds)
			images.GET("/:id", imageHandler.GetByID)
			images.DELETE("/:id", imageHandler.Delete)
			images.GET("/:id/download", imageHandler.Download)
			// 回收站相关路由
			images.GET("/trash", imageHandler.ListDeleted)
			images.POST("/trash/restore", imageHandler.RestoreImages)
			images.POST("/trash/delete", imageHandler.PermanentlyDelete)
		}

		// 搜索路由（需要认证）
		api.POST("/search", middleware.AuthMiddleware(configCompose.AdminConfig), imageHandler.Search)

		// 标签路由（需要认证）
		api.GET("/tags", middleware.AuthMiddleware(configCompose.AdminConfig), imageHandler.GetTags)

		// 分享管理路由（需要认证）
		shares := api.Group("/shares")
		shares.Use(middleware.AuthMiddleware(configCompose.AdminConfig))
		{
			shares.POST("", shareHandler.Create)
			shares.GET("", shareHandler.List)
			shares.PUT("/:id", shareHandler.Update)
			shares.DELETE("/:id", shareHandler.Delete)
		}

		// 相册管理路由（需要认证）
		albums := api.Group("/albums")
		albums.Use(middleware.AuthMiddleware(configCompose.AdminConfig))
		{
			albums.GET("", albumHandler.List)
			albums.POST("", albumHandler.Create)
			albums.POST("/batch-delete", albumHandler.BatchDelete)
			albums.POST("/batch-copy", albumHandler.BatchCopy)
			albums.POST("/batch-merge", albumHandler.BatchMerge)
			albums.POST("/batch-get", albumHandler.BatchGet)
			albums.POST("/ai-naming", albumHandler.AINaming)
			albums.PUT("/:id", albumHandler.Update)
			albums.GET("/:id/images", albumHandler.GetImages)
			albums.POST("/:id/images", albumHandler.AddImages)
			albums.DELETE("/:id/images", albumHandler.RemoveImages)
			albums.PUT("/:id/cover", albumHandler.SetCover)
			albums.DELETE("/:id/cover", albumHandler.RemoveCover)
			albums.PUT("/:id/cover/average", albumHandler.SetAverageCover)
		}

		// 设置路由（无需认证）
		settings := api.Group("/settings")
		settings.Use(middleware.AuthMiddleware(configCompose.AdminConfig))
		{
			settings.GET("/:category", settingHandler.GetByCategory)
			settings.GET("/password/status", settingHandler.GetPasswordStatus)
			settings.PUT("/password", settingHandler.UpdatePassword)
			settings.PUT("/cleanup", settingHandler.UpdateCleanup)
			settings.PUT("/ai", settingHandler.UpdateAI)
			settings.GET("/configed-default-model/:type", settingHandler.HasConfigDefaultModel)

			// 存储配置 CRUD
			settings.POST("/storage", settingHandler.AddStorage)                                 // 添加存储配置
			settings.POST("/storage/s3/test", settingHandler.TestS3Connection)                   // 测试 S3 连接
			settings.PUT("/storage/default", settingHandler.SetDefaultStorage)                   // 设置默认存储（必须在 :storageId 之前）
			settings.PUT("/storage/alyunpan/global", settingHandler.UpdateAliyunpanGlobalConfig) // 更新全局配置（必须在 :storageId 之前）
			settings.PUT("/storage/:storageId", settingHandler.UpdateStorage)                    // 修改存储配置
			settings.DELETE("/storage/:storageId", settingHandler.DeleteStorage)                 // 删除存储配置
		}

		// 存储统计路由（需要认证）
		storageGroup := api.Group("/storage")
		storageGroup.Use(middleware.AuthMiddleware(configCompose.AdminConfig))
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

		// AI 相关路由（需要认证）
		ai := api.Group("/ai")
		ai.Use(middleware.AuthMiddleware(configCompose.AdminConfig))
		{
			// AI 连接测试（使用临时配置）
			ai.POST("/test-connection", aiHandler.TestConnection)

			// 获取可用的嵌入模型列表
			ai.GET("/embedding-models", aiHandler.GetEmbeddingModels)

			// 提示词优化
			ai.POST("/optimize-prompt", aiHandler.OptimizePrompt)

			// AI 队列管理
			ai.GET("/queues", aiHandler.GetQueueStatus)                   // 获取所有队列状态
			ai.GET("/queues/:id", aiHandler.GetQueueDetail)               // 获取队列详情
			ai.POST("/queues/:id/retry", aiHandler.RetryQueueFailedItems) // 重试队列所有失败项目

			// AI 任务项操作
			ai.POST("/task-items/:id/retry", aiHandler.RetryTaskItem)   // 重试单个任务项
			ai.POST("/task-items/:id/ignore", aiHandler.IgnoreTaskItem) // 忽略单个任务项
			// 智能相册异步任务路由
			ai.POST("/smart-albums-generate", aiHandler.GenerateSmartAlbums)
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
