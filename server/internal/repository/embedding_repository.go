package repository

import (
	"context"
	"fmt"

	"gallary/server/internal/model"
	"gallary/server/pkg/database"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// EmbeddingRepository 向量嵌入仓库接口
type EmbeddingRepository interface {
	// 基础操作
	Save(ctx context.Context, embedding *model.ImageEmbedding) error
	FindByID(ctx context.Context, id int64) (*model.ImageEmbedding, error)
	FindByImageAndModel(ctx context.Context, imageID int64, modelID string) (*model.ImageEmbedding, error)
	Delete(ctx context.Context, id int64) error
	DeleteByImageID(ctx context.Context, imageID int64) error

	// 批量操作
	SaveBatch(ctx context.Context, embeddings []*model.ImageEmbedding) error
	FindByImageIDs(ctx context.Context, imageIDs []int64, modelID string) ([]*model.ImageEmbedding, error)

	// 向量搜索
	VectorSearchByModelName(ctx context.Context, modelName string, embedding []float32, limit int) ([]*model.ImageEmbedding, error)
	VectorSearchWithDistance(ctx context.Context, modelID string, embedding []float32, limit int) ([]EmbeddingWithDistance, error)
	VectorSearchWithinIDs(ctx context.Context, modelName string, embedding []float32, candidateIDs []int64, limit int) ([]EmbeddingWithDistance, error)

	// 查询未处理的图片
	FindImagesWithoutEmbedding(ctx context.Context, modelName string, limit int) ([]int64, error)
}

// EmbeddingWithDistance 带距离的嵌入结果
type EmbeddingWithDistance struct {
	Embedding *model.ImageEmbedding
	Distance  float64
}

type embeddingRepository struct{}

// NewEmbeddingRepository 创建向量嵌入仓库实例
func NewEmbeddingRepository() EmbeddingRepository {
	return &embeddingRepository{}
}

// Save 保存或更新向量嵌入
func (r *embeddingRepository) Save(ctx context.Context, embedding *model.ImageEmbedding) error {
	return database.GetDB(ctx).WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "image_id"}, {Name: "model_name"}},
			DoUpdates: clause.AssignmentColumns([]string{"embedding", "model_id", "dimension", "updated_at"}),
		}).
		Create(embedding).Error
}

// FindByID 根据 ID 查找
func (r *embeddingRepository) FindByID(ctx context.Context, id int64) (*model.ImageEmbedding, error) {
	var embedding model.ImageEmbedding
	err := database.GetDB(ctx).WithContext(ctx).First(&embedding, id).Error
	if err != nil {
		return nil, err
	}
	return &embedding, nil
}

// FindByImageAndModel 根据图片ID和模型ID查找
func (r *embeddingRepository) FindByImageAndModel(ctx context.Context, imageID int64, modelID string) (*model.ImageEmbedding, error) {
	var embedding model.ImageEmbedding
	err := database.GetDB(ctx).WithContext(ctx).
		Where("image_id = ? AND model_id = ?", imageID, modelID).
		First(&embedding).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &embedding, nil
}

// Delete 删除向量嵌入
func (r *embeddingRepository) Delete(ctx context.Context, id int64) error {
	return database.GetDB(ctx).WithContext(ctx).Delete(&model.ImageEmbedding{}, id).Error
}

// DeleteByImageID 删除图片的所有向量嵌入
func (r *embeddingRepository) DeleteByImageID(ctx context.Context, imageID int64) error {
	return database.GetDB(ctx).WithContext(ctx).
		Where("image_id = ?", imageID).
		Delete(&model.ImageEmbedding{}).Error
}

// SaveBatch 批量保存向量嵌入
func (r *embeddingRepository) SaveBatch(ctx context.Context, embeddings []*model.ImageEmbedding) error {
	if len(embeddings) == 0 {
		return nil
	}
	return database.GetDB(ctx).WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "image_id"}, {Name: "model_name"}},
			DoUpdates: clause.AssignmentColumns([]string{"embedding", "model_id", "dimension", "updated_at"}),
		}).
		CreateInBatches(embeddings, 100).Error
}

// FindByImageIDs 批量查找图片的向量嵌入
func (r *embeddingRepository) FindByImageIDs(ctx context.Context, imageIDs []int64, modelID string) ([]*model.ImageEmbedding, error) {
	var embeddings []*model.ImageEmbedding
	query := database.GetDB(ctx).WithContext(ctx).
		Where("image_id IN ?", imageIDs)
	if modelID != "" {
		query = query.Where("model_id = ?", modelID)
	}
	err := query.Find(&embeddings).Error
	return embeddings, err
}

// VectorSearch 向量相似性搜索

// VectorSearchByModelName 根据模型名称进行向量相似性搜索
func (r *embeddingRepository) VectorSearchByModelName(ctx context.Context, modelName string, embedding []float32, limit int) ([]*model.ImageEmbedding, error) {
	var embeddings []*model.ImageEmbedding

	// 构建向量字符串
	vectorStr := floatsToVectorString(embedding)

	err := database.GetDB(ctx).WithContext(ctx).
		Where("model_name = ?", modelName).
		Order(fmt.Sprintf("embedding <=> '%s'", vectorStr)).
		Limit(limit).
		Preload("Image").
		Find(&embeddings).Error

	return embeddings, err
}

// VectorSearchWithDistance 向量相似性搜索（返回距离）
func (r *embeddingRepository) VectorSearchWithDistance(ctx context.Context, modelID string, embedding []float32, limit int) ([]EmbeddingWithDistance, error) {
	vectorStr := floatsToVectorString(embedding)

	var results []struct {
		model.ImageEmbedding
		Distance float64 `gorm:"column:distance"`
	}

	err := database.GetDB(ctx).WithContext(ctx).
		Model(&model.ImageEmbedding{}).
		Select("*, embedding <=> ? as distance", vectorStr).
		Where("model_name = ?", modelID).
		Order("distance ASC").
		Limit(limit).
		Preload("Image").
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

// VectorSearchWithinIDs 在指定图片ID范围内进行向量相似性搜索
func (r *embeddingRepository) VectorSearchWithinIDs(ctx context.Context, modelName string, embedding []float32, candidateIDs []int64, limit int) ([]EmbeddingWithDistance, error) {
	if candidateIDs != nil && len(candidateIDs) == 0 {
		return []EmbeddingWithDistance{}, nil
	}

	vectorStr := floatsToVectorString(embedding)

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

// floatsToVectorString 将 float32 数组转换为 PostgreSQL 向量字符串
func floatsToVectorString(floats []float32) string {
	if len(floats) == 0 {
		return "[]"
	}

	result := "["
	for i, f := range floats {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("%f", f)
	}
	result += "]"
	return result
}

// FindImagesWithoutEmbedding 查询未计算向量的图片 ID（未被删除的图片）
func (r *embeddingRepository) FindImagesWithoutEmbedding(ctx context.Context, modelName string, limit int) ([]int64, error) {
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
