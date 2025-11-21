package repository

import (
	"context"
	"fmt"
	"gallary/server/pkg/database"

	"gorm.io/gorm"

	"gallary/server/internal/model"
)

// ImageRepository 图片仓库接口
type ImageRepository interface {
	Create(ctx context.Context, image *model.Image) error
	FindByID(ctx context.Context, id int64) (*model.Image, error)
	FindByHash(ctx context.Context, hash string) (*model.Image, error)
	List(ctx context.Context, page, pageSize int) ([]*model.Image, int64, error)
	Update(ctx context.Context, image *model.Image) error
	Delete(ctx context.Context, id int64) error
	DeleteBatch(ctx context.Context, ids []int64) error
	FindByIDs(ctx context.Context, ids []int64) ([]*model.Image, error)
	Search(ctx context.Context, params *SearchParams) ([]*model.Image, int64, error)
	Restore(ctx context.Context, id int64) error // 恢复逻辑删除的记录

	// 元数据相关方法
	GetMetadata(ctx context.Context, imageID int64) ([]model.ImageMetadata, error)
	CreateMetadata(ctx context.Context, metadata *model.ImageMetadata) error
	UpdateMetadata(ctx context.Context, metadata *model.ImageMetadata) error
	DeleteMetadata(ctx context.Context, metadataID int64) error

	// 标签相关方法
	FindTagByName(ctx context.Context, name string) (*model.Tag, error)
	CreateTag(ctx context.Context, tag *model.Tag) error
	UpdateImageTags(ctx context.Context, imageID int64, tagIDs []int64) error
}

// SearchParams 搜索参数
type SearchParams struct {
	Keyword      string
	StartDate    *string
	EndDate      *string
	Tags         []int64
	LocationName string
	CameraModel  string
	Page         int
	PageSize     int
}

type imageRepository struct {
}

// NewImageRepository 创建图片仓库实例
func NewImageRepository() ImageRepository {
	return &imageRepository{}
}

// Create 创建图片记录
func (r *imageRepository) Create(ctx context.Context, image *model.Image) error {
	return database.GetDB(ctx).WithContext(ctx).Create(image).Error
}

// FindByID 根据ID查找图片
func (r *imageRepository) FindByID(ctx context.Context, id int64) (*model.Image, error) {
	var image model.Image
	err := database.GetDB(ctx).WithContext(ctx).
		Preload("Tags").
		Preload("Metadata").
		First(&image, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("图片不存在")
		}
		return nil, err
	}

	return &image, nil
}

// FindByUUID 根据UUID查找图片
func (r *imageRepository) FindByUUID(ctx context.Context, uuid string) (*model.Image, error) {
	var image model.Image
	err := database.GetDB(ctx).WithContext(ctx).
		Preload("Tags").
		Preload("Metadata").
		Where("uuid = ?", uuid).
		First(&image).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("图片不存在")
		}
		return nil, err
	}

	return &image, nil
}

// FindByHash 根据Hash查找图片（用于去重），包括逻辑删除的记录
func (r *imageRepository) FindByHash(ctx context.Context, hash string) (*model.Image, error) {
	var image model.Image
	err := database.GetDB(ctx).WithContext(ctx).
		Unscoped(). // 包含逻辑删除的记录
		Where("file_hash = ?", hash).
		First(&image).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 未找到返回 nil 而不是错误
		}
		return nil, err
	}

	return &image, nil
}

// List 分页获取图片列表
func (r *imageRepository) List(ctx context.Context, page, pageSize int) ([]*model.Image, int64, error) {
	var images []*model.Image
	var total int64

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 查询总数
	if err := database.GetDB(ctx).WithContext(ctx).Model(&model.Image{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询数据
	err := database.GetDB(ctx).WithContext(ctx).
		Preload("Tags").
		Order("taken_at DESC, created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&images).Error

	if err != nil {
		return nil, 0, err
	}

	return images, total, nil
}

// Update 更新图片信息
func (r *imageRepository) Update(ctx context.Context, image *model.Image) error {
	return database.GetDB(ctx).Save(image).Error
}

// Delete 删除图片（软删除）
func (r *imageRepository) Delete(ctx context.Context, id int64) error {
	return database.GetDB(ctx).Delete(&model.Image{}, id).Error
}

// DeleteBatch 批量删除图片
func (r *imageRepository) DeleteBatch(ctx context.Context, ids []int64) error {
	return database.GetDB(ctx).Delete(&model.Image{}, ids).Error
}

// FindByIDs 根据ID列表查找图片
func (r *imageRepository) FindByIDs(ctx context.Context, ids []int64) ([]*model.Image, error) {
	var images []*model.Image
	err := database.GetDB(ctx).
		Where("id IN ?", ids).
		Find(&images).Error

	if err != nil {
		return nil, err
	}

	return images, nil
}

// Search 搜索图片
func (r *imageRepository) Search(ctx context.Context, params *SearchParams) ([]*model.Image, int64, error) {
	var images []*model.Image
	var total int64

	query := database.GetDB(ctx).Model(&model.Image{})

	// 关键词搜索（搜索文件名）
	if params.Keyword != "" {
		query = query.Where("original_name ILIKE ?", "%"+params.Keyword+"%")
	}

	// 时间范围搜索
	if params.StartDate != nil {
		query = query.Where("taken_at >= ?", *params.StartDate)
	}
	if params.EndDate != nil {
		query = query.Where("taken_at <= ?", *params.EndDate)
	}

	// 地点搜索
	if params.LocationName != "" {
		query = query.Where("location_name ILIKE ?", "%"+params.LocationName+"%")
	}

	// 相机型号搜索
	if params.CameraModel != "" {
		query = query.Where("camera_model ILIKE ?", "%"+params.CameraModel+"%")
	}

	// 标签搜索
	if len(params.Tags) > 0 {
		query = query.Joins("JOIN image_tags ON images.id = image_tags.image_id").
			Where("image_tags.tag_id IN ?", params.Tags).
			Distinct("images.id")
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (params.Page - 1) * params.PageSize
	err := query.Preload("Tags").
		Order("created_at DESC").
		Limit(params.PageSize).
		Offset(offset).
		Find(&images).Error

	if err != nil {
		return nil, 0, err
	}

	return images, total, nil
}

// Restore 恢复逻辑删除的记录
func (r *imageRepository) Restore(ctx context.Context, id int64) error {
	return database.GetDB(ctx).Unscoped().Model(&model.Image{}).
		Where("id = ?", id).
		Update("deleted_at", nil).Error
}

// GetMetadata 获取图片的所有元数据
func (r *imageRepository) GetMetadata(ctx context.Context, imageID int64) ([]model.ImageMetadata, error) {
	var metadata []model.ImageMetadata
	err := database.GetDB(ctx).
		Where("image_id = ?", imageID).
		Find(&metadata).Error
	return metadata, err
}

// CreateMetadata 创建元数据
func (r *imageRepository) CreateMetadata(ctx context.Context, metadata *model.ImageMetadata) error {
	return database.GetDB(ctx).Create(metadata).Error
}

// UpdateMetadata 更新元数据
func (r *imageRepository) UpdateMetadata(ctx context.Context, metadata *model.ImageMetadata) error {
	return database.GetDB(ctx).Save(metadata).Error
}

// DeleteMetadata 删除元数据
func (r *imageRepository) DeleteMetadata(ctx context.Context, metadataID int64) error {
	return database.GetDB(ctx).Delete(&model.ImageMetadata{}, metadataID).Error
}

// FindTagByName 根据名称查找标签
func (r *imageRepository) FindTagByName(ctx context.Context, name string) (*model.Tag, error) {
	var tag model.Tag
	err := database.GetDB(ctx).
		Where("name = ?", name).
		First(&tag).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 未找到返回 nil
		}
		return nil, err
	}

	return &tag, nil
}

// CreateTag 创建标签
func (r *imageRepository) CreateTag(ctx context.Context, tag *model.Tag) error {
	return database.GetDB(ctx).Create(tag).Error
}

// UpdateImageTags 更新图片标签关联
func (r *imageRepository) UpdateImageTags(ctx context.Context, imageID int64, tagIDs []int64) error {
	return database.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 删除现有的标签关联
		if err := tx.Where("image_id = ?", imageID).Delete(&model.ImageTag{}).Error; err != nil {
			return err
		}

		// 2. 创建新的标签关联
		if len(tagIDs) > 0 {
			var imageTags []model.ImageTag
			for _, tagID := range tagIDs {
				imageTags = append(imageTags, model.ImageTag{
					ImageID: imageID,
					TagID:   tagID,
				})
			}
			if err := tx.Create(&imageTags).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
