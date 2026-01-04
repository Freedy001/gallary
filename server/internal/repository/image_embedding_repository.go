package repository

import (
	"context"
	"gallary/server/internal/model"
	"gallary/server/pkg/database"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ImageEmbeddingRepository 向量嵌入仓库接口
type ImageEmbeddingRepository interface {
	// 基础操作
	Save(ctx context.Context, embedding *model.ImageEmbedding) error
	FindByImageAndModel(ctx context.Context, imageID int64, modelID string) (*model.ImageEmbedding, error)

	// 向量搜索
	VectorSearchWithinIDs(ctx context.Context, modelName string, embedding []float32, candidateIDs []int64, limit int) ([]EmbeddingWithDistance, error)

	// 查询未处理的图片
	FindImagesWithoutEmbedding(ctx context.Context, modelName string, limit int) ([]int64, error)
}

// EmbeddingWithDistance 带距离的嵌入结果
type EmbeddingWithDistance struct {
	Embedding *model.ImageEmbedding
	Distance  float64
}

type imageEmbeddingRepository struct{}

// NewEmbeddingRepository 创建向量嵌入仓库实例
func NewEmbeddingRepository() ImageEmbeddingRepository {
	return &imageEmbeddingRepository{}
}

// Save 保存或更新向量嵌入
func (r *imageEmbeddingRepository) Save(ctx context.Context, embedding *model.ImageEmbedding) error {
	return database.GetDB(ctx).WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "image_id"}, {Name: "model_name"}},
			DoUpdates: clause.AssignmentColumns([]string{"embedding", "model_name", "dimension", "updated_at"}),
		}).
		Create(embedding).Error
}

// FindByID 根据 ID 查找

// FindByImageAndModel 根据图片ID和模型ID查找
func (r *imageEmbeddingRepository) FindByImageAndModel(ctx context.Context, imageID int64, modelID string) (*model.ImageEmbedding, error) {
	var embedding model.ImageEmbedding
	err := database.GetDB(ctx).WithContext(ctx).
		Where("image_id = ? AND model_name = ?", imageID, modelID).
		First(&embedding).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &embedding, nil
}

// VectorSearchWithinIDs 在指定图片ID范围内进行向量相似性搜索
func (r *imageEmbeddingRepository) VectorSearchWithinIDs(ctx context.Context, modelName string, embedding []float32, candidateIDs []int64, limit int) ([]EmbeddingWithDistance, error) {
	if candidateIDs != nil && len(candidateIDs) == 0 {
		return []EmbeddingWithDistance{}, nil
	}

	vectorStr := model.FloatsToVectorString(embedding)

	var results []struct {
		model.ImageEmbedding
		Distance float64 `gorm:"column:distance"`
	}

	db := database.GetDB(ctx).WithContext(ctx).
		Model(&model.ImageEmbedding{}).
		Select("*, embedding <=> ? as distance", vectorStr).
		Where("model_name = ?", modelName)

	if candidateIDs != nil {
		db.Where("image_id IN ?", candidateIDs)
	}

	err := db.Order("distance ASC").
		Limit(limit).
		Preload("Image").
		Preload("Image.Tags").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	embeddings := make([]EmbeddingWithDistance, len(results))
	for i, r := range results {
		e := r.ImageEmbedding
		embeddings[i] = EmbeddingWithDistance{
			Embedding: &e,
			Distance:  r.Distance,
		}
	}

	return embeddings, nil
}

// FindImagesWithoutEmbedding 查询未计算向量的图片 ID（未被删除的图片）
func (r *imageEmbeddingRepository) FindImagesWithoutEmbedding(ctx context.Context, modelName string, limit int) ([]int64, error) {
	var imageIDs []int64

	// 使用子查询排除已经有向量的图片
	subQuery := database.GetDB(ctx).WithContext(ctx).
		Model(&model.ImageEmbedding{}).
		Select("image_id").
		Where("model_name = ?", modelName)

	err := database.GetDB(ctx).WithContext(ctx).
		Model(&model.Image{}).
		Select("id").
		Where("deleted_at IS NULL").
		Where("id NOT IN (?)", subQuery).
		Order("created_at DESC").
		Limit(limit).
		Pluck("id", &imageIDs).Error

	return imageIDs, err
}
