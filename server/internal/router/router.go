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
) *gin.Engine {
	// 设置运行模式
	gin.SetMode(cfg.Server.Mode)

	r := gin.New()

	// 使用中间件
	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware(&cfg.CORS))

	// 静态文件服务（用于访问本地存储的图片）
	if cfg.Storage.Default == "local" {
		r.Static(cfg.Storage.Local.URLPrefix, cfg.Storage.Local.BasePath)
	}

	// API路由组
	api := r.Group("/api")
	{
		// 认证相关路由（无需认证）
		auth := api.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.GET("/check", middleware.AuthMiddleware(cfg), authHandler.Check)
		}

		// 图片相关路由（需要认证）
		images := api.Group("/images")
		images.Use(middleware.AuthMiddleware(cfg))
		{
			images.POST("/upload", imageHandler.Upload)
			images.POST("/batch-delete", imageHandler.BatchDelete)
			images.PUT("/metadata", imageHandler.BatchUpdateMetadata)
			images.GET("", imageHandler.List)
			images.GET("/clusters", imageHandler.GetClusters)
			images.GET("/clusters/images", imageHandler.GetClusterImages)
			images.GET("/:id", imageHandler.GetByID)
			images.DELETE("/:id", imageHandler.Delete)
			images.GET("/:id/download", imageHandler.Download)
		}

		// 搜索路由（需要认证）
		api.GET("/search", middleware.AuthMiddleware(cfg), imageHandler.Search)
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	return r
}
