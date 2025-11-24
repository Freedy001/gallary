package repository

import (
	"context"
	"fmt"
	"gallary/server/internal/model"
	"gallary/server/pkg/database"

	"gorm.io/gorm"
)

// ShareRepository 分享仓库接口
type ShareRepository interface {
	Create(ctx context.Context, share *model.Share, imageIDs []int64) error
	FindByID(ctx context.Context, id int64) (*model.Share, error)
	FindByCode(ctx context.Context, code string) (*model.Share, error)
	List(ctx context.Context, page, pageSize int) ([]*model.Share, int64, error)
	Update(ctx context.Context, share *model.Share) error
	Delete(ctx context.Context, id int64) error
	IncrementViewCount(ctx context.Context, id int64) error
	IncrementDownloadCount(ctx context.Context, id int64) error
	GetImages(ctx context.Context, shareID int64) ([]*model.Image, error)
	GetImagesPaginated(ctx context.Context, shareID int64, page, pageSize int) ([]*model.Image, int64, error)
}

type shareRepository struct{}

// NewShareRepository 创建分享仓库实例
func NewShareRepository() ShareRepository {
	return &shareRepository{}
}

// Create 创建分享记录并关联图片
func (r *shareRepository) Create(ctx context.Context, share *model.Share, imageIDs []int64) error {
	return database.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(share).Error; err != nil {
			return err
		}

		if len(imageIDs) > 0 {
			var shareImages []model.ShareImage
			for i, imgID := range imageIDs {
				shareImages = append(shareImages, model.ShareImage{
					ShareID:   share.ID,
					ImageID:   imgID,
					SortOrder: i,
				})
			}
			if err := tx.Create(&shareImages).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// FindByID 根据ID查找分享
func (r *shareRepository) FindByID(ctx context.Context, id int64) (*model.Share, error) {
	var share model.Share
	err := database.GetDB(ctx).WithContext(ctx).First(&share, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("分享不存在")
		}
		return nil, err
	}
	return &share, nil
}

// FindByCode 根据分享码查找分享
func (r *shareRepository) FindByCode(ctx context.Context, code string) (*model.Share, error) {
	var share model.Share
	err := database.GetDB(ctx).WithContext(ctx).Where("share_code = ?", code).First(&share).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("分享不存在")
		}
		return nil, err
	}
	return &share, nil
}

// List 分页获取分享列表
func (r *shareRepository) List(ctx context.Context, page, pageSize int) ([]*model.Share, int64, error) {
	var shares []*model.Share
	var total int64

	offset := (page - 1) * pageSize

	if err := database.GetDB(ctx).WithContext(ctx).Model(&model.Share{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := database.GetDB(ctx).WithContext(ctx).
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&shares).Error

	if err != nil {
		return nil, 0, err
	}

	return shares, total, nil
}

// Update 更新分享信息
func (r *shareRepository) Update(ctx context.Context, share *model.Share) error {
	return database.GetDB(ctx).Save(share).Error
}

// Delete 删除分享
func (r *shareRepository) Delete(ctx context.Context, id int64) error {
	return database.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除关联表数据
		if err := tx.Where("share_id = ?", id).Delete(&model.ShareImage{}).Error; err != nil {
			return err
		}
		// 删除分享记录
		return tx.Delete(&model.Share{}, id).Error
	})
}

// IncrementViewCount 增加查看次数
func (r *shareRepository) IncrementViewCount(ctx context.Context, id int64) error {
	return database.GetDB(ctx).Model(&model.Share{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error
}

// IncrementDownloadCount 增加下载次数
func (r *shareRepository) IncrementDownloadCount(ctx context.Context, id int64) error {
	return database.GetDB(ctx).Model(&model.Share{}).Where("id = ?", id).
		UpdateColumn("download_count", gorm.Expr("download_count + ?", 1)).Error
}

// GetImages 获取分享包含的图片
func (r *shareRepository) GetImages(ctx context.Context, shareID int64) ([]*model.Image, error) {
	var images []*model.Image
	err := database.GetDB(ctx).WithContext(ctx).
		Joins("JOIN share_images ON images.id = share_images.image_id").
		Where("share_images.share_id = ?", shareID).
		Order("share_images.sort_order ASC").
		Preload("Tags").
		Preload("Metadata").
		Find(&images).Error

	return images, err
}

// GetImagesPaginated 分页获取分享包含的图片
func (r *shareRepository) GetImagesPaginated(ctx context.Context, shareID int64, page, pageSize int) ([]*model.Image, int64, error) {
	var images []*model.Image
	var total int64

	offset := (page - 1) * pageSize

	// 先获取总数
	if err := database.GetDB(ctx).WithContext(ctx).
		Model(&model.ShareImage{}).
		Where("share_id = ?", shareID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页获取图片
	err := database.GetDB(ctx).WithContext(ctx).
		Joins("JOIN share_images ON images.id = share_images.image_id").
		Where("share_images.share_id = ?", shareID).
		Order("share_images.sort_order ASC,images.taken_at DESC").
		Limit(pageSize).
		Offset(offset).
		Preload("Tags").
		Preload("Metadata").
		Find(&images).Error

	return images, total, err
}
