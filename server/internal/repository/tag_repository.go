package repository

import (
	"context"
	"errors"
	"gallary/server/internal/model"
	"gallary/server/pkg/database"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TagRepository 标签仓库接口
type TagRepository interface {
	// 基础操作
	FindByID(ctx context.Context, id int64) (*model.Tag, error)
	FindByIDs(ctx context.Context, ids []int64) ([]*model.Tag, error)
	// 查找或创建
	FindOrCreateByName(ctx context.Context, name string, nameEn *string, vectorDesc *string, categoryId *string, subCategoryId *string) (*model.Tag, error)
	// 向量相关查询
	FindTagsNeedingEmbedding(ctx context.Context, modelName string) ([]*model.Tag, error)
	// 查找有向量但没有普通标签的图片ID（用于自动打标签队列）
	FindImagesWithoutTags(ctx context.Context, modelName string, limit int) ([]int64, error)
	FindMainCategoryTag(ctx context.Context, categoryId string) (*model.Tag, error)
}

type tagRepository struct{}

// NewTagRepository 创建标签仓库实例
func NewTagRepository() TagRepository {
	return &tagRepository{}
}

// FindByID 根据ID查询标签
func (r *tagRepository) FindByID(ctx context.Context, id int64) (*model.Tag, error) {
	var tag model.Tag
	err := database.GetDB(ctx).WithContext(ctx).
		Where("id = ?", id).
		First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// FindByIDs 根据ID批量查询标签
func (r *tagRepository) FindByIDs(ctx context.Context, ids []int64) ([]*model.Tag, error) {
	if len(ids) == 0 {
		return []*model.Tag{}, nil
	}
	var tags []*model.Tag
	err := database.GetDB(ctx).WithContext(ctx).
		Where("id IN ?", ids).
		Find(&tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// FindOrCreateByName 根据名称查找或创建标签
func (r *tagRepository) FindOrCreateByName(ctx context.Context, name string, nameEn *string, vectorDesc *string, categoryId *string, subCategoryId *string) (*model.Tag, error) {
	var tag model.Tag

	// 先尝试查找
	err := database.GetDB(ctx).WithContext(ctx).
		Where("name = ?", name).
		First(&tag).Error

	if err == nil {
		// 标签已存在，更新向量相关字段（如果有变化）
		needUpdate := false
		if nameEn != nil && (tag.NameEn == nil || *tag.NameEn != *nameEn) {
			tag.NameEn = nameEn
			needUpdate = true
		}
		if vectorDesc != nil && (tag.VectorDescription == nil || *tag.VectorDescription != *vectorDesc) {
			tag.VectorDescription = vectorDesc
			needUpdate = true
		}
		if categoryId != nil && (tag.SourceCategoryId == nil || *tag.SourceCategoryId != *categoryId) {
			tag.SourceCategoryId = categoryId
			needUpdate = true
		}
		if subCategoryId != nil && (tag.SubCategoryId == nil || *tag.SubCategoryId != *subCategoryId) {
			tag.SubCategoryId = subCategoryId
			needUpdate = true
		}

		if needUpdate {
			if updateErr := database.GetDB(ctx).WithContext(ctx).Save(&tag).Error; updateErr != nil {
				return nil, updateErr
			}
		}
		return &tag, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 创建新标签
	tag = model.Tag{
		Name:              name,
		NameEn:            nameEn,
		VectorDescription: vectorDesc,
		SourceCategoryId:  categoryId,
		SubCategoryId:     subCategoryId,
		Type:              model.TagTypeNormal,
	}

	err = database.GetDB(ctx).WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoUpdates: clause.AssignmentColumns([]string{"name_en", "vector_description", "source_category_id", "sub_category_id", "updated_at"}),
		}).
		Create(&tag).Error

	if err != nil {
		return nil, err
	}

	return &tag, nil
}

// FindTagsNeedingEmbedding 查找需要生成向量的标签（有描述但无对应模型向量）
func (r *tagRepository) FindTagsNeedingEmbedding(ctx context.Context, modelName string) ([]*model.Tag, error) {
	var tags []*model.Tag

	db := database.GetDB(ctx).WithContext(ctx)
	// 主查询：有向量描述但不在子查询结果中的标签
	err := db.
		Where("vector_description IS NOT NULL AND vector_description != ''").
		// 子查询：获取已有该模型向量的 TagID
		Where("id NOT IN (?)", db.Model(&model.TagEmbedding{}).Select("tag_id").Where("model_name = ?", modelName)).
		Order("source_category_id ASC, id ASC").
		Find(&tags).
		Error

	return tags, err
}

// FindImagesWithoutTags 查找有向量但没有普通标签的图片ID（用于自动打标签队列）
func (r *tagRepository) FindImagesWithoutTags(ctx context.Context, modelName string, limit int) ([]int64, error) {
	var imageIDs []int64

	db := database.GetDB(ctx).WithContext(ctx)
	// 子查询：获取已有普通标签的图片ID
	// 主查询：查找有向量但没有普通标签的图片
	err := db.
		Model(&model.ImageEmbedding{}).
		Select("DISTINCT image_id").
		Where("model_name = ?", modelName).
		Where("image_id NOT IN (?)", db.Model(&model.ImageTag{}).Select("DISTINCT image_tags.image_id")).
		Limit(limit).
		Pluck("image_id", &imageIDs).Error

	return imageIDs, err
}

func (r *tagRepository) FindMainCategoryTag(ctx context.Context, categoryId string) (*model.Tag, error) {
	var tag model.Tag
	err := database.GetDB(ctx).WithContext(ctx).
		Where("source_category_id = ?", categoryId).
		Where("sub_category_id = ?", model.MainCategoryMarker).
		First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}
