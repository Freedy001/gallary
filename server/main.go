package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"gallary/server/config"
	"gallary/server/internal/handler"
	"gallary/server/internal/repository"
	"gallary/server/internal/router"
	"gallary/server/internal/service"
	"gallary/server/internal/storage"
	"gallary/server/pkg/database"
	"gallary/server/pkg/logger"
)

func main() {
	// 1. 加载配置
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 2. 初始化日志
	if err := logger.InitLogger(&cfg.Logger); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer logger.Sync()

	logger.Info("正在启动图片管理系统...")

	// 3. 初始化数据库
	if err := database.InitDatabase(&cfg.Database); err != nil {
		logger.Fatal("初始化数据库失败", zap.Error(err))
	}

	// 4. 自动迁移数据库表
	if err := database.AutoMigrate(); err != nil {
		logger.Fatal("数据库迁移失败", zap.Error(err))
	}

	// 5. 初始化存储
	var storageImpl storage.Storage
	switch cfg.Storage.Default {
	case "local":
		storageImpl, err = storage.NewLocalStorage(&cfg.Storage.Local)
		if err != nil {
			logger.Fatal("初始化本地存储失败", zap.Error(err))
		}
		logger.Info("使用本地存储")
	default:
		logger.Fatal("不支持的存储类型", zap.String("type", cfg.Storage.Default))
	}

	// 6. 初始化Repository层
	imageRepo := repository.NewImageRepository(database.GetDB())

	// 7. 初始化Service层
	imageService := service.NewImageService(imageRepo, storageImpl, cfg)

	// 8. 初始化Handler层
	authHandler := handler.NewAuthHandler(cfg)
	imageHandler := handler.NewImageHandler(imageService)

	// 9. 设置路由
	r := router.SetupRouter(cfg, authHandler, imageHandler)

	// 10. 创建HTTP服务器
	srv := &http.Server{
		Addr:    cfg.Server.GetAddr(),
		Handler: r,
	}

	// 11. 启动服务器（异步）
	go func() {
		logger.Info("服务器启动成功",
			zap.String("addr", cfg.Server.GetAddr()),
			zap.String("mode", cfg.Server.Mode))

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("服务器启动失败", zap.Error(err))
		}
	}()

	// 12. 等待中断信号优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务器...")

	// 13. 优雅关闭（5秒超时）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("服务器强制关闭", zap.Error(err))
	}

	// 14. 关闭数据库连接
	if err := database.Close(); err != nil {
		logger.Error("关闭数据库连接失败", zap.Error(err))
	}

	logger.Info("服务器已关闭")
}
