package repository

import (
	"context"
	"fmt"
	"time"

	"gallary/server/pkg/database"

	"gorm.io/gorm"

	"gallary/server/internal/model"
)

// ImageRepository 图片仓库接口
type ImageRepository interface {
	GetImagesWithLocation(ctx context.Context) ([]*model.Image, error)
	Create(ctx context.Context, image *model.Image) error
	FindByID(ctx context.Context, id int64) (*model.Image, error)
	FindByHash(ctx context.Context, hash string) (*model.Image, error)
	List(ctx context.Context, page, pageSize int) ([]*model.Image, int64, error)
	Count(ctx context.Context) (int64, error)
	Update(ctx context.Context, image *model.Image) error
	Delete(ctx context.Context, id int64) error
	DeleteBatch(ctx context.Context, ids []int64) error
	FindByIDs(ctx context.Context, ids []int64) ([]*model.Image, error)
	Search(ctx context.Context, params *model.SearchParams) ([]*model.Image, int64, error)
	SearchIDs(ctx context.Context, params *model.SearchParams, limit int) ([]int64, error)
	Restore(ctx context.Context, id int64) error // 恢复逻辑删除的记录

	// 回收站相关方法
	ListDeleted(ctx context.Context, page, pageSize int) ([]*model.Image, int64, error)
	FindDeletedByIDs(ctx context.Context, ids []int64) ([]*model.Image, error)
	RestoreBatch(ctx context.Context, ids []int64) error
	HardDeleteBatch(ctx context.Context, ids []int64) error
	FindExpiredDeleted(ctx context.Context, days int) ([]*model.Image, error)

	// 元数据相关方法
	GetMetadata(ctx context.Context, imageID int64) ([]model.ImageMetadata, error)
	CreateMetadata(ctx context.Context, metadata *model.ImageMetadata) error
	UpdateMetadata(ctx context.Context, metadata *model.ImageMetadata) error
	DeleteMetadata(ctx context.Context, metadataID int64) error

	// 标签相关方法
	FindTagByName(ctx context.Context, name string) (*model.Tag, error)
	CreateTag(ctx context.Context, tag *model.Tag) error
	UpdateImageTags(ctx context.Context, imageID int64, tagIDs []int64) error
	AddImageTags(ctx context.Context, imageID int64, tagIDs []int64) error
	FindAllNormalTags(ctx context.Context) ([]*model.Tag, error)
	SearchTags(ctx context.Context, keyword string, limit int) ([]*model.Tag, error)
	GetPopularTags(ctx context.Context, limit int) ([]*model.Tag, error)

	// 聚合相关
	GetClusters(ctx context.Context, minLat, maxLat, minLng, maxLng float64, gridSizeLat, gridSizeLng float64) ([]*model.ClusterResult, error)
	GetClusterImages(ctx context.Context, minLat, maxLat, minLng, maxLng float64, page, pageSize int) ([]*model.Image, int64, error)
	GetGeoBounds(ctx context.Context) (*model.GeoBounds, error)

	// 迁移相关方法
	CountByStorageType(ctx context.Context, storageType string) (int, error)

	// AI 评分相关方法
	FindImagesWithoutAIScore(ctx context.Context, limit int) ([]int64, error)
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

// Count 获取图片总数（不包括已删除）
func (r *imageRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := database.GetDB(ctx).WithContext(ctx).Model(&model.Image{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
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
func (r *imageRepository) Search(ctx context.Context, params *model.SearchParams) ([]*model.Image, int64, error) {
	var images []*model.Image
	var total int64

	query := database.GetDB(ctx).Model(&model.Image{})

	r.buildSearchCondition(params, query)

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (params.Page - 1) * params.PageSize

	// 如果有经纬度搜索，使用 PostGIS ST_Distance 按真实地球距离排序
	if params.Latitude != nil && params.Longitude != nil {
		// 使用 geography 类型进行精确的距离计算（米为单位）
		distanceExpr := fmt.Sprintf(
			"ST_Distance(location::geography, ST_SetSRID(ST_MakePoint(%f, %f), 4326)::geography)",
			*params.Longitude, *params.Latitude,
		)
		err := query.Preload("Tags").
			Order(distanceExpr + " ASC").
			Limit(params.PageSize).
			Offset(offset).
			Find(&images).Error

		if err != nil {
			return nil, 0, err
		}
	} else {
		err := query.Preload("Tags").
			Order("created_at DESC").
			Limit(params.PageSize).
			Offset(offset).
			Find(&images).Error

		if err != nil {
			return nil, 0, err
		}
	}

	return images, total, nil
}

// SearchIDs 搜索图片并只返回ID列表（用于混合搜索的第一步筛选）
func (r *imageRepository) SearchIDs(ctx context.Context, params *model.SearchParams, limit int) ([]int64, error) {
	var ids []int64

	query := database.GetDB(ctx).Model(&model.Image{}).Select("images.id")

	r.buildSearchCondition(params, query)

	// 限制数量并获取ID列表
	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Order("created_at DESC").Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func (r *imageRepository) buildSearchCondition(params *model.SearchParams, query *gorm.DB) {
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

	// 使用 PostGIS 进行经纬度范围搜索（基于中心点和半径）
	if params.Latitude != nil && params.Longitude != nil {
		radius := 10.0 // 默认 10 公里
		if params.Radius != nil && *params.Radius > 0 {
			radius = *params.Radius
		}
		radiusMeters := radius * 1000
		query = query.Where(
			"ST_DWithin(location::geography, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography, ?)",
			*params.Longitude, *params.Latitude, radiusMeters,
		)
	}

	// 标签搜索
	if len(params.Tags) > 0 {
		query = query.Joins("JOIN image_tags ON images.id = image_tags.image_id").
			Where("image_tags.tag_id IN ?", params.Tags).
			Distinct("images.id")
	}
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

// AddImageTags 添加图片标签关联（不删除现有标签）
func (r *imageRepository) AddImageTags(ctx context.Context, imageID int64, tagIDs []int64) error {
	if len(tagIDs) == 0 {
		return nil
	}

	return database.GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取已存在的标签关联
		var existingTags []model.ImageTag
		if err := tx.Where("image_id = ?", imageID).Find(&existingTags).Error; err != nil {
			return err
		}

		// 创建已存在标签ID的映射
		existingTagIDs := make(map[int64]bool)
		for _, tag := range existingTags {
			existingTagIDs[tag.TagID] = true
		}

		// 只添加不存在的标签关联
		var newImageTags []model.ImageTag
		for _, tagID := range tagIDs {
			if !existingTagIDs[tagID] {
				newImageTags = append(newImageTags, model.ImageTag{
					ImageID: imageID,
					TagID:   tagID,
				})
			}
		}

		// 批量创建新的标签关联
		if len(newImageTags) > 0 {
			if err := tx.Create(&newImageTags).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// FindAllNormalTags 获取所有普通标签（非相册类型）
func (r *imageRepository) FindAllNormalTags(ctx context.Context) ([]*model.Tag, error) {
	var tags []*model.Tag
	err := database.GetDB(ctx).
		Where("type <> ?", model.TagTypeAlbum).
		Order("name ASC").
		Find(&tags).Error

	if err != nil {
		return nil, err
	}

	return tags, nil
}

// SearchTags 根据关键字搜索标签
func (r *imageRepository) SearchTags(ctx context.Context, keyword string, limit int) ([]*model.Tag, error) {
	var tags []*model.Tag
	query := database.GetDB(ctx).Model(&model.Tag{}).
		Where("type <> ?", model.TagTypeAlbum)

	if keyword != "" {
		// 支持中文名称和英文名称模糊搜索
		query = query.Where("name ILIKE ? OR name_en ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Order("name ASC").Find(&tags).Error
	if err != nil {
		return nil, err
	}

	return tags, nil
}

// GetPopularTags 获取热门标签（按使用次数排序）
func (r *imageRepository) GetPopularTags(ctx context.Context, limit int) ([]*model.Tag, error) {
	var tags []*model.Tag
	err := database.GetDB(ctx).
		Select("tags.*, COUNT(image_tags.id) as usage_count").
		Joins("LEFT JOIN image_tags ON tags.id = image_tags.tag_id").
		Where("tags.type <> ?", model.TagTypeAlbum).
		Group("tags.id").
		Order("usage_count DESC, tags.name ASC").
		Limit(limit).
		Find(&tags).Error

	if err != nil {
		return nil, err
	}

	return tags, nil
}

// GetImagesWithLocation 获取带有地理位置的图片
func (r *imageRepository) GetImagesWithLocation(ctx context.Context) ([]*model.Image, error) {
	var images []*model.Image
	err := database.GetDB(ctx).WithContext(ctx).
		Where("location IS NOT NULL").
		Find(&images).Error

	if err != nil {
		return nil, err
	}

	return images, nil
}

// GetClusters 获取聚合后的图片数据
func (r *imageRepository) GetClusters(ctx context.Context, minLat, maxLat, minLng, maxLng float64, gridSizeLat, gridSizeLng float64) ([]*model.ClusterResult, error) {
	var results []*model.ClusterResult

	// 1. 聚合查询
	// 使用 floor(lat/gridSizeLat) 和 floor(lng/gridSizeLng) 进行分组
	type clusterRaw struct {
		GridIndexLat int
		GridIndexLng int
		Lat          float64
		Lng          float64
		Count        int64
		CoverID      int64
	}
	var raws []clusterRaw

	// PostgreSQL 语法 - 使用 PostGIS 从 location 字段提取经纬度
	// 确保只查询有坐标的图片
	err := database.GetDB(ctx).WithContext(ctx).Model(&model.Image{}).Raw(`
		SELECT
			FLOOR(ST_Y(location) / ?) as grid_index_lat,
			FLOOR(ST_X(location) / ?) as grid_index_lng,
			AVG(ST_Y(location)) as lat,
			AVG(ST_X(location)) as lng,
			COUNT(*) as count,
			MAX(id) as cover_id
		FROM images
		WHERE location IS NOT NULL
		  AND ST_Y(location) BETWEEN ? AND ?
		  AND ST_X(location) BETWEEN ? AND ?
		  AND deleted_at IS NULL
		GROUP BY grid_index_lat, grid_index_lng
	`, gridSizeLat, gridSizeLng, minLat, maxLat, minLng, maxLng).Scan(&raws).Error

	if err != nil {
		return nil, err
	}

	if len(raws) == 0 {
		return []*model.ClusterResult{}, nil
	}

	// 2. 获取 Cover Image 详情
	coverIDs := make([]int64, 0, len(raws))
	for _, raw := range raws {
		coverIDs = append(coverIDs, raw.CoverID)
	}

	var images []*model.Image
	err = database.GetDB(ctx).WithContext(ctx).
		Where("id IN ?", coverIDs).
		Find(&images).Error

	if err != nil {
		return nil, err
	}

	// 建立 ID -> Image 映射
	imageMap := make(map[int64]*model.Image)
	for _, img := range images {
		imageMap[img.ID] = img
	}

	// 3. 组装结果
	for _, raw := range raws {
		if img, ok := imageMap[raw.CoverID]; ok {
			// 计算该网格的经纬度范围
			minLat := float64(raw.GridIndexLat) * gridSizeLat
			maxLat := float64(raw.GridIndexLat+1) * gridSizeLat
			minLng := float64(raw.GridIndexLng) * gridSizeLng
			maxLng := float64(raw.GridIndexLng+1) * gridSizeLng

			results = append(results, &model.ClusterResult{
				MinLat:     minLat,
				MaxLat:     maxLat,
				MinLng:     minLng,
				MaxLng:     maxLng,
				Latitude:   raw.Lat,
				Longitude:  raw.Lng,
				Count:      raw.Count,
				CoverImage: img,
			})
		}
	}

	return results, nil
}

// GetClusterImages 获取指定聚合组内的图片（分页）
func (r *imageRepository) GetClusterImages(ctx context.Context, minLat, maxLat, minLng, maxLng float64, page, pageSize int) ([]*model.Image, int64, error) {
	var images []*model.Image
	var total int64

	db := database.GetDB(ctx).WithContext(ctx).Model(&model.Image{}).
		Where("location IS NOT NULL").
		Where("ST_Y(location) >= ? AND ST_Y(location) < ?", minLat, maxLat).
		Where("ST_X(location) >= ? AND ST_X(location) < ?", minLng, maxLng)

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Order("taken_at DESC").Find(&images).Error; err != nil {
		return nil, 0, err
	}

	return images, total, nil
}

// GetGeoBounds 获取所有带坐标图片的地理边界
func (r *imageRepository) GetGeoBounds(ctx context.Context) (*model.GeoBounds, error) {
	var result struct {
		MinLat float64
		MaxLat float64
		MinLng float64
		MaxLng float64
		Count  int64
	}

	err := database.GetDB(ctx).WithContext(ctx).Model(&model.Image{}).
		Select("MIN(ST_Y(location)) as min_lat, MAX(ST_Y(location)) as max_lat, MIN(ST_X(location)) as min_lng, MAX(ST_X(location)) as max_lng, COUNT(*) as count").
		Where("location IS NOT NULL").
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	if result.Count == 0 {
		return nil, nil
	}

	return &model.GeoBounds{
		MinLat: result.MinLat,
		MaxLat: result.MaxLat,
		MinLng: result.MinLng,
		MaxLng: result.MaxLng,
		Count:  result.Count,
	}, nil
}

// ListDeleted 分页获取已删除的图片列表
func (r *imageRepository) ListDeleted(ctx context.Context, page, pageSize int) ([]*model.Image, int64, error) {
	var images []*model.Image
	var total int64

	offset := (page - 1) * pageSize

	// 使用 Unscoped 查询已删除的记录
	db := database.GetDB(ctx).WithContext(ctx).Unscoped().Model(&model.Image{}).
		Where("deleted_at IS NOT NULL")

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Preload("Tags").
		Order("deleted_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&images).Error

	if err != nil {
		return nil, 0, err
	}

	return images, total, nil
}

// FindDeletedByIDs 根据ID列表查找已删除的图片
func (r *imageRepository) FindDeletedByIDs(ctx context.Context, ids []int64) ([]*model.Image, error) {
	var images []*model.Image
	err := database.GetDB(ctx).WithContext(ctx).Unscoped().
		Where("id IN ? AND deleted_at IS NOT NULL", ids).
		Find(&images).Error

	if err != nil {
		return nil, err
	}

	return images, nil
}

// RestoreBatch 批量恢复已删除的图片
func (r *imageRepository) RestoreBatch(ctx context.Context, ids []int64) error {
	return database.GetDB(ctx).WithContext(ctx).Unscoped().Model(&model.Image{}).
		Where("id IN ?", ids).
		Update("deleted_at", nil).Error
}

// HardDelete 物理删除单个图片记录

// HardDeleteBatch 物理删除批量图片记录
func (r *imageRepository) HardDeleteBatch(ctx context.Context, ids []int64) error {
	return database.GetDB(ctx).WithContext(ctx).Unscoped().Delete(&model.Image{}, ids).Error
}

// FindExpiredDeleted 查找超过指定天数的已删除图片
func (r *imageRepository) FindExpiredDeleted(ctx context.Context, days int) ([]*model.Image, error) {
	var images []*model.Image
	expireTime := time.Now().AddDate(0, 0, -days)

	err := database.GetDB(ctx).WithContext(ctx).Unscoped().
		Where("deleted_at IS NOT NULL AND deleted_at < ?", expireTime).
		Find(&images).Error

	if err != nil {
		return nil, err
	}

	return images, nil
}

// CountByStorageType 按存储类型统计图片数量
func (r *imageRepository) CountByStorageType(ctx context.Context, storageType string) (int, error) {
	var count int64
	err := database.GetDB(ctx).WithContext(ctx).Model(&model.Image{}).
		Where("storage_type = ?", storageType).
		Count(&count).Error

	if err != nil {
		return 0, err
	}

	return int(count), nil
}

// ListByStorageType 按存储类型分页获取图片列表

// UpdateStoragePath 更新图片存储路径

// FindImagesWithoutAIScore 查找没有 AI 评分的图片ID列表
func (r *imageRepository) FindImagesWithoutAIScore(ctx context.Context, limit int) ([]int64, error) {
	var ids []int64
	err := database.GetDB(ctx).WithContext(ctx).Model(&model.Image{}).
		Select("id").
		Where("ai_score IS NULL").
		Order("created_at DESC").
		Limit(limit).
		Pluck("id", &ids).Error

	if err != nil {
		return nil, err
	}

	return ids, nil
}
