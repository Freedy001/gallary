package repository

import (
	"context"

	"gallary/server/internal/model"
	"gallary/server/pkg/database"

	"gorm.io/gorm"
)

// StorageMigrationRepository 存储迁移仓库接口
type StorageMigrationRepository interface {
	// 任务相关
	Create(ctx context.Context, task *model.StorageMigrationTask) error
	// 查询（带统计）
	List(ctx context.Context) ([]*model.StorageMigrationTask, error)
	GetByID(ctx context.Context, id int64) (*model.StorageMigrationTask, error)
	GetByIDWithStats(ctx context.Context, id int64) (*model.StorageMigrationTask, error)
	Update(ctx context.Context, task *model.StorageMigrationTask) error
	UpdateStatus(ctx context.Context, id int64, status model.StorageMigrationStatus, errorMsg *string) error
	Delete(ctx context.Context, id int64) error

	// 文件记录相关
	CreateFileRecords(ctx context.Context, records []*model.MigrationFileRecord) error
	GetPendingFileRecords(ctx context.Context, taskID int64, limit int) ([]*model.MigrationFileRecord, error)
	GetFailedFileRecords(ctx context.Context, taskID int64, page, pageSize int) ([]*model.MigrationFileRecord, int64, error)
	UpdateFileRecordStatus(ctx context.Context, id int64, status string, errorMsg *string) error
	GetTotalSizeByTaskID(ctx context.Context, taskID int64, migrationType model.MigrationType) (int64, error)

	ResetFailedFileRecords(ctx context.Context, taskID int64) (int64, error)
	DeleteFileRecordsByTaskID(ctx context.Context, taskID int64) error
}

type storageMigrationRepository struct{}

// NewStorageMigrationRepository 创建存储迁移仓库实例
func NewStorageMigrationRepository() StorageMigrationRepository {
	return &storageMigrationRepository{}
}

// Create 创建迁移任务
func (r *storageMigrationRepository) Create(ctx context.Context, task *model.StorageMigrationTask) error {
	return database.GetDB(ctx).WithContext(ctx).Create(task).Error
}

// GetByID 根据ID获取迁移任务（不含统计）
func (r *storageMigrationRepository) GetByID(ctx context.Context, id int64) (*model.StorageMigrationTask, error) {
	var task model.StorageMigrationTask
	err := database.GetDB(ctx).WithContext(ctx).First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetByIDWithStats 根据ID获取迁移任务（含统计）
func (r *storageMigrationRepository) GetByIDWithStats(ctx context.Context, id int64) (*model.StorageMigrationTask, error) {
	task, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	task.ProcessedFiles, task.FailedFiles, _ = r.GetFileRecordStats(ctx, id)
	return task, nil
}

// Update 更新迁移任务
func (r *storageMigrationRepository) Update(ctx context.Context, task *model.StorageMigrationTask) error {
	return database.GetDB(ctx).WithContext(ctx).Save(task).Error
}

// UpdateStatus 更新任务状态
func (r *storageMigrationRepository) UpdateStatus(ctx context.Context, id int64, status model.StorageMigrationStatus, errorMsg *string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if errorMsg != nil {
		updates["error_message"] = *errorMsg
	}
	return database.GetDB(ctx).WithContext(ctx).
		Model(&model.StorageMigrationTask{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// GetActive 获取当前活跃的迁移任务（优先返回 running 状态）

// ListActive 获取所有活跃的迁移任务

// List 获取所有活跃的迁移任务（包含统计数据）
func (r *storageMigrationRepository) List(ctx context.Context) ([]*model.StorageMigrationTask, error) {
	var tasks []*model.StorageMigrationTask

	err := database.GetDB(ctx).WithContext(ctx).
		Order("created_at DESC").
		Find(&tasks).Error
	if err != nil {
		return nil, err
	}

	// 填充统计数据
	for _, task := range tasks {
		task.ProcessedFiles, task.FailedFiles, _ = r.GetFileRecordStats(ctx, task.ID)
	}

	return tasks, nil
}

// ListHistory 获取迁移历史记录

// GetFileRecordStats 获取文件记录统计（已处理数和失败数）
func (r *storageMigrationRepository) GetFileRecordStats(ctx context.Context, taskID int64) (processed int, failed int, err error) {
	type stats struct {
		Status string
		Count  int
	}
	var results []stats

	err = database.GetDB(ctx).WithContext(ctx).
		Model(&model.MigrationFileRecord{}).
		Select("status, COUNT(*) as count").
		Where("task_id = ?", taskID).
		Group("status").
		Scan(&results).Error
	if err != nil {
		return 0, 0, err
	}

	for _, r := range results {
		switch r.Status {
		case model.MigrationFileRecordSuccess:
			processed = r.Count
		case model.MigrationFileRecordFailed:
			failed = r.Count
		}
	}

	return processed, failed, nil
}

// CreateFileRecords 批量创建文件记录
func (r *storageMigrationRepository) CreateFileRecords(ctx context.Context, records []*model.MigrationFileRecord) error {
	if len(records) == 0 {
		return nil
	}
	return database.GetDB(ctx).WithContext(ctx).CreateInBatches(records, 100).Error
}

// GetPendingFileRecords 获取待处理的文件记录并原子更新为 in_progress 状态
func (r *storageMigrationRepository) GetPendingFileRecords(ctx context.Context, taskID int64, limit int) ([]*model.MigrationFileRecord, error) {
	var records []*model.MigrationFileRecord

	// 使用事务确保原子操作
	err := database.GetDB(ctx).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先查询待处理的记录 ID（加锁防止并发）
		var ids []int64
		if err := tx.Model(&model.MigrationFileRecord{}).
			Where("task_id = ? AND status = ?", taskID, model.MigrationFileRecordPending).
			Limit(limit).
			Pluck("id", &ids).Error; err != nil {
			return err
		}

		if len(ids) == 0 {
			return nil
		}

		// 批量更新状态为 in_progress
		if err := tx.Model(&model.MigrationFileRecord{}).
			Where("id IN ?", ids).
			Update("status", model.MigrationFileRecordInProgress).Error; err != nil {
			return err
		}

		// 获取完整记录（包含预加载的图片数据）
		return tx.Preload("Image").
			Where("id IN ?", ids).
			Find(&records).Error
	})

	return records, err
}

// UpdateFileRecordStatus 更新文件记录状态
func (r *storageMigrationRepository) UpdateFileRecordStatus(ctx context.Context, id int64, status string, errorMsg *string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if errorMsg != nil {
		updates["error_msg"] = *errorMsg
	}
	return database.GetDB(ctx).WithContext(ctx).
		Model(&model.MigrationFileRecord{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// CountPendingFileRecords 统计待处理的文件记录数

// CountFailedFileRecords 统计失败的文件记录数

// GetFailedFileRecords 获取失败的文件记录（分页）
func (r *storageMigrationRepository) GetFailedFileRecords(ctx context.Context, taskID int64, page, pageSize int) ([]*model.MigrationFileRecord, int64, error) {
	var records []*model.MigrationFileRecord
	var total int64

	db := database.GetDB(ctx).WithContext(ctx).
		Model(&model.MigrationFileRecord{}).
		Where("task_id = ? AND status = ?", taskID, model.MigrationFileRecordFailed)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := db.Preload("Image").
		Order("updated_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&records).Error

	return records, total, err
}

// ResetFailedFileRecords 重置失败的文件记录为待处理状态（用于重试）
func (r *storageMigrationRepository) ResetFailedFileRecords(ctx context.Context, taskID int64) (int64, error) {
	result := database.GetDB(ctx).WithContext(ctx).
		Model(&model.MigrationFileRecord{}).
		Where("task_id = ? AND status IN ?", taskID, []string{model.MigrationFileRecordFailed, model.MigrationFileRecordInProgress}).
		Updates(map[string]interface{}{
			"status":    model.MigrationFileRecordPending,
			"error_msg": nil,
		})
	return result.RowsAffected, result.Error
}

// DeleteFileRecordsByTaskID 删除任务的所有文件记录
func (r *storageMigrationRepository) DeleteFileRecordsByTaskID(ctx context.Context, taskID int64) error {
	return database.GetDB(ctx).WithContext(ctx).
		Where("task_id = ?", taskID).
		Delete(&model.MigrationFileRecord{}).Error
}

// GetTotalSizeByTaskID 获取任务待迁移文件的总大小
func (r *storageMigrationRepository) GetTotalSizeByTaskID(ctx context.Context, taskID int64, migrationType model.MigrationType) (int64, error) {
	// 根据迁移类型选择正确的大小字段
	sizeField := "images.file_size"
	if migrationType == model.MigrationTypeThumbnail {
		sizeField = "images.thumbnail_size"
	}

	var totalSize int64
	err := database.GetDB(ctx).WithContext(ctx).
		Model(&model.MigrationFileRecord{}).
		Select("COALESCE(SUM("+sizeField+"), 0)").
		Joins("JOIN images ON images.id = migration_file_records.image_id").
		Where("migration_file_records.task_id = ?", taskID).
		Scan(&totalSize).Error
	return totalSize, err
}

// Delete 删除迁移任务（包含所有文件记录）
func (r *storageMigrationRepository) Delete(ctx context.Context, id int64) error {
	return database.GetDB(ctx).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除所有文件记录
		if err := tx.Where("task_id = ?", id).Delete(&model.MigrationFileRecord{}).Error; err != nil {
			return err
		}
		// 删除任务
		return tx.Delete(&model.StorageMigrationTask{}, id).Error
	})
}
