package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gallary/server/internal/model"
	"gallary/server/pkg/database"
)

// AlbumRepository 相册仓库接口
type AlbumRepository interface {
	Create(ctx context.Context, album *model.Tag) error
	FindByID(ctx context.Context, id int64) (*model.Tag, error)
	FindByIDs(ctx context.Context, ids []int64) ([]*model.Tag, error)
	List(ctx context.Context, page, pageSize int, isSmart *bool) ([]*model.Tag, int64, error)
	Update(ctx context.Context, album *model.Tag) error
	Delete(ctx context.Context, id int64) error

	// 图片关联
	AddImages(ctx context.Context, albumID int64, imageIDs []int64) error
	RemoveImages(ctx context.Context, albumID int64, imageIDs []int64) error
	RemoveImagesFromAllAlbums(ctx context.Context, imageIDs []int64) error
	GetImages(ctx context.Context, albumID int64, page, pageSize int, sortBy string) ([]*model.Image, int64, error)
	GetImageCount(ctx context.Context, albumID int64) (int64, error)
	GetImageCounts(ctx context.Context, albumIDs []int64) (map[int64]int64, error)
	CopyImages(ctx context.Context, srcAlbumID, dstAlbumID int64) error

	// 封面
	GetCoverImage(ctx context.Context, coverImageID int64) (*model.Image, error)
	GetFirstImages(ctx context.Context, albumIDs []int64) (map[int64]*model.Image, error)
	FindBestCoverByAverageVector(ctx context.Context, albumID int64, modelName string) (int64, error)
	Merge(ctx context.Context, sourceAlbumIDs []int64, targetAlbumID int64) error
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("相册不存在")
		}
		return nil, err
	}

	return &album, nil
}

// FindByIDs 根据ID列表批量查找相册
func (r *albumRepository) FindByIDs(ctx context.Context, ids []int64) ([]*model.Tag, error) {
	if len(ids) == 0 {
		return []*model.Tag{}, nil
	}

	var albums []*model.Tag
	err := database.GetDB(ctx).WithContext(ctx).
		Where("id IN ? AND type = ?", ids, model.TagTypeAlbum).
		Find(&albums).Error

	if err != nil {
		return nil, err
	}

	return albums, nil
}

// List 分页获取相册列表
// isSmart: nil-不过滤, true-只返回智能相册, false-只返回普通相册
func (r *albumRepository) List(ctx context.Context, page, pageSize int, isSmart *bool) ([]*model.Tag, int64, error) {
	var albums []*model.Tag
	var total int64

	offset := (page - 1) * pageSize

	db := database.GetDB(ctx).WithContext(ctx).Model(&model.Tag{}).
		Where("type = ?", model.TagTypeAlbum)

	// 根据 isSmart 参数过滤智能相册
	if isSmart != nil {
		if *isSmart {
			db = db.Where("metadata->>'is_smart_album' = 'true'")
		} else {
			db = db.Where("metadata->>'is_smart_album' IS NULL OR metadata->>'is_smart_album' = 'false'")
		}
	}

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

// RemoveImagesFromAllAlbums 从所有相册移除指定图片
func (r *albumRepository) RemoveImagesFromAllAlbums(ctx context.Context, imageIDs []int64) error {
	if len(imageIDs) == 0 {
		return nil
	}
	return database.GetDB(ctx).WithContext(ctx).
		Where("image_id IN ?", imageIDs).
		Delete(&model.ImageTag{}).Error
}

// CopyImages 复制相册中的图片关联到新相册
func (r *albumRepository) CopyImages(ctx context.Context, srcAlbumID, dstAlbumID int64) error {
	// 复制源相册的所有图片关联到目标相册
	return database.GetDB(ctx).WithContext(ctx).Exec(`
		INSERT INTO image_tags (image_id, tag_id, sort_order, created_at)
		SELECT image_id, ?, sort_order, NOW()
		FROM image_tags
		WHERE tag_id = ?
		ON CONFLICT (image_id, tag_id) DO NOTHING
	`, dstAlbumID, srcAlbumID).Error
}

// GetImages 分页获取相册内图片
// sortBy: taken_at-按拍摄时间排序, ai_score-按美学评分排序
func (r *albumRepository) GetImages(ctx context.Context, albumID int64, page, pageSize int, sortBy string) ([]*model.Image, int64, error) {
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

	// 根据排序方式构建排序语句
	var orderClause string
	switch sortBy {
	case "ai_score":
		// 按美学评分降序，NULL 排最后，相同评分按拍摄时间降序
		orderClause = "images.ai_score DESC NULLS LAST, COALESCE(images.taken_at, images.created_at) DESC"
	default:
		// 默认按拍摄时间降序
		orderClause = "image_tags.sort_order ASC, COALESCE(images.taken_at, images.created_at) DESC"
	}

	// 查询图片数据
	err := database.GetDB(ctx).WithContext(ctx).
		Table("images").
		Joins("JOIN image_tags ON images.id = image_tags.image_id").
		Where("image_tags.tag_id = ? AND images.deleted_at IS NULL", albumID).
		Order(orderClause).
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &image, nil
}

// GetFirstImages 批量获取相册的第一张图片（作为默认封面）
// 优先选择美学评分最高的图片，如果没有评分则选择第一张
func (r *albumRepository) GetFirstImages(ctx context.Context, albumIDs []int64) (map[int64]*model.Image, error) {
	if len(albumIDs) == 0 {
		return make(map[int64]*model.Image), nil
	}

	// 使用窗口函数获取每个相册的最佳图片（优先美学评分最高，其次最小 ID）
	type firstImageResult struct {
		TagID   int64
		ImageID int64
	}

	var results []firstImageResult
	// 使用 DISTINCT ON 获取每个相册中评分最高的图片
	// 按 tag_id 分组，然后按 ai_score 降序（NULL 排最后）、id 升序排序
	err := database.GetDB(ctx).WithContext(ctx).
		Raw(`
			SELECT DISTINCT ON (tag_id) tag_id, image_id
			FROM image_tags
			JOIN images ON images.id = image_tags.image_id
			WHERE image_tags.tag_id in ? AND images.deleted_at IS NULL
			ORDER BY tag_id, images.ai_score DESC NULLS LAST, image_tags.id ASC
		`, albumIDs).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return make(map[int64]*model.Image), nil
	}

	var images []*model.Image
	err = database.GetDB(ctx).WithContext(ctx).
		Where("id IN ? AND deleted_at IS NULL", lo.Uniq(lo.Map(results, func(item firstImageResult, index int) int64 { return item.ImageID }))).
		Find(&images).Error
	if err != nil {
		return nil, err
	}

	idImageMap := lo.KeyBy(images, func(item *model.Image) int64 { return item.ID })
	return lo.SliceToMap(results, func(item firstImageResult) (int64, *model.Image) { return item.TagID, idImageMap[item.ImageID] }), nil
}

// FindBestCoverByAverageVector 通过平均向量查找最佳封面
// 计算相册中所有图片向量的平均值，然后找到与平均值最接近的图片作为封面
func (r *albumRepository) FindBestCoverByAverageVector(ctx context.Context, albumID int64, modelName string) (int64, error) {
	// 使用 pgvector 的聚合函数计算平均向量并找到最接近的图片
	// 只计算指定模型的向量
	type result struct {
		ImageID int64
	}

	var res result
	err := database.GetDB(ctx).WithContext(ctx).Raw(`
		WITH avg_vector AS (
			-- 计算指定模型的所有图片向量的平均值
			SELECT AVG(embedding) as avg_emb
			FROM image_embeddings ie
			JOIN image_tags it ON ie.image_id = it.image_id
			WHERE it.tag_id = $1 AND ie.model_name = $2
		)
		-- 找到与平均向量余弦距离最小的图片
		SELECT ie.image_id
		FROM image_embeddings ie
		JOIN image_tags it ON ie.image_id = it.image_id
		CROSS JOIN avg_vector av
		WHERE it.tag_id = $1 AND ie.model_name = $2
		ORDER BY ie.embedding <=> av.avg_emb  -- 余弦距离
		LIMIT 1
	`, albumID, modelName).Scan(&res).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, fmt.Errorf("该相册中没有使用模型 %s 的向量数据", modelName)
		}
		return 0, fmt.Errorf("查询平均向量封面失败: %w", err)
	}

	if res.ImageID == 0 {
		return 0, fmt.Errorf("该相册中没有使用模型 %s 的向量数据", modelName)
	}

	return res.ImageID, nil
}

// Merge 合并相册：将源相册的图片移动到目标相册，并删除源相册
func (r *albumRepository) Merge(ctx context.Context, sourceAlbumIDs []int64, targetAlbumID int64) error {
	return database.GetDB(ctx).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 将源相册的所有图片关联复制到目标相册
		// 使用 INSERT INTO ... SELECT ... ON CONFLICT DO NOTHING
		// 注意：需要处理 sourceAlbumIDs 列表

		for _, sourceID := range sourceAlbumIDs {
			// 跳过目标相册本身
			if sourceID == targetAlbumID {
				continue
			}

			// 将源相册的图片复制到目标相册
			// 为了保持图片顺序，我们获取目标相册当前的最大排序值
			var maxSort int
			tx.Model(&model.ImageTag{}).
				Where("tag_id = ?", targetAlbumID).
				Select("COALESCE(MAX(sort_order), 0)").
				Scan(&maxSort)

			// 插入新记录，注意 sort_order 的处理
			// 这里简单起见，直接使用 SQL 批量插入，sort_order 累加
			err := tx.Exec(`
				INSERT INTO image_tags (image_id, tag_id, sort_order, created_at)
				SELECT image_id, ?, ? + sort_order, NOW()
				FROM image_tags
				WHERE tag_id = ?
				ON CONFLICT (image_id, tag_id) DO NOTHING
			`, targetAlbumID, maxSort, sourceID).Error

			if err != nil {
				return err
			}

			// 2. 删除源相册的关联关系
			if err := tx.Where("tag_id = ?", sourceID).Delete(&model.ImageTag{}).Error; err != nil {
				return err
			}

			// 3. 删除源相册
			if err := tx.Where("id = ? AND type = ?", sourceID, model.TagTypeAlbum).Delete(&model.Tag{}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
