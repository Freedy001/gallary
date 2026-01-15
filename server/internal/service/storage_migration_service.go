package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"gallary/server/internal/model"
	"gallary/server/internal/repository"
	"gallary/server/internal/storage"
	"gallary/server/internal/websocket"
	"gallary/server/pkg/database"
	"gallary/server/pkg/logger"

	"go.uber.org/zap"
)

// StorageMigrationService 存储迁移服务接口
type StorageMigrationService interface {
	// 创建迁移任务
	CreateMigration(ctx context.Context, req *CreateMigrationRequest) (*model.StorageMigrationTask, error)
	// 预览迁移（统计待迁移文件）
	PreviewMigration(ctx context.Context, req *CreateMigrationRequest) (*MigrationPreview, error)

	// 获取失败的文件记录
	GetFailedFileRecords(ctx context.Context, taskID int64, page, pageSize int) ([]*model.MigrationFileRecordVO, int64, error)
	// 获取迁移状态 VO（用于 WebSocket）
	GetMigrationStatusVO(ctx context.Context) (*model.MigrationStatusVO, error)

	// 暂停迁移任务
	PauseMigration(ctx context.Context, taskID int64) error
	// 恢复迁移任务
	ResumeMigration(ctx context.Context, taskID int64) error
	// 重试失败的文件
	RetryFailedFiles(ctx context.Context, taskID int64) error
	// 忽略失败的文件（清除失败记录，标记任务完成）
	DismissMigration(ctx context.Context, taskID int64) error
}

// CreateMigrationRequest 创建迁移请求
type CreateMigrationRequest struct {
	MigrationType              model.MigrationType              `json:"migration_type" binding:"required"`
	SourceStorageId            model.StorageId                  `json:"source_storage_id" binding:"required"`
	TargetStorageId            model.StorageId                  `json:"target_storage_id" binding:"required"`
	Filter                     *model.MigrationFilterConditions `json:"filter,omitempty"`
	DeleteSourceAfterMigration bool                             `json:"delete_source_after_migration"`
}

// MigrationPreview 迁移预览
type MigrationPreview struct {
	FilesCount           int   `json:"files_count"`
	TotalSize            int64 `json:"total_size"`
	EstimatedTimeSeconds int   `json:"estimated_time_seconds"`
}

type storageMigrationService struct {
	migrationRepo  repository.StorageMigrationRepository
	imageRepo      repository.ImageRepository
	storageManager *storage.StorageManager
	notifier       websocket.Notifier

	// 迁移控制
	migrationMu sync.Mutex
	isRunning   atomic.Bool
	// 用于取消/暂停的控制
	cancelFuncs sync.Map // map[int64]context.CancelFunc
}

// NewStorageMigrationService 创建存储迁移服务实例
func NewStorageMigrationService(
	migrationRepo repository.StorageMigrationRepository,
	imageRepo repository.ImageRepository,
	storageManager *storage.StorageManager,
	notifier websocket.Notifier,
) StorageMigrationService {
	return &storageMigrationService{
		migrationRepo:  migrationRepo,
		imageRepo:      imageRepo,
		storageManager: storageManager,
		notifier:       notifier,
	}
}

// CreateMigration 创建迁移任务
func (s *storageMigrationService) CreateMigration(ctx context.Context, req *CreateMigrationRequest) (*model.StorageMigrationTask, error) {
	defer s.notifyProgress(ctx)
	// 验证参数
	if req.SourceStorageId == req.TargetStorageId {
		return nil, fmt.Errorf("源存储和目标存储不能相同")
	}

	// 统计需要迁移的文件数量
	count, _, err := s.countFilesToMigrate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("统计文件数量失败: %w", err)
	}

	if count == 0 {
		return nil, fmt.Errorf("没有需要迁移的文件")
	}

	// 使用事务创建任务和文件记录
	task, err := database.Transaction1(ctx, func(txCtx context.Context) (*model.StorageMigrationTask, error) {
		// 创建迁移任务
		now := time.Now()
		task := &model.StorageMigrationTask{
			MigrationType:              req.MigrationType,
			Status:                     model.StorageMigrationPending,
			SourceStorageId:            req.SourceStorageId,
			TargetStorageId:            req.TargetStorageId,
			FilterConditions:           req.Filter,
			TotalFiles:                 count,
			DeleteSourceAfterMigration: req.DeleteSourceAfterMigration,
			CreatedAt:                  now,
			UpdatedAt:                  now,
		}

		if err := s.migrationRepo.Create(txCtx, task); err != nil {
			return nil, fmt.Errorf("创建迁移任务失败: %w", err)
		}

		// 创建文件记录
		if err := s.createFileRecords(txCtx, task.ID, req); err != nil {
			return nil, fmt.Errorf("创建文件记录失败: %w", err)
		}

		return task, nil
	})

	if err != nil {
		return nil, err
	}

	// 异步执行迁移
	go s.executeMigration(task.ID)

	return task, nil
}

// PreviewMigration 预览迁移
func (s *storageMigrationService) PreviewMigration(ctx context.Context, req *CreateMigrationRequest) (*MigrationPreview, error) {
	count, totalSize, err := s.countFilesToMigrate(ctx, req)
	if err != nil {
		return nil, err
	}

	// 估算时间（假设每个文件平均 1 秒）
	estimatedTime := count

	return &MigrationPreview{
		FilesCount:           count,
		TotalSize:            totalSize,
		EstimatedTimeSeconds: estimatedTime,
	}, nil
}

// ListActiveMigrations 获取所有活跃迁移任务

// CancelMigration 取消迁移任务

// PauseMigration 暂停迁移任务
func (s *storageMigrationService) PauseMigration(ctx context.Context, taskID int64) error {
	defer s.notifyProgress(ctx)
	task, err := s.migrationRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}

	if task.Status != model.StorageMigrationRunning {
		return fmt.Errorf("只有运行中的任务才能暂停")
	}

	// 取消当前运行的 goroutine
	if cancel, ok := s.cancelFuncs.Load(taskID); ok {
		cancel.(context.CancelFunc)()
	}

	// 更新状态为暂停
	return s.migrationRepo.UpdateStatus(ctx, taskID, model.StorageMigrationPaused, nil)
}

// ResumeMigration 恢复迁移任务
func (s *storageMigrationService) ResumeMigration(ctx context.Context, taskID int64) error {
	defer s.notifyProgress(ctx)

	task, err := s.migrationRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}

	if task.Status != model.StorageMigrationPaused {
		return fmt.Errorf("只有暂停的任务才能恢复")
	}

	// 重置 in_progress 状态的记录为 pending（暂停时可能有正在处理的记录）
	_, _ = s.migrationRepo.ResetFailedFileRecords(ctx, taskID)

	// 异步执行迁移
	go s.executeMigration(taskID)

	return nil
}

// ListMigrations 获取迁移历史

// RetryFailedFiles 重试失败的文件
func (s *storageMigrationService) RetryFailedFiles(ctx context.Context, taskID int64) error {
	defer s.notifyProgress(ctx)
	task, err := s.migrationRepo.GetByIDWithStats(ctx, taskID)
	if err != nil {
		return err
	}

	// 只有已完成或失败且有失败文件的任务才能重试
	if (task.Status != model.StorageMigrationCompleted && task.Status != model.StorageMigrationFailed) || task.FailedFiles == 0 {
		return fmt.Errorf("只有已完成或失败且有失败文件的任务才能重试")
	}

	// 重置失败的文件记录为待处理状态
	resetCount, err := s.migrationRepo.ResetFailedFileRecords(ctx, taskID)
	if err != nil {
		return fmt.Errorf("重置失败记录失败: %w", err)
	}

	if resetCount == 0 {
		return fmt.Errorf("没有需要重试的失败文件")
	}

	// 更新任务状态
	task.Status = model.StorageMigrationPending
	task.CompletedAt = nil
	if err := s.migrationRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 异步执行迁移
	go s.executeMigration(taskID)

	logger.Info("重试失败文件", zap.Int64("task_id", taskID), zap.Int64("reset_count", resetCount))
	return nil
}

// GetFailedFileRecords 获取失败的文件记录
func (s *storageMigrationService) GetFailedFileRecords(ctx context.Context, taskID int64, page, pageSize int) ([]*model.MigrationFileRecordVO, int64, error) {
	records, total, err := s.migrationRepo.GetFailedFileRecords(ctx, taskID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// 转换为 VO 并填充缩略图 URL
	vos := make([]*model.MigrationFileRecordVO, 0, len(records))
	for _, record := range records {
		vo := &model.MigrationFileRecordVO{
			ID:        record.ID,
			TaskID:    record.TaskID,
			ImageID:   record.ImageID,
			Status:    record.Status,
			ErrorMsg:  record.ErrorMsg,
			CreatedAt: record.CreatedAt.Format("2006-01-02 15:04:05"),
		}

		// 填充图片名称和缩略图 URL
		if record.Image != nil {
			vo.ImageName = record.Image.OriginalName
			if s.storageManager != nil {
				_, thumbURL := s.storageManager.ImageUrl(record.Image)
				vo.ThumbURL = thumbURL
			}
		}

		vos = append(vos, vo)
	}

	return vos, total, nil
}

// DismissFailedFiles 忽略失败的文件（删除失败记录）
func (s *storageMigrationService) DismissMigration(ctx context.Context, taskID int64) error {
	defer s.notifyProgress(ctx)
	// 取消正在运行的 goroutine
	if cancel, ok := s.cancelFuncs.Load(taskID); ok {
		cancel.(context.CancelFunc)()
	}

	// 更新状态
	return s.migrationRepo.Delete(ctx, taskID)
}

// DeleteMigration 删除迁移任务（仅已完成、失败或已取消的任务可删除）

// GetMigrationStatusVO 获取迁移状态 VO（包含活跃任务和有失败文件的已完成任务）
func (s *storageMigrationService) GetMigrationStatusVO(ctx context.Context) (*model.MigrationStatusVO, error) {
	tasks, err := s.migrationRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	statusVO := &model.MigrationStatusVO{
		Tasks: make([]model.MigrationProgressVO, 0),
	}

	for _, task := range tasks {
		statusVO.Tasks = append(statusVO.Tasks, task.ToProgressVO())
		if task.Status == model.StorageMigrationRunning {
			statusVO.TotalRunning++
		} else if task.Status == model.StorageMigrationPaused {
			statusVO.TotalPaused++
		}
	}

	return statusVO, nil
}

// countFilesToMigrate 统计需要迁移的文件数量和总大小
func (s *storageMigrationService) countFilesToMigrate(ctx context.Context, req *CreateMigrationRequest) (int, int64, error) {
	return s.imageRepo.CountByMigrationFilter(ctx, req.MigrationType, req.SourceStorageId, req.Filter)
}

// createFileRecords 创建文件记录
func (s *storageMigrationService) createFileRecords(ctx context.Context, taskID int64, req *CreateMigrationRequest) error {
	// 获取需要迁移的图片 ID 列表
	imageIDs, err := s.imageRepo.FindIDsByMigrationFilter(ctx, req.MigrationType, req.SourceStorageId, req.Filter)
	if err != nil {
		return err
	}

	// 批量创建文件记录
	records := make([]*model.MigrationFileRecord, 0, len(imageIDs))
	now := time.Now()
	for _, imageID := range imageIDs {
		records = append(records, &model.MigrationFileRecord{
			TaskID:    taskID,
			ImageID:   imageID,
			Status:    model.MigrationFileRecordPending,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	return s.migrationRepo.CreateFileRecords(ctx, records)
}

// executeMigration 执行迁移任务（流水线模式）
func (s *storageMigrationService) executeMigration(taskID int64) {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFuncs.Store(taskID, cancel)
	defer func() {
		cancel()
		s.cancelFuncs.Delete(taskID)
	}()

	// 获取任务
	task, err := s.migrationRepo.GetByID(ctx, taskID)
	if err != nil || task == nil {
		logger.Error("获取迁移任务失败", zap.Int64("task_id", taskID), zap.Error(err))
		return
	}

	// 更新状态为运行中
	now := time.Now()
	task.Status = model.StorageMigrationRunning
	task.StartedAt = &now
	if err := s.migrationRepo.Update(ctx, task); err != nil {
		logger.Error("更新任务状态失败", zap.Int64("task_id", taskID), zap.Error(err))
		return
	}

	// 通知前端
	s.notifyProgress(ctx)

	logger.Info("开始存储迁移",
		zap.Int64("task_id", taskID),
		zap.String("migration_type", string(task.MigrationType)),
		zap.String("source", string(task.SourceStorageId)),
		zap.String("target", string(task.TargetStorageId)))

	// 创建 channel 作为流水线
	recordChan := make(chan *model.MigrationFileRecord, 50) // 缓冲区大小 50
	var producerWg sync.WaitGroup
	var consumerWg sync.WaitGroup

	// 启动消费者（工作协程，并发处理迁移）
	workerCount := 5
	for i := 0; i < workerCount; i++ {
		consumerWg.Add(1)
		go func() {
			defer consumerWg.Done()
			for record := range recordChan {
				select {
				case <-ctx.Done():
					return
				default:
					s.migrateFile(ctx, task, record)
				}
			}
		}()
	}

	// 启动生产者（获取待处理记录并发送到 channel）
	producerWg.Add(1)
	go func() {
		defer producerWg.Done()
		defer close(recordChan) // 关闭 channel，通知消费者退出

		for {
			// 检查是否被取消或暂停
			select {
			case <-ctx.Done():
				logger.Info("迁移任务被取消", zap.Int64("task_id", taskID))
				return
			default:
			}

			// 检查任务状态
			currentTask, err := s.migrationRepo.GetByID(ctx, taskID)
			if err != nil {
				logger.Error("获取任务状态失败", zap.Error(err))
				return
			}
			if currentTask.Status == model.StorageMigrationPaused || currentTask.Status == model.StorageMigrationCancelled {
				logger.Info("迁移任务状态变更，停止处理", zap.Int64("task_id", taskID), zap.String("status", string(currentTask.Status)))
				return
			}

			// 获取待处理记录（已预加载图片数据）
			records, err := s.migrationRepo.GetPendingFileRecords(ctx, taskID, 100)
			if err != nil {
				logger.Error("获取待处理记录失败", zap.Error(err))
				return
			}

			if len(records) == 0 {
				return // 没有更多记录，退出
			}

			// 将记录发送到 channel，不需要等待所有记录处理完成
			for _, record := range records {
				select {
				case <-ctx.Done():
					return
				case recordChan <- record: // 发送到 channel
				}
			}

			// 通知进度（每批次通知一次）
			s.notifyProgress(ctx)
		}
	}()

	// 等待生产者完成
	producerWg.Wait()

	// 等待所有消费者完成
	consumerWg.Wait()

	// 检查最终状态
	finalTask, err := s.migrationRepo.GetByIDWithStats(ctx, taskID)
	// 只有不是暂停或取消的情况下才更新最终状态
	if err == nil && finalTask.Status == model.StorageMigrationRunning {
		completedAt := time.Now()

		// 根据是否有失败文件决定最终状态
		if finalTask.FailedFiles > 0 {
			// 有失败文件，标记为失败状态
			finalTask.Status = model.StorageMigrationFailed
			finalTask.CompletedAt = &completedAt
			if err := s.migrationRepo.Update(ctx, finalTask); err != nil {
				logger.Error("更新任务失败状态失败", zap.Int64("task_id", taskID), zap.Error(err))
			}
			logger.Info("存储迁移完成，但存在失败文件",
				zap.Int64("task_id", taskID),
				zap.Int("failed_files", finalTask.FailedFiles))
		} else {
			// 无失败文件，自动删除任务和所有文件记录
			if err := s.migrationRepo.Delete(ctx, taskID); err != nil {
				logger.Error("自动删除迁移任务失败", zap.Int64("task_id", taskID), zap.Error(err))
			} else {
				logger.Info("迁移任务完成且无失败文件，已自动删除", zap.Int64("task_id", taskID))
			}
		}
	}

	// 最终通知
	s.notifyProgress(ctx)
}

// updateFileRecordStatus 更新文件记录状态
func (s *storageMigrationService) updateFileRecordStatus(ctx context.Context, taskID, recordID int64, status string, errMsg *string) {
	defer s.notifyProgress(ctx)
	if err := s.migrationRepo.UpdateFileRecordStatus(ctx, recordID, status, errMsg); err != nil {
		logger.Error("更新文件记录状态失败",
			zap.Int64("task_id", taskID),
			zap.Int64("record_id", recordID),
			zap.String("status", status),
			zap.Error(err))
	}

	if status == model.MigrationFileRecordFailed && errMsg != nil {
		logger.Error("文件迁移失败",
			zap.Int64("task_id", taskID),
			zap.Int64("record_id", recordID),
			zap.String("error", *errMsg))
	}
}

// migrateFile 迁移单个文件
func (s *storageMigrationService) migrateFile(ctx context.Context, task *model.StorageMigrationTask, record *model.MigrationFileRecord) {
	// 使用预加载的图片数据
	if record.Image == nil {
		errMsg := fmt.Sprintf("图片数据为空")
		s.updateFileRecordStatus(ctx, task.ID, record.ID, model.MigrationFileRecordFailed, &errMsg)
		return
	}

	image := record.Image

	var sourcePath string
	var sourceStorageId model.StorageId

	if task.MigrationType == model.MigrationTypeOriginal {
		sourcePath = image.StoragePath
		sourceStorageId = image.StorageId
	} else {
		sourcePath = image.ThumbnailPath
		sourceStorageId = image.ThumbnailStorageId
	}

	// 验证源存储匹配
	if sourceStorageId != task.SourceStorageId {
		// 跳过不匹配的文件
		s.updateFileRecordStatus(ctx, task.ID, record.ID, model.MigrationFileRecordSuccess, nil)
		return
	}

	// 下载文件
	reader, err := s.storageManager.Download(ctx, task.SourceStorageId, sourcePath)
	if err != nil {
		errMsg := fmt.Sprintf("下载文件失败: %v", err)
		s.updateFileRecordStatus(ctx, task.ID, record.ID, model.MigrationFileRecordFailed, &errMsg)
		return
	}
	defer reader.Close()

	// 上传到目标存储
	targetCtx := context.WithValue(ctx, storage.OverrideStorageType, task.TargetStorageId)
	_, err = s.storageManager.UploadToDefaultStorage(targetCtx, reader, sourcePath)
	if err != nil {
		errMsg := fmt.Sprintf("上传文件失败: %v", err)
		s.updateFileRecordStatus(ctx, task.ID, record.ID, model.MigrationFileRecordFailed, &errMsg)
		return
	}

	// 更新数据库中的存储 ID
	if task.MigrationType == model.MigrationTypeOriginal {
		image.StorageId = task.TargetStorageId
	} else {
		image.ThumbnailStorageId = task.TargetStorageId
	}

	if err := database.GetDB(ctx).Save(image).Error; err != nil {
		errMsg := fmt.Sprintf("更新图片记录失败: %v", err)
		s.updateFileRecordStatus(ctx, task.ID, record.ID, model.MigrationFileRecordFailed, &errMsg)
		return
	}

	// 删除源文件（如果配置了）
	if task.DeleteSourceAfterMigration {
		if err := s.storageManager.Delete(ctx, task.SourceStorageId, sourcePath); err != nil {
			logger.Warn("删除源文件失败", zap.String("path", sourcePath), zap.Error(err))
		}
	}

	logger.Info(fmt.Sprintf("文件迁移成功： %s -> %s %s", string(sourceStorageId), string(task.TargetStorageId), sourcePath))
	// 更新记录状态
	s.updateFileRecordStatus(ctx, task.ID, record.ID, model.MigrationFileRecordSuccess, nil)
}

// notifyProgress 通知进度
func (s *storageMigrationService) notifyProgress(ctx context.Context) {
	if s.notifier == nil {
		return
	}

	statusVO, err := s.GetMigrationStatusVO(ctx)
	if err != nil {
		logger.Error("获取迁移状态失败", zap.Error(err))
		return
	}

	s.notifier.NotifyMigrationProgress(statusVO)
}
