package repository

import (
	"context"
	"fmt"

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
	Search(ctx context.Context, params *SearchParams) ([]*model.Image, int64, error)
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
	db *gorm.DB
}

// NewImageRepository 创建图片仓库实例
func NewImageRepository(db *gorm.DB) ImageRepository {
	return &imageRepository{db: db}
}

// Create 创建图片记录
func (r *imageRepository) Create(ctx context.Context, image *model.Image) error {
	return r.db.WithContext(ctx).Create(image).Error
}

// FindByID 根据ID查找图片
func (r *imageRepository) FindByID(ctx context.Context, id int64) (*model.Image, error) {
	var image model.Image
	err := r.db.WithContext(ctx).
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
	err := r.db.WithContext(ctx).
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

// FindByHash 根据Hash查找图片（用于去重）
func (r *imageRepository) FindByHash(ctx context.Context, hash string) (*model.Image, error) {
	var image model.Image
	err := r.db.WithContext(ctx).
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
	if err := r.db.WithContext(ctx).Model(&model.Image{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询数据
	err := r.db.WithContext(ctx).
		Preload("Tags").
		Order("created_at DESC").
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
	return r.db.WithContext(ctx).Save(image).Error
}

// Delete 删除图片（软删除）
func (r *imageRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Image{}, id).Error
}

// Search 搜索图片
func (r *imageRepository) Search(ctx context.Context, params *SearchParams) ([]*model.Image, int64, error) {
	var images []*model.Image
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Image{})

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
