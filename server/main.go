package main

import (
	_ "net/http/pprof"
)
import (
	"context"
	"gallary/server/internal"
	"gallary/server/internal/model"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"gallary/server/config"
	"gallary/server/internal/handler"
	"gallary/server/internal/middleware"
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

	platformConfig := &internal.PlatformConfig{
		AdminConfig: &middleware.AdminConfig{
			JWTConfig: cfg.JWT,
		},
		DynamicStaticConfig: middleware.NewDynamicStaticConfig(),
	}

	settingService, migrationService, imageService, shareService := initService(platformConfig, cfg)

	// 11. 设置路由
	r := router.SetupRouter(
		cfg,
		handler.NewAuthHandler(platformConfig),
		handler.NewImageHandler(imageService),
		handler.NewShareHandler(shareService),
		handler.NewSettingHandler(settingService),
		handler.NewStorageHandler(settingService.GetStorageManager(), settingService),
		handler.NewMigrationHandler(migrationService),
		platformConfig,
	)

	// 12. 启动回收站自动清理任务
	stopCleanup := startTrashCleanupTask(imageService, platformConfig)
	defer stopCleanup()

	// 13. 创建HTTP服务器
	srv := &http.Server{
		Addr:    cfg.Server.GetAddr(),
		Handler: r,
	}

	// 14. 启动服务器（异步）
	go func() {
		logger.Info("服务器启动成功",
			zap.String("addr", cfg.Server.GetAddr()),
			zap.String("mode", cfg.Server.Mode))

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("服务器启动失败", zap.Error(err))
		}
	}()

	// 15. 等待中断信号优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务器...")

	// 16. 优雅关闭（5秒超时）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("服务器强制关闭", zap.Error(err))
	}

	// 17. 关闭数据库连接
	if err := database.Close(); err != nil {
		logger.Error("关闭数据库连接失败", zap.Error(err))
	}

	logger.Info("服务器已关闭")
}

func initService(platformConfig *internal.PlatformConfig, cfg *config.Config) (service.SettingService, service.MigrationService, service.ImageService, service.ShareService) {
	var err error
	// 6. 初始化Repository层
	imageRepo, shareRepo, settingRepo, migrationRepo := repository.NewImageRepository(), repository.NewShareRepository(), repository.NewSettingRepository(), repository.NewMigrationRepository()

	// 7. 初始化设置服务并加载数据库设置
	settingService := service.NewSettingService(settingRepo, platformConfig)

	// 7.1 初始化默认设置（使用代码默认值）
	if err := settingService.InitializeDefaults(context.Background()); err != nil {
		logger.Warn("初始化默认设置失败", zap.Error(err))
	}

	// 7.2 从数据库加载设置并应用到运行时（必须在初始化存储之前）
	var storageManager *storage.StorageManager
	if storageManager, err = settingService.ResetStorage(context.Background()); err != nil {
		logger.Fatal("初始化存储失败", zap.Error(err))
	}

	// 7.3 设置平台设置
	initPlatformConfig(platformConfig, settingService)

	// 8 初始化迁移服务
	migrationService := service.NewMigrationService(
		migrationRepo,
		imageRepo,
		settingRepo,
		storageManager,
	)

	// 9. 初始化Service层
	imageService := service.NewImageService(imageRepo, storageManager, cfg)
	shareService := service.NewShareService(shareRepo, storageManager)

	// 10. 初始化Handler层
	// 10.1 连接 SettingService 和 MigrationService
	settingService.SetMigrationService(migrationService)
	return settingService, migrationService, imageService, shareService
}

func initPlatformConfig(platformConfig *internal.PlatformConfig, settingService service.SettingService) {
	var err error
	// 8.1 初始化动态静态文件配置
	platformConfig.AdminConfig.Password, err = settingService.GetPassword(context.Background())
	if err != nil {
		logger.Fatal("获取账户信息失败", zap.Error(err))
	}
	platformConfig.AdminConfig.PasswordVersion, err = settingService.GetPasswordVersion(context.Background())
	if err != nil {
		logger.Fatal("获取账户信息失败", zap.Error(err))
	}

	settings, err := settingService.GetSettingsByCategory(context.Background(), model.SettingCategoryCleanup)
	if err != nil {
		logger.Fatal("获取账户信息失败", zap.Error(err))
	}
	po := settings.(model.CleanupPO)
	platformConfig.CleanupPO = &po

	storage, err := settingService.GetSettingsByCategory(context.Background(), model.SettingCategoryStorage)
	if err != nil {
		logger.Fatal("获取存储设置失败", zap.Error(err))
	}

	localConfig := storage.(service.StorageConfigDTO).LocalConfig
	platformConfig.DynamicStaticConfig.Update(localConfig.URLPrefix, localConfig.BasePath)
	platformConfig.Enable()
}

// startTrashCleanupTask 启动回收站自动清理任务
func startTrashCleanupTask(imageService service.ImageService, cfg *internal.PlatformConfig) func() {
	if cfg.TrashAutoDeleteDays <= 0 {
		logger.Info("回收站自动清理已禁用")
		return func() {}
	}

	ticker := time.NewTicker(1 * time.Hour) // 每小时检查一次
	done := make(chan bool)

	go func() {
		logger.Info("回收站自动清理任务已启动",
			zap.Int("auto_delete_days", cfg.TrashAutoDeleteDays))

		// 启动时先执行一次
		ctx := context.Background()
		if count, err := imageService.CleanupExpiredTrash(ctx); err != nil {
			logger.Error("回收站自动清理失败", zap.Error(err))
		} else if count > 0 {
			logger.Info("回收站自动清理完成", zap.Int("deleted_count", count))
		}

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				ctx := context.Background()
				if count, err := imageService.CleanupExpiredTrash(ctx); err != nil {
					logger.Error("回收站自动清理失败", zap.Error(err))
				} else if count > 0 {
					logger.Info("回收站自动清理完成", zap.Int("deleted_count", count))
				}
			}
		}
	}()

	return func() {
		ticker.Stop()
		done <- true
		logger.Info("回收站自动清理任务已停止")
	}
}
