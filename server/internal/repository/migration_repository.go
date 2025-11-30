package repository

import (
	"context"

	"gallary/server/internal/model"
	"gallary/server/pkg/database"

	"gorm.io/gorm"
)

// MigrationRepository 迁移仓库接口
type MigrationRepository interface {
	Create(ctx context.Context, task *model.MigrationTask) error
	GetByID(ctx context.Context, id int64) (*model.MigrationTask, error)
	GetActive(ctx context.Context) (*model.MigrationTask, error)
	Update(ctx context.Context, task *model.MigrationTask) error
	UpdateStatus(ctx context.Context, id int64, status model.MigrationStatus, errorMsg *string) error
	IncrementProcessed(ctx context.Context, id int64) error
	List(ctx context.Context, page, pageSize int) ([]model.MigrationTask, int64, error)
}

type migrationRepository struct {
}

// NewMigrationRepository 创建迁移仓库实例
func NewMigrationRepository() MigrationRepository {
	return &migrationRepository{}
}

// Create 创建迁移任务
func (r *migrationRepository) Create(ctx context.Context, task *model.MigrationTask) error {
	return database.GetDB(ctx).WithContext(ctx).Create(task).Error
}

// GetByID 根据ID获取迁移任务
func (r *migrationRepository) GetByID(ctx context.Context, id int64) (*model.MigrationTask, error) {
	var task model.MigrationTask
	err := database.GetDB(ctx).WithContext(ctx).
		Where("id = ?", id).
		First(&task).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &task, nil
}

// GetActive 获取当前活跃的迁移任务（pending 或 running）
func (r *migrationRepository) GetActive(ctx context.Context) (*model.MigrationTask, error) {
	var task model.MigrationTask
	err := database.GetDB(ctx).WithContext(ctx).
		Where("status IN ?", []model.MigrationStatus{
			model.MigrationStatusPending,
			model.MigrationStatusRunning,
		}).
		Order("created_at DESC").
		First(&task).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &task, nil
}

// Update 更新迁移任务
func (r *migrationRepository) Update(ctx context.Context, task *model.MigrationTask) error {
	return database.GetDB(ctx).WithContext(ctx).Save(task).Error
}

// UpdateStatus 更新迁移任务状态
func (r *migrationRepository) UpdateStatus(ctx context.Context, id int64, status model.MigrationStatus, errorMsg *string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
	}
	if errorMsg != nil {
		updates["error_message"] = *errorMsg
	}

	return database.GetDB(ctx).WithContext(ctx).
		Model(&model.MigrationTask{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// IncrementProcessed 增加已处理文件数
func (r *migrationRepository) IncrementProcessed(ctx context.Context, id int64) error {
	return database.GetDB(ctx).WithContext(ctx).
		Model(&model.MigrationTask{}).
		Where("id = ?", id).
		UpdateColumn("processed_files", gorm.Expr("processed_files + 1")).Error
}

// List 获取迁移任务列表
func (r *migrationRepository) List(ctx context.Context, page, pageSize int) ([]model.MigrationTask, int64, error) {
	var tasks []model.MigrationTask
	var total int64

	db := database.GetDB(ctx).WithContext(ctx).Model(&model.MigrationTask{})

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := db.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&tasks).Error

	if err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}
