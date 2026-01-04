package repository

import (
	"context"

	"gallary/server/internal/model"
	"gallary/server/pkg/database"

	"gorm.io/gorm/clause"
)

// TagEmbeddingRepository 标签向量仓库接口
type TagEmbeddingRepository interface {
	// 基础操作
	Save(ctx context.Context, embedding *model.TagEmbedding) error

	// 向量搜索（使用 pgvector）
	VectorSearchByCategory(ctx context.Context, modelName string, embedding []float32, limit int) ([]TagEmbeddingWithDistance, error)
}

// TagEmbeddingWithDistance 带距离的标签向量结果
type TagEmbeddingWithDistance struct {
	TagEmbedding *model.TagEmbedding
	Distance     float64 // pgvector 余弦距离（0-2，越小越相似）
	Similarity   float64 // 转换后的相似度（0-1，越大越相似）
}

type tagEmbeddingRepository struct{}

// NewTagEmbeddingRepository 创建标签向量仓库实例
func NewTagEmbeddingRepository() TagEmbeddingRepository {
	return &tagEmbeddingRepository{}
}

// Save 保存或更新标签向量
func (r *tagEmbeddingRepository) Save(ctx context.Context, embedding *model.TagEmbedding) error {
	return database.GetDB(ctx).WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tag_id"}, {Name: "model_name"}},
			DoUpdates: clause.AssignmentColumns([]string{"embedding", "dimension", "vector_description", "category_id", "updated_at"}),
		}).
		Create(embedding).Error
}

// VectorSearchMainCategories 搜索主分类向量（用于判断图片属于哪个主分类）
// 只搜索 sub_category_id 为 '__main_category__' 的特殊记录

// VectorSearchByCategory 在指定分类下搜索相似标签（使用 pgvector）
func (r *tagEmbeddingRepository) VectorSearchByCategory(ctx context.Context, modelName string, embedding []float32, limit int) ([]TagEmbeddingWithDistance, error) {
	vectorStr := model.FloatsToVectorString(embedding)

	var results []struct {
		model.TagEmbedding
		Distance float64 `gorm:"column:distance"`
	}

	err := database.GetDB(ctx).WithContext(ctx).
		Model(&model.TagEmbedding{}).
		Select("*, embedding <=> ? as distance", vectorStr).
		Joins("left join tags on tags.id = tag_embeddings.tag_id ").
		Where("model_name = ?", modelName).
		Where("tags.sub_category_id != ?", "__main_category__"). // 排除主分类向量
		Order("distance ASC").
		Limit(limit).
		Preload("Tag").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	embeddings := make([]TagEmbeddingWithDistance, len(results))
	for i, r := range results {
		e := r.TagEmbedding
		embeddings[i] = TagEmbeddingWithDistance{
			TagEmbedding: &e,
			Distance:     r.Distance,
			Similarity:   1 - r.Distance/2,
		}
	}

	return embeddings, nil
}
