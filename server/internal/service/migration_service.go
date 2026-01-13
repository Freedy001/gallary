package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"gallary/server/internal/model"
	"gallary/server/internal/repository"
	"gallary/server/internal/storage"
	"gallary/server/pkg/database"
	"gallary/server/pkg/logger"

	"go.uber.org/zap"
)

// MigrationService 迁移服务接口
type MigrationService interface {
	StartSelfMigration(ctx context.Context, oldBasePath, newBasePath string, storageType model.StorageId, onFinish func(error)) (*model.MigrationTask, error)

	// 获取迁移状态
	GetMigrationStatus(ctx context.Context, taskID int64) (*model.MigrationTask, error)

	// 获取当前正在进行的迁移
	GetActiveMigration(ctx context.Context) (*model.MigrationTask, error)

	// 检查是否正在迁移
	IsRunning() bool
}

type migrationService struct {
	migrationRepo repository.MigrationRepository
	imageRepo     repository.ImageRepository
	settingRepo   repository.SettingRepository
	//dynamicStaticConfig *middleware.DynamicStaticConfig
	storageManager *storage.StorageManager

	configMu sync.RWMutex

	// 迁移锁，确保同一时间只有一个迁移任务
	migrationMu sync.Mutex

	// 是否正在迁移
	isRunning atomic.Bool
}

// NewMigrationService 创建迁移服务实例
func NewMigrationService(
	migrationRepo repository.MigrationRepository,
	imageRepo repository.ImageRepository,
	settingRepo repository.SettingRepository,
	storageManager *storage.StorageManager,
) MigrationService {

	svc := &migrationService{
		migrationRepo:  migrationRepo,
		imageRepo:      imageRepo,
		settingRepo:    settingRepo,
		storageManager: storageManager,
	}

	return svc
}

// IsRunning 检查是否正在迁移
func (s *migrationService) IsRunning() bool {
	return s.isRunning.Load()
}

// StartSelfMigration 开始阿里云盘迁移任务
func (s *migrationService) StartSelfMigration(ctx context.Context, oldBasePath, newBasePath string, storageType model.StorageId, onFinish func(error)) (*model.MigrationTask, error) {
	// 1. 获取迁移锁
	if !s.migrationMu.TryLock() {
		return nil, fmt.Errorf("已有迁移任务正在进行")
	}

	// 检查配置是否相同
	if oldBasePath == newBasePath {
		s.migrationMu.Unlock()
		return nil, fmt.Errorf("新配置与当前配置相同")
	}

	// 2. 统计需要迁移的文件数量（只统计阿里云盘存储的图片）
	totalFiles, err := s.countFilesToMigrate(ctx, storageType)
	if err != nil {
		s.migrationMu.Unlock()
		return nil, fmt.Errorf("统计文件数量失败: %w", err)
	}

	// 3. 创建迁移任务
	now := time.Now()
	task := &model.MigrationTask{
		Status:      model.MigrationStatusPending,
		OldBasePath: oldBasePath,
		NewBasePath: newBasePath,
		TotalFiles:  totalFiles,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.migrationRepo.Create(ctx, task); err != nil {
		s.migrationMu.Unlock()
		return nil, fmt.Errorf("创建迁移任务失败: %w", err)
	}

	// 4. 设置运行状态
	s.isRunning.Store(true)

	// 5. 异步执行阿里云盘迁移
	go s.executeSelfMigration(task.ID, oldBasePath, newBasePath, storageType, onFinish)

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
	_ = f.Close()
	_ = os.Remove(testFile)

	return nil
}

// countFilesToMigrate 统计需要迁移的本地存储文件数量
func (s *migrationService) countFilesToMigrate(ctx context.Context, storageType model.StorageId) (int, error) {
	// 获取所有本地存储的未删除图片数量
	count, err := s.imageRepo.CountByStorageType(ctx, string(storageType))
	if err != nil {
		return 0, err
	}
	// 每张图片可能有原图和缩略图
	return count * 2, nil
}

// executeMigration 执行迁移（goroutine 中运行）
func (s *migrationService) executeMigration(taskID int64, oldBasePath, newBasePath string) {
	defer s.migrationMu.Unlock()
	defer s.isRunning.Store(false)

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

	// 3. 标记本地存储图片为迁移中
	if err := s.markImagesMigrating(ctx, taskID, model.StorageTypeLocal); err != nil {
		s.handleMigrationFailure(ctx, taskID, "标记图片状态失败: "+err.Error())
		return
	}

	// 4. 执行文件复制
	if err := s.executeMove(ctx, model.StorageTypeLocal, oldBasePath, newBasePath); err != nil {
		s.handleMigrationFailure(ctx, taskID, "复制文件失败: "+err.Error())
		return
	}

	// 5. 提交迁移：更新数据库配置和图片状态
	if err := s.clearImageMigrationStatus(ctx, taskID); err != nil {
		s.handleMigrationFailure(ctx, taskID, "提交迁移失败: "+err.Error())
		return
	}

	// 7. 更新任务状态为完成
	completedAt := time.Now()
	task, _ = s.migrationRepo.GetByID(ctx, taskID)
	if task != nil {
		task.Status = model.MigrationStatusCompleted
		task.CompletedAt = &completedAt
		task.UpdatedAt = completedAt
		s.migrationRepo.Update(ctx, task)
	}

	// 10. 重新初始化存储管理器以使用新路径
	//if s.storageManager != nil {
	//	newStorageCfg := &config.StorageConfig{
	//		Default: config.StorageTypeLocal,
	//		Local: config.LocalStorageConfig{
	//			Path:  newBasePath,
	//			URLPrefix: newURLPrefix,
	//		},
	//	}
	//	if err := s.storageManager.SwitchStorage(newStorageCfg); err != nil {
	//		logger.Warn("重新初始化存储管理器失败", zap.Error(err))
	//	} else {
	//		logger.DriverId("存储管理器已更新到新路径", zap.String("path", newBasePath))
	//	}
	//}

	logger.Info("存储迁移完成", zap.Int64("task_id", taskID))
}

// executeSelfMigration 执行阿里云盘迁移（goroutine 中运行）
func (s *migrationService) executeSelfMigration(taskID int64, oldBasePath, newBasePath string, storageType model.StorageId, onFinish func(error)) {
	defer s.migrationMu.Unlock()
	defer s.isRunning.Store(false)

	ctx := context.Background()

	// 1. 更新任务状态为运行中
	now := time.Now()
	task, err := s.migrationRepo.GetByID(ctx, taskID)
	if err != nil {
		onFinish(err)
		logger.Error("未找到迁移任务", zap.Int64("task_id", taskID), zap.Error(err))
		return
	}

	task.Status = model.MigrationStatusRunning
	task.StartedAt = &now
	task.UpdatedAt = now
	err = s.migrationRepo.Update(ctx, task)
	if err != nil {
		onFinish(err)
		logger.Error("迁移失败", zap.Int64("task_id", taskID), zap.Error(err))
		return
	}

	logger.Info("开始阿里云盘存储迁移",
		zap.Int64("task_id", taskID),
		zap.String("old_path", oldBasePath),
		zap.String("new_path", newBasePath))

	// 2. 标记阿里云盘存储图片为迁移中
	if err := s.markImagesMigrating(ctx, taskID, storageType); err != nil {
		s.handleMigrationFailure(ctx, taskID, "标记图片状态失败: "+err.Error())
		onFinish(err)
		return
	}

	// 3. 使用 Move API 移动文件
	if err := s.executeMove(ctx, storageType, oldBasePath, newBasePath); err != nil {
		s.handleMigrationFailure(ctx, taskID, "移动文件失败: "+err.Error())
		onFinish(err)
		return
	}

	// 4. 提交阿里云盘迁移
	if err := s.clearImageMigrationStatus(ctx, taskID); err != nil {
		s.handleMigrationFailure(ctx, taskID, "提交迁移失败: "+err.Error())
		onFinish(err)
		return
	}

	// 5. 更新任务状态为完成
	completedAt := time.Now()
	task.Status = model.MigrationStatusCompleted
	task.CompletedAt = &completedAt
	task.UpdatedAt = completedAt
	err = s.migrationRepo.Update(ctx, task)
	if err != nil {
		logger.Error("更新任务状态失败", zap.Int64("task_id", taskID), zap.Error(err))
		onFinish(err)
		return
	}

	onFinish(nil)
	logger.Info("阿里云盘存储迁移完成", zap.Int64("task_id", taskID))
}

// markImagesMigrating 标记本地存储图片为迁移中
func (s *migrationService) markImagesMigrating(ctx context.Context, taskID int64, storageType model.StorageId) error {
	status := string(model.MigrationStatusRunning)
	return database.GetDB(ctx).Model(&model.Image{}).
		Where("deleted_at IS NULL AND storage_type = ?", string(storageType)).
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

// executeMove 复制本地文件到新目录
func (s *migrationService) executeMove(ctx context.Context, storageType model.StorageId, oldBasePath, newBasePath string) error {
	err := s.storageManager.Move(context.WithValue(ctx, storage.OverrideStorageType, storageType), "", oldBasePath, newBasePath)
	if err != nil {
		return err
	}

	logger.Info("迁移完成", zap.String("storageType", string(storageType)))
	return nil
}

// commitAliyunPanMigration 提交阿里云盘迁移
//func (s *migrationService) commitAliyunPanMigration(ctx context.Context, taskID int64, newBasePath string) error {
//	// 更新数据库中的阿里云盘基础路径配置
//	now := time.Now()
//	settings := []model.Setting{
//		{
//			Category:  model.SettingCategoryStorage,
//			Key:       model.SettingKeyAliyunPanBasePath,
//			Value:     newBasePath,
//			ValueType: model.SettingValueTypeString,
//			UpdatedAt: now,
//		},
//	}
//
//	if err := s.settingRepo.BatchUpsert(ctx, settings); err != nil {
//		return fmt.Errorf("更新阿里云盘配置失败: %w", err)
//	}
//
//	// 清除图片迁移状态
//	if err := s.clearImageMigrationStatus(ctx, taskID); err != nil {
//		return fmt.Errorf("清除迁移状态失败: %w", err)
//	}
//
//	logger.DriverId("阿里云盘迁移提交完成")
//	return nil
//}
//
//// commitMigration 提交迁移
//func (s *migrationService) commitMigration(ctx context.Context, taskID int64, newBasePath string) error {
//	// 更新数据库中的存储配置
//	now := time.Now()
//	settings := []model.Setting{
//		{
//			Category:  model.SettingCategoryStorage,
//			Key:       model.SettingKeyLocalBasePath,
//			Value:     newBasePath,
//			ValueType: model.SettingValueTypeString,
//			UpdatedAt: now,
//		},
//		{
//			Category:  model.SettingCategoryStorage,
//			Key:       model.SettingKeyLocalURLPrefix,
//			Value:     newURLPrefix,
//			ValueType: model.SettingValueTypeString,
//			UpdatedAt: now,
//		},
//	}
//
//	if err := s.settingRepo.BatchUpsert(ctx, settings); err != nil {
//		return fmt.Errorf("更新存储配置失败: %w", err)
//	}
//
//	// 清除图片迁移状态
//	if err := s.clearImageMigrationStatus(ctx, taskID); err != nil {
//		return fmt.Errorf("清除迁移状态失败: %w", err)
//	}
//
//	logger.DriverId("迁移提交完成")
//	return nil
//}
//

// handleMigrationFailure 处理迁移失败
func (s *migrationService) handleMigrationFailure(ctx context.Context, taskID int64, errorMsg string) {
	logger.Error("迁移失败", zap.Int64("task_id", taskID), zap.String("error", errorMsg))

	task, _ := s.migrationRepo.GetByID(ctx, taskID)
	if task != nil {
		task.Status = model.MigrationStatusFailed
		task.ErrorMessage = &errorMsg
		task.UpdatedAt = time.Now()
		_ = s.migrationRepo.Update(ctx, task)
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
