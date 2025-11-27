package repository

import (
	"context"

	"gallary/server/internal/model"
	"gallary/server/pkg/database"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SettingRepository 设置仓库接口
type SettingRepository interface {
	GetByKey(ctx context.Context, key string) (*model.Setting, error)
	GetByCategory(ctx context.Context, category string) ([]model.Setting, error)
	GetAll(ctx context.Context) ([]model.Setting, error)
	Upsert(ctx context.Context, setting *model.Setting) error
	BatchUpsert(ctx context.Context, settings []model.Setting) error
	Delete(ctx context.Context, key string) error
	Count(ctx context.Context) (int64, error)
}

type settingRepository struct {
}

// NewSettingRepository 创建设置仓库实例
func NewSettingRepository() SettingRepository {
	return &settingRepository{}
}

// GetByKey 根据键名获取设置
func (r *settingRepository) GetByKey(ctx context.Context, key string) (*model.Setting, error) {
	var setting model.Setting
	err := database.GetDB(ctx).WithContext(ctx).
		Where("\"key\" = ?", key).
		First(&setting).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 未找到返回 nil
		}
		return nil, err
	}

	return &setting, nil
}

// GetByCategory 根据分类获取设置列表
func (r *settingRepository) GetByCategory(ctx context.Context, category string) ([]model.Setting, error) {
	var settings []model.Setting
	err := database.GetDB(ctx).WithContext(ctx).
		Where("category = ?", category).
		Order("\"key\" ASC").
		Find(&settings).Error

	if err != nil {
		return nil, err
	}

	return settings, nil
}

// GetAll 获取所有设置
func (r *settingRepository) GetAll(ctx context.Context) ([]model.Setting, error) {
	var settings []model.Setting
	err := database.GetDB(ctx).WithContext(ctx).
		Order("category ASC, \"key\" ASC").
		Find(&settings).Error

	if err != nil {
		return nil, err
	}

	return settings, nil
}

// Upsert 创建或更新设置（如果存在则更新，不存在则创建）
func (r *settingRepository) Upsert(ctx context.Context, setting *model.Setting) error {
	return database.GetDB(ctx).WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "\"key\""}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "value_type", "category", "updated_at"}),
		}).
		Create(setting).Error
}

// BatchUpsert 批量创建或更新设置
func (r *settingRepository) BatchUpsert(ctx context.Context, settings []model.Setting) error {
	if len(settings) == 0 {
		return nil
	}

	return database.GetDB(ctx).WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "\"key\""}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "value_type", "category", "updated_at"}),
		}).
		Create(&settings).Error
}

// Delete 删除设置
func (r *settingRepository) Delete(ctx context.Context, key string) error {
	return database.GetDB(ctx).WithContext(ctx).
		Where("\"key\" = ?", key).
		Delete(&model.Setting{}).Error
}

// Count 获取设置总数
func (r *settingRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := database.GetDB(ctx).WithContext(ctx).
		Model(&model.Setting{}).
		Count(&count).Error
	return count, err
}
