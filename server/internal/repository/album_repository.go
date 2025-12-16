package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gallary/server/internal/model"
	"gallary/server/pkg/database"
)

// AlbumRepository 相册仓库接口
type AlbumRepository interface {
	Create(ctx context.Context, album *model.Tag) error
	FindByID(ctx context.Context, id int64) (*model.Tag, error)
	List(ctx context.Context, page, pageSize int) ([]*model.Tag, int64, error)
	Update(ctx context.Context, album *model.Tag) error
	Delete(ctx context.Context, id int64) error

	// 图片关联
	AddImages(ctx context.Context, albumID int64, imageIDs []int64) error
	RemoveImages(ctx context.Context, albumID int64, imageIDs []int64) error
	GetImages(ctx context.Context, albumID int64, page, pageSize int) ([]*model.Image, int64, error)
	GetImageCount(ctx context.Context, albumID int64) (int64, error)
	GetImageCounts(ctx context.Context, albumIDs []int64) (map[int64]int64, error)

	// 封面
	GetCoverImage(ctx context.Context, coverImageID int64) (*model.Image, error)
	GetFirstImages(ctx context.Context, albumIDs []int64) (map[int64]*model.Image, error)
}

type albumRepository struct{}

// NewAlbumRepository 创建相册仓库实例
func NewAlbumRepository() AlbumRepository {
	return &albumRepository{}
}

// Create 创建相册
func (r *albumRepository) Create(ctx context.Context, album *model.Tag) error {
	album.Type = model.TagTypeAlbum
	return database.GetDB(ctx).WithContext(ctx).Create(album).Error
}

// FindByID 根据ID查找相册
func (r *albumRepository) FindByID(ctx context.Context, id int64) (*model.Tag, error) {
	var album model.Tag
	err := database.GetDB(ctx).WithContext(ctx).
		Where("id = ? AND type = ?", id, model.TagTypeAlbum).
		First(&album).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("相册不存在")
		}
		return nil, err
	}

	return &album, nil
}

// List 分页获取相册列表
func (r *albumRepository) List(ctx context.Context, page, pageSize int) ([]*model.Tag, int64, error) {
	var albums []*model.Tag
	var total int64

	offset := (page - 1) * pageSize

	db := database.GetDB(ctx).WithContext(ctx).Model(&model.Tag{}).
		Where("type = ?", model.TagTypeAlbum)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 按 metadata 中的 sort_order 排序，然后按创建时间降序
	err := db.Order("COALESCE((metadata->>'sort_order')::int, 0) ASC, created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&albums).Error

	if err != nil {
		return nil, 0, err
	}

	return albums, total, nil
}

// Update 更新相册信息
func (r *albumRepository) Update(ctx context.Context, album *model.Tag) error {
	return database.GetDB(ctx).WithContext(ctx).Save(album).Error
}

// Delete 删除相册
func (r *albumRepository) Delete(ctx context.Context, id int64) error {
	return database.GetDB(ctx).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除关联关系
		if err := tx.Where("tag_id = ?", id).Delete(&model.ImageTag{}).Error; err != nil {
			return err
		}
		// 删除相册
		return tx.Where("id = ? AND type = ?", id, model.TagTypeAlbum).Delete(&model.Tag{}).Error
	})
}

// AddImages 添加图片到相册
func (r *albumRepository) AddImages(ctx context.Context, albumID int64, imageIDs []int64) error {
	if len(imageIDs) == 0 {
		return nil
	}

	// 获取当前最大排序值
	var maxSort int
	database.GetDB(ctx).WithContext(ctx).Model(&model.ImageTag{}).
		Where("tag_id = ?", albumID).
		Select("COALESCE(MAX(sort_order), 0)").
		Scan(&maxSort)

	var imageTags []model.ImageTag
	for i, imgID := range imageIDs {
		imageTags = append(imageTags, model.ImageTag{
			ImageID:   imgID,
			TagID:     albumID,
			SortOrder: maxSort + i + 1,
		})
	}

	// 使用 ON CONFLICT DO NOTHING 避免重复
	return database.GetDB(ctx).WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "image_id"}, {Name: "tag_id"}},
		DoNothing: true,
	}).Create(&imageTags).Error
}

// RemoveImages 从相册移除图片
func (r *albumRepository) RemoveImages(ctx context.Context, albumID int64, imageIDs []int64) error {
	return database.GetDB(ctx).WithContext(ctx).
		Where("tag_id = ? AND image_id IN ?", albumID, imageIDs).
		Delete(&model.ImageTag{}).Error
}

// GetImages 分页获取相册内图片
func (r *albumRepository) GetImages(ctx context.Context, albumID int64, page, pageSize int) ([]*model.Image, int64, error) {
	var images []*model.Image
	var total int64

	offset := (page - 1) * pageSize

	// 先统计总数
	countErr := database.GetDB(ctx).WithContext(ctx).Model(&model.ImageTag{}).
		Where("tag_id = ?", albumID).
		Count(&total).Error
	if countErr != nil {
		return nil, 0, countErr
	}

	// 查询图片数据
	err := database.GetDB(ctx).WithContext(ctx).
		Table("images").
		Joins("JOIN image_tags ON images.id = image_tags.image_id").
		Where("image_tags.tag_id = ? AND images.deleted_at IS NULL", albumID).
		Order("image_tags.sort_order ASC, images.taken_at DESC").
		Preload("Tags").
		Limit(pageSize).
		Offset(offset).
		Find(&images).Error

	if err != nil {
		return nil, 0, err
	}

	return images, total, nil
}

// GetImageCount 获取相册图片数量
func (r *albumRepository) GetImageCount(ctx context.Context, albumID int64) (int64, error) {
	var count int64
	err := database.GetDB(ctx).WithContext(ctx).Model(&model.ImageTag{}).
		Joins("JOIN images ON images.id = image_tags.image_id").
		Where("image_tags.tag_id = ? AND images.deleted_at IS NULL", albumID).
		Count(&count).Error
	return count, err
}

// GetImageCounts 批量获取相册图片数量
func (r *albumRepository) GetImageCounts(ctx context.Context, albumIDs []int64) (map[int64]int64, error) {
	if len(albumIDs) == 0 {
		return make(map[int64]int64), nil
	}

	type countResult struct {
		TagID int64
		Count int64
	}

	var results []countResult
	err := database.GetDB(ctx).WithContext(ctx).
		Table("image_tags").
		Select("image_tags.tag_id, COUNT(*) as count").
		Joins("JOIN images ON images.id = image_tags.image_id").
		Where("image_tags.tag_id IN ? AND images.deleted_at IS NULL", albumIDs).
		Group("image_tags.tag_id").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	countMap := make(map[int64]int64)
	for _, r := range results {
		countMap[r.TagID] = r.Count
	}

	return countMap, nil
}

// GetCoverImage 获取封面图片
func (r *albumRepository) GetCoverImage(ctx context.Context, coverImageID int64) (*model.Image, error) {
	var image model.Image
	err := database.GetDB(ctx).WithContext(ctx).First(&image, coverImageID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &image, nil
}

// GetFirstImages 批量获取相册的第一张图片（作为默认封面）
func (r *albumRepository) GetFirstImages(ctx context.Context, albumIDs []int64) (map[int64]*model.Image, error) {
	if len(albumIDs) == 0 {
		return make(map[int64]*model.Image), nil
	}

	// 使用子查询获取每个相册的第一张图片
	type firstImageResult struct {
		TagID   int64
		ImageID int64
	}

	var results []firstImageResult
	subQuery := database.GetDB(ctx).WithContext(ctx).
		Table("image_tags").
		Select("tag_id, MIN(image_id) as image_id").
		Where("tag_id IN ?", albumIDs).
		Group("tag_id")

	err := subQuery.Scan(&results).Error
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return make(map[int64]*model.Image), nil
	}

	// 获取图片详情
	imageIDs := make([]int64, 0, len(results))
	tagImageMap := make(map[int64]int64)
	for _, r := range results {
		imageIDs = append(imageIDs, r.ImageID)
		tagImageMap[r.ImageID] = r.TagID
	}

	var images []*model.Image
	err = database.GetDB(ctx).WithContext(ctx).
		Where("id IN ? AND deleted_at IS NULL", imageIDs).
		Find(&images).Error
	if err != nil {
		return nil, err
	}

	// 构建结果映射
	resultMap := make(map[int64]*model.Image)
	for _, img := range images {
		if tagID, ok := tagImageMap[img.ID]; ok {
			resultMap[tagID] = img
		}
	}

	return resultMap, nil
}
