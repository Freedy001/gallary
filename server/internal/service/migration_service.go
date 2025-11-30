package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"gallary/server/config"
	"gallary/server/internal/middleware"
	"gallary/server/internal/model"
	"gallary/server/internal/repository"
	"gallary/server/internal/storage"
	"gallary/server/pkg/database"
	"gallary/server/pkg/logger"

	"go.uber.org/zap"
)

// MigrationService 迁移服务接口
type MigrationService interface {
	// 开始迁移（异步）
	StartMigration(ctx context.Context, newBasePath, newURLPrefix string) (*model.MigrationTask, error)

	// 获取迁移状态
	GetMigrationStatus(ctx context.Context, taskID int64) (*model.MigrationTask, error)

	// 获取当前正在进行的迁移
	GetActiveMigration(ctx context.Context) (*model.MigrationTask, error)

	// 取消迁移（触发回滚）
	CancelMigration(ctx context.Context, taskID int64) error

	// 获取当前存储配置
	GetCurrentConfig() (basePath, urlPrefix string)
}

type migrationService struct {
	migrationRepo       repository.MigrationRepository
	imageRepo           repository.ImageRepository
	settingRepo         repository.SettingRepository
	dynamicStaticConfig *middleware.DynamicStaticConfig
	storageManager      *storage.StorageManager

	// 当前存储配置
	currentBasePath  string
	currentURLPrefix string
	configMu         sync.RWMutex

	// 迁移锁，确保同一时间只有一个迁移任务
	migrationMu sync.Mutex

	// 取消信号
	cancelChan chan struct{}
	cancelMu   sync.Mutex
}

// NewMigrationService 创建迁移服务实例
func NewMigrationService(
	migrationRepo repository.MigrationRepository,
	imageRepo repository.ImageRepository,
	settingRepo repository.SettingRepository,
	dynamicStaticConfig *middleware.DynamicStaticConfig,
	storageManager *storage.StorageManager,
	initialBasePath, initialURLPrefix string,
) MigrationService {

	svc := &migrationService{
		migrationRepo:       migrationRepo,
		imageRepo:           imageRepo,
		settingRepo:         settingRepo,
		dynamicStaticConfig: dynamicStaticConfig,
		storageManager:      storageManager,
		currentBasePath:     initialBasePath,
		currentURLPrefix:    initialURLPrefix,
	}

	// 初始化动态静态配置
	if initialBasePath != "" && initialURLPrefix != "" {
		dynamicStaticConfig.Update(initialURLPrefix, initialBasePath)
	}

	return svc
}

// GetCurrentConfig 获取当前存储配置
func (s *migrationService) GetCurrentConfig() (basePath, urlPrefix string) {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.currentBasePath, s.currentURLPrefix
}

// updateCurrentConfig 更新当前存储配置
func (s *migrationService) updateCurrentConfig(basePath, urlPrefix string) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	s.currentBasePath = basePath
	s.currentURLPrefix = urlPrefix
}

// StartMigration 开始迁移任务
func (s *migrationService) StartMigration(ctx context.Context, newBasePath, newURLPrefix string) (*model.MigrationTask, error) {
	// 1. 获取迁移锁
	if !s.migrationMu.TryLock() {
		return nil, fmt.Errorf("已有迁移任务正在进行")
	}

	// 2. 获取当前配置
	oldBasePath, oldURLPrefix := s.GetCurrentConfig()

	// 检查配置是否相同
	if oldBasePath == newBasePath && oldURLPrefix == newURLPrefix {
		s.migrationMu.Unlock()
		return nil, fmt.Errorf("新配置与当前配置相同")
	}

	// 3. 验证新目录可写
	if err := s.validateNewPath(newBasePath); err != nil {
		s.migrationMu.Unlock()
		return nil, fmt.Errorf("新目录验证失败: %w", err)
	}

	// 4. 统计需要迁移的文件数量
	totalFiles, err := s.countFilesToMigrate(ctx)
	if err != nil {
		s.migrationMu.Unlock()
		return nil, fmt.Errorf("统计文件数量失败: %w", err)
	}

	// 5. 创建迁移任务
	now := time.Now()
	task := &model.MigrationTask{
		Status:       model.MigrationStatusPending,
		OldBasePath:  oldBasePath,
		OldURLPrefix: oldURLPrefix,
		NewBasePath:  newBasePath,
		NewURLPrefix: newURLPrefix,
		TotalFiles:   totalFiles,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.migrationRepo.Create(ctx, task); err != nil {
		s.migrationMu.Unlock()
		return nil, fmt.Errorf("创建迁移任务失败: %w", err)
	}

	// 6. 初始化取消信号
	s.cancelMu.Lock()
	s.cancelChan = make(chan struct{})
	s.cancelMu.Unlock()

	// 7. 异步执行迁移
	go s.executeMigration(task.ID, oldBasePath, oldURLPrefix, newBasePath, newURLPrefix)

	return task, nil
}

// validateNewPath 验证新路径可用
func (s *migrationService) validateNewPath(newBasePath string) error {
	// 创建目录
	if err := os.MkdirAll(newBasePath, 0755); err != nil {
		return fmt.Errorf("无法创建目录: %w", err)
	}

	// 测试写入权限
	testFile := filepath.Join(newBasePath, ".migration_test")
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("目录不可写: %w", err)
	}
	f.Close()
	os.Remove(testFile)

	return nil
}

// countFilesToMigrate 统计需要迁移的文件数量
func (s *migrationService) countFilesToMigrate(ctx context.Context) (int, error) {
	// 获取所有未删除的图片数量
	_, total, err := s.imageRepo.List(ctx, 1, 1)
	if err != nil {
		return 0, err
	}
	// 每张图片可能有原图和缩略图
	return int(total) * 2, nil
}

// executeMigration 执行迁移（goroutine 中运行）
func (s *migrationService) executeMigration(taskID int64, oldBasePath, oldURLPrefix, newBasePath, newURLPrefix string) {
	defer s.migrationMu.Unlock()

	ctx := context.Background()

	// 1. 更新任务状态为运行中
	now := time.Now()
	task, _ := s.migrationRepo.GetByID(ctx, taskID)
	if task != nil {
		task.Status = model.MigrationStatusRunning
		task.StartedAt = &now
		task.UpdatedAt = now
		s.migrationRepo.Update(ctx, task)
	}

	logger.Info("开始存储迁移",
		zap.Int64("task_id", taskID),
		zap.String("old_path", oldBasePath),
		zap.String("new_path", newBasePath))

	// 2. 禁用静态文件服务（迁移期间图片不可访问）
	s.dynamicStaticConfig.Disable()

	// 3. 标记所有图片为迁移中
	if err := s.markAllImagesMigrating(ctx, taskID); err != nil {
		s.handleMigrationFailure(ctx, taskID, "标记图片状态失败: "+err.Error())
		s.rollback(ctx, taskID, oldBasePath, oldURLPrefix, newBasePath)
		return
	}

	// 检查是否被取消
	if s.isCancelled() {
		s.handleMigrationCancelled(ctx, taskID)
		s.rollback(ctx, taskID, oldBasePath, oldURLPrefix, newBasePath)
		return
	}

	// 4. 执行文件复制
	if err := s.copyFiles(ctx, taskID, oldBasePath, newBasePath); err != nil {
		s.handleMigrationFailure(ctx, taskID, "复制文件失败: "+err.Error())
		s.rollback(ctx, taskID, oldBasePath, oldURLPrefix, newBasePath)
		return
	}

	// 5. 验证所有文件
	if err := s.verifyFiles(ctx, taskID, oldBasePath, newBasePath); err != nil {
		s.handleMigrationFailure(ctx, taskID, "验证文件失败: "+err.Error())
		s.rollback(ctx, taskID, oldBasePath, oldURLPrefix, newBasePath)
		return
	}

	// 6. 提交迁移：更新数据库配置和图片状态
	if err := s.commitMigration(ctx, taskID, newBasePath, newURLPrefix); err != nil {
		s.handleMigrationFailure(ctx, taskID, "提交迁移失败: "+err.Error())
		s.rollback(ctx, taskID, oldBasePath, oldURLPrefix, newBasePath)
		return
	}

	// 7. 删除旧文件
	s.deleteOldFiles(ctx, oldBasePath)

	// 8. 更新任务状态为完成
	completedAt := time.Now()
	task, _ = s.migrationRepo.GetByID(ctx, taskID)
	if task != nil {
		task.Status = model.MigrationStatusCompleted
		task.CompletedAt = &completedAt
		task.UpdatedAt = completedAt
		s.migrationRepo.Update(ctx, task)
	}

	// 9. 更新当前配置并重新启用静态文件服务
	s.updateCurrentConfig(newBasePath, newURLPrefix)
	s.dynamicStaticConfig.Update(newURLPrefix, newBasePath)

	// 10. 重新初始化存储管理器以使用新路径
	if s.storageManager != nil {
		newStorageCfg := &config.StorageConfig{
			Default: config.StorageTypeLocal,
			Local: config.LocalStorageConfig{
				BasePath:  newBasePath,
				URLPrefix: newURLPrefix,
			},
		}
		if err := s.storageManager.SwitchStorage(newStorageCfg); err != nil {
			logger.Warn("重新初始化存储管理器失败", zap.Error(err))
		} else {
			logger.Info("存储管理器已更新到新路径", zap.String("path", newBasePath))
		}
	}

	logger.Info("存储迁移完成", zap.Int64("task_id", taskID))
}

// markAllImagesMigrating 标记所有图片为迁移中
func (s *migrationService) markAllImagesMigrating(ctx context.Context, taskID int64) error {
	status := string(model.MigrationStatusRunning)
	return database.GetDB(ctx).Model(&model.Image{}).
		Where("deleted_at IS NULL").
		Updates(map[string]interface{}{
			"migration_status":  status,
			"migration_task_id": taskID,
		}).Error
}

// clearImageMigrationStatus 清除图片迁移状态
func (s *migrationService) clearImageMigrationStatus(ctx context.Context, taskID int64) error {
	return database.GetDB(ctx).Model(&model.Image{}).
		Where("migration_task_id = ?", taskID).
		Updates(map[string]interface{}{
			"migration_status":  nil,
			"migration_task_id": nil,
		}).Error
}

// copyFiles 复制文件到新目录
func (s *migrationService) copyFiles(ctx context.Context, taskID int64, oldBasePath, newBasePath string) error {
	page := 1
	pageSize := 100
	var processedCount int32

	for {
		// 检查是否被取消
		if s.isCancelled() {
			return fmt.Errorf("迁移已取消")
		}

		images, total, err := s.imageRepo.List(ctx, page, pageSize)
		if err != nil {
			return err
		}

		for _, img := range images {
			// 检查是否被取消
			if s.isCancelled() {
				return fmt.Errorf("迁移已取消")
			}

			// 复制原图
			if img.StoragePath != "" {
				if err := s.copyFile(oldBasePath, newBasePath, img.StoragePath); err != nil {
					return fmt.Errorf("复制图片 %d 失败: %w", img.ID, err)
				}
				atomic.AddInt32(&processedCount, 1)
				s.migrationRepo.IncrementProcessed(ctx, taskID)
			}

			// 复制缩略图
			if img.ThumbnailPath != "" {
				if err := s.copyFile(oldBasePath, newBasePath, img.ThumbnailPath); err != nil {
					return fmt.Errorf("复制缩略图 %d 失败: %w", img.ID, err)
				}
				atomic.AddInt32(&processedCount, 1)
				s.migrationRepo.IncrementProcessed(ctx, taskID)
			}
		}

		if int64(page*pageSize) >= total {
			break
		}
		page++
	}

	logger.Info("文件复制完成", zap.Int32("count", processedCount))
	return nil
}

// copyFile 复制单个文件
func (s *migrationService) copyFile(oldBasePath, newBasePath, relativePath string) error {
	srcPath := filepath.Join(oldBasePath, relativePath)
	dstPath := filepath.Join(newBasePath, relativePath)

	// 检查源文件是否存在
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		logger.Warn("源文件不存在，跳过", zap.String("path", srcPath))
		return nil
	}

	// 确保目标目录存在
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 打开源文件
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer src.Close()

	// 创建目标文件
	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dst.Close()

	// 复制内容
	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(dstPath) // 清理失败的文件
		return fmt.Errorf("复制内容失败: %w", err)
	}

	return nil
}

// verifyFiles 验证复制的文件
func (s *migrationService) verifyFiles(ctx context.Context, taskID int64, oldBasePath, newBasePath string) error {
	page := 1
	pageSize := 100

	for {
		images, total, err := s.imageRepo.List(ctx, page, pageSize)
		if err != nil {
			return err
		}

		for _, img := range images {
			// 验证原图
			if img.StoragePath != "" {
				if err := s.verifyFile(oldBasePath, newBasePath, img.StoragePath); err != nil {
					return fmt.Errorf("验证图片 %d 失败: %w", img.ID, err)
				}
			}

			// 验证缩略图
			if img.ThumbnailPath != "" {
				if err := s.verifyFile(oldBasePath, newBasePath, img.ThumbnailPath); err != nil {
					return fmt.Errorf("验证缩略图 %d 失败: %w", img.ID, err)
				}
			}
		}

		if int64(page*pageSize) >= total {
			break
		}
		page++
	}

	logger.Info("文件验证完成")
	return nil
}

// verifyFile 验证单个文件
func (s *migrationService) verifyFile(oldBasePath, newBasePath, relativePath string) error {
	srcPath := filepath.Join(oldBasePath, relativePath)
	dstPath := filepath.Join(newBasePath, relativePath)

	// 检查源文件是否存在
	srcInfo, err := os.Stat(srcPath)
	if os.IsNotExist(err) {
		// 源文件不存在，跳过验证
		return nil
	}
	if err != nil {
		return fmt.Errorf("获取源文件信息失败: %w", err)
	}

	// 检查目标文件
	dstInfo, err := os.Stat(dstPath)
	if err != nil {
		return fmt.Errorf("目标文件不存在: %w", err)
	}

	// 验证文件大小
	if srcInfo.Size() != dstInfo.Size() {
		return fmt.Errorf("文件大小不匹配: 源=%d, 目标=%d", srcInfo.Size(), dstInfo.Size())
	}

	return nil
}

// commitMigration 提交迁移
func (s *migrationService) commitMigration(ctx context.Context, taskID int64, newBasePath, newURLPrefix string) error {
	// 更新数据库中的存储配置
	now := time.Now()
	settings := []model.Setting{
		{
			Category:  model.SettingCategoryStorage,
			Key:       model.SettingKeyLocalBasePath,
			Value:     newBasePath,
			ValueType: model.SettingValueTypeString,
			UpdatedAt: now,
		},
		{
			Category:  model.SettingCategoryStorage,
			Key:       model.SettingKeyLocalURLPrefix,
			Value:     newURLPrefix,
			ValueType: model.SettingValueTypeString,
			UpdatedAt: now,
		},
	}

	if err := s.settingRepo.BatchUpsert(ctx, settings); err != nil {
		return fmt.Errorf("更新存储配置失败: %w", err)
	}

	// 清除图片迁移状态
	if err := s.clearImageMigrationStatus(ctx, taskID); err != nil {
		return fmt.Errorf("清除迁移状态失败: %w", err)
	}

	logger.Info("迁移提交完成")
	return nil
}

// deleteOldFiles 删除旧目录中的文件
func (s *migrationService) deleteOldFiles(ctx context.Context, oldBasePath string) {
	// 遍历删除旧目录中的所有文件
	err := os.RemoveAll(oldBasePath)
	if err != nil {
		logger.Warn("删除旧目录失败", zap.String("path", oldBasePath), zap.Error(err))
	} else {
		logger.Info("旧目录已删除", zap.String("path", oldBasePath))
	}
}

// rollback 回滚迁移
func (s *migrationService) rollback(ctx context.Context, taskID int64, oldBasePath, oldURLPrefix, newBasePath string) {
	logger.Info("开始回滚迁移", zap.Int64("task_id", taskID))

	// 1. 删除新目录中已复制的文件
	if err := os.RemoveAll(newBasePath); err != nil {
		logger.Warn("删除新目录失败", zap.String("path", newBasePath), zap.Error(err))
	}

	// 2. 恢复图片状态
	if err := s.clearImageMigrationStatus(ctx, taskID); err != nil {
		logger.Warn("清除迁移状态失败", zap.Error(err))
	}

	// 3. 重新启用静态文件服务（使用旧配置）
	s.dynamicStaticConfig.Update(oldURLPrefix, oldBasePath)

	logger.Info("迁移回滚完成", zap.Int64("task_id", taskID))
}

// handleMigrationFailure 处理迁移失败
func (s *migrationService) handleMigrationFailure(ctx context.Context, taskID int64, errorMsg string) {
	logger.Error("迁移失败", zap.Int64("task_id", taskID), zap.String("error", errorMsg))

	task, _ := s.migrationRepo.GetByID(ctx, taskID)
	if task != nil {
		task.Status = model.MigrationStatusFailed
		task.ErrorMessage = &errorMsg
		task.UpdatedAt = time.Now()
		s.migrationRepo.Update(ctx, task)
	}
}

// handleMigrationCancelled 处理迁移取消
func (s *migrationService) handleMigrationCancelled(ctx context.Context, taskID int64) {
	logger.Info("迁移已取消", zap.Int64("task_id", taskID))

	task, _ := s.migrationRepo.GetByID(ctx, taskID)
	if task != nil {
		task.Status = model.MigrationStatusCancelled
		task.UpdatedAt = time.Now()
		s.migrationRepo.Update(ctx, task)
	}
}

// isCancelled 检查是否被取消
func (s *migrationService) isCancelled() bool {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()

	if s.cancelChan == nil {
		return false
	}

	select {
	case <-s.cancelChan:
		return true
	default:
		return false
	}
}

// GetMigrationStatus 获取迁移状态
func (s *migrationService) GetMigrationStatus(ctx context.Context, taskID int64) (*model.MigrationTask, error) {
	return s.migrationRepo.GetByID(ctx, taskID)
}

// GetActiveMigration 获取当前活跃的迁移任务
func (s *migrationService) GetActiveMigration(ctx context.Context) (*model.MigrationTask, error) {
	return s.migrationRepo.GetActive(ctx)
}

// CancelMigration 取消迁移
func (s *migrationService) CancelMigration(ctx context.Context, taskID int64) error {
	task, err := s.migrationRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("迁移任务不存在")
	}

	if !task.IsActive() {
		return fmt.Errorf("迁移任务已结束，无法取消")
	}

	// 发送取消信号
	s.cancelMu.Lock()
	if s.cancelChan != nil {
		close(s.cancelChan)
		s.cancelChan = nil
	}
	s.cancelMu.Unlock()

	return nil
}
