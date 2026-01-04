package service

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"gallary/server/internal/llms"
	"gallary/server/internal/model"
	"gallary/server/internal/repository"
	"gallary/server/pkg/logger"
	"os"
	"path/filepath"
	"slices"

	"github.com/samber/lo"
	"go.uber.org/zap"
)

// TaggingService 自动打标签服务接口
type TaggingService interface {
	// SyncTagsIfChanged 同步tag
	SyncTagsIfChanged(ctx context.Context) error

	// TaggingImage给图片打标机
	TaggingImage(ctx context.Context, imageID int64, modelName string) error
}

type taggingService struct {
	tagRepo          repository.TagRepository
	tagEmbeddingRepo repository.TagEmbeddingRepository
	embeddingRepo    repository.ImageEmbeddingRepository
	imageRepo        repository.ImageRepository
	loadBalancer     *llms.ModelLoadBalancer
	tagsDir          string
	lastTagFilesHash string // 上次标签文件的 MD5 哈希
}

// NewTaggingService 创建打标签服务实例
func NewTaggingService(
	tagRepo repository.TagRepository,
	tagEmbeddingRepo repository.TagEmbeddingRepository,
	embeddingRepo repository.ImageEmbeddingRepository,
	imageRepo repository.ImageRepository,
	loadBalancer *llms.ModelLoadBalancer,
	tagsDir string,
) TaggingService {
	return &taggingService{
		tagRepo:          tagRepo,
		tagEmbeddingRepo: tagEmbeddingRepo,
		embeddingRepo:    embeddingRepo,
		imageRepo:        imageRepo,
		loadBalancer:     loadBalancer,
		tagsDir:          tagsDir,
	}
}

// SetSimilarityCalculator 设置相似度计算器

// ================== 标签加载 ==================

func (s *taggingService) SyncTagsIfChanged(ctx context.Context) error {
	// 1. 计算当前标签文件的 MD5
	currentHash, err := s.GetTagFilesHash(ctx)
	if err != nil {
		return fmt.Errorf("计算标签文件哈希失败 %v", err)
	}

	// 2. 检查是否有变化（首次运行时 lastTagFilesHash 为空，会触发同步）
	if currentHash != s.lastTagFilesHash {
		logger.Info("检测到标签文件变化，开始同步",
			zap.String("old_hash", s.lastTagFilesHash),
			zap.String("new_hash", currentHash))

		// 3. 同步标签到数据库
		if err := s.LoadAndSyncTagsFromJSON(ctx); err != nil {
			return fmt.Errorf("同步标签失败 %v", err)
		}

		// 4. 更新缓存的 MD5
		s.lastTagFilesHash = currentHash
	}

	return nil
}

// loadTagsFromFile 从单个 JSON 文件加载标签（递归解析嵌套结构）
func (s *taggingService) loadTagsFromFile(filePath, categoryId string) ([]model.SourceTag, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// 解析为通用 map 结构，递归提取标签
	var content map[string]interface{}
	if err := json.Unmarshal(data, &content); err != nil {
		return nil, err
	}

	// 从 categories 开始解析
	if categories, ok := content["categories"].(map[string]interface{}); ok {
		return s.extractTagsRecursiveWithSubCategory(categories, categoryId, ""), nil
	}

	return s.extractTagsRecursiveWithSubCategory(content, categoryId, ""), nil
}

// extractTagsRecursiveWithSubCategory 递归提取标签（同时记录子分类ID）
func (s *taggingService) extractTagsRecursiveWithSubCategory(data map[string]interface{}, categoryId, currentKey string) []model.SourceTag {
	var tags []model.SourceTag

	// 检查是否是标签数组
	if tagsArray, ok := data["tags"].([]interface{}); ok {
		for _, tagItem := range tagsArray {
			if tagMap, ok := tagItem.(map[string]interface{}); ok {
				tag := model.SourceTag{
					Name:              getString(tagMap, "name"),
					NameEn:            getString(tagMap, "name_en"),
					VectorDescription: getString(tagMap, "vector_description"),
					CategoryId:        categoryId,
					SubCategoryId:     currentKey, // 当前节点的 key 就是子分类ID
				}
				if tag.VectorDescription != "" && tag.Name != "" {
					tags = append(tags, tag)
				}
			}
		}
	}

	// 递归处理嵌套结构
	for key, value := range data {
		// 跳过非结构化字段
		if key == "tags" || key == "name" || key == "name_en" || key == "description" {
			continue
		}
		if nestedMap, ok := value.(map[string]interface{}); ok {
			// 检查是否有 subcategories
			if subcategories, ok := nestedMap["subcategories"].(map[string]interface{}); ok {
				// 递归处理 subcategories
				nestedTags := s.extractTagsRecursiveWithSubCategory(subcategories, categoryId, "")
				tags = append(tags, nestedTags...)
			} else {
				// 直接递归处理，使用 key 作为子分类ID
				nestedTags := s.extractTagsRecursiveWithSubCategory(nestedMap, categoryId, key)
				tags = append(tags, nestedTags...)
			}
		}
	}

	return tags
}

// getString 安全获取字符串值
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// loadTagsFromJSON 从 JSON 文件加载所有标签（包括主分类向量）
func (s *taggingService) loadTagsFromJSON() ([]model.SourceTag, error) {
	// 读取索引文件
	indexPath := filepath.Join(s.tagsDir, "index.json")
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("读取索引文件失败: %v", err)
	}

	var tagIndex model.TagIndex
	if err := json.Unmarshal(indexData, &tagIndex); err != nil {
		return nil, fmt.Errorf("解析索引文件失败: %v", err)
	}

	var allTags []model.SourceTag

	// 遍历所有分类文件
	for _, category := range tagIndex.Categories {
		// 如果主分类有 vector_description，创建主分类的虚拟标签
		if category.VectorDescription != "" {
			mainCategoryTag := model.SourceTag{
				Name:              category.Name,
				NameEn:            category.NameEn,
				VectorDescription: category.VectorDescription,
				CategoryId:        category.ID,
				SubCategoryId:     model.MainCategoryMarker, // 使用特殊标记
			}
			allTags = append(allTags, mainCategoryTag)
		}

		// 加载分类下的所有标签
		filePath := filepath.Join(s.tagsDir, category.File)
		tags, err := s.loadTagsFromFile(filePath, category.ID)
		if err != nil {
			logger.Warn("加载标签文件失败",
				zap.String("file", category.File),
				zap.Error(err))
			continue
		}
		allTags = append(allTags, tags...)
	}

	logger.Info("标签加载完成", zap.Int("total_count", len(allTags)))
	return allTags, nil
}

// LoadAndSyncTagsFromJSON 从 JSON 加载标签并同步到 tags 表
func (s *taggingService) LoadAndSyncTagsFromJSON(ctx context.Context) error {
	sourceTags, err := s.loadTagsFromJSON()
	if err != nil {
		return err
	}

	syncedCount := 0
	for _, sourceTag := range sourceTags {
		nameEn := sourceTag.NameEn
		vectorDesc := sourceTag.VectorDescription
		categoryId := sourceTag.CategoryId
		subCategoryId := sourceTag.SubCategoryId

		_, err := s.tagRepo.FindOrCreateByName(ctx, sourceTag.Name, &nameEn, &vectorDesc, &categoryId, &subCategoryId)
		if err != nil {
			logger.Warn("同步标签失败",
				zap.String("tag_name", sourceTag.Name),
				zap.Error(err))
			continue
		}
		syncedCount++
	}

	logger.Info("标签同步完成",
		zap.Int("synced_count", syncedCount),
		zap.Int("total_count", len(sourceTags)))
	return nil
}

// GetTagFilesHash 计算标签目录所有 JSON 文件的组合 MD5 哈希
func (s *taggingService) GetTagFilesHash(ctx context.Context) (string, error) {
	indexPath := filepath.Join(s.tagsDir, "index.json")

	// 读取索引文件获取所有标签文件列表
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		return "", fmt.Errorf("读取索引文件失败: %v", err)
	}

	var tagIndex model.TagIndex
	if err := json.Unmarshal(indexData, &tagIndex); err != nil {
		return "", fmt.Errorf("解析索引文件失败: %v", err)
	}

	// 计算所有文件的组合哈希
	hash := md5.New()

	// 先计算索引文件的哈希
	hash.Write(indexData)

	// 按固定顺序计算每个分类文件的哈希
	for _, category := range tagIndex.Categories {
		filePath := filepath.Join(s.tagsDir, category.File)
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue // 跳过读取失败的文件
		}
		hash.Write(data)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// InitializeTagEmbeddings 为标签生成向量（增量更新）

// ================== 相似度匹配 ==================

// 默认配置
const (
	defaultMaxTagsPerCategory = 3 // 每个主分类最大标签数
)

// MatchedTag 匹配结果
type MatchedTag struct {
	TagID         int64   `json:"tag_id"`          // 关联 Tag 表的 ID
	TagName       string  `json:"tag_name"`        // 中文名称
	TagNameEn     string  `json:"tag_name_en"`     // 英文名称
	CategoryId    string  `json:"category_id"`     // 主分类ID
	SubCategoryId string  `json:"sub_category_id"` // 子分类ID
	Score         float64 `json:"score"`           // 相似度分数
}

// matchTagIdsForImage 为图片匹配标签
func (s *taggingService) matchTagIdsForImage(ctx context.Context, imageID int64, modelName string) ([]int64, error) {
	// 1. 获取图片向量
	imageEmbedding, err := s.embeddingRepo.FindByImageAndModel(ctx, imageID, modelName)
	if err != nil {
		return nil, fmt.Errorf("获取图片向量失败: %v", err)
	}
	if imageEmbedding == nil {
		return nil, fmt.Errorf("图片向量不存在: image_id=%d, model=%s", imageID, modelName)
	}
	imageVector := []float32(imageEmbedding.Embedding)

	// 3. 使用数据库向量搜索获取匹配的标签
	tagResults, err := s.tagEmbeddingRepo.VectorSearchByCategory(ctx, modelName, imageVector, 100)
	if err != nil {
		return nil, fmt.Errorf("向量搜索标签失败: %v", err)
	}

	results := s.deduplicateBySubCategory(ctx, tagResults)

	logger.Info("标签匹配完成",
		zap.Int64("image_id", imageID),
		zap.Int("candidates_count", len(tagResults)),
		zap.Any("results", lo.Map(results, func(item *model.Tag, index int) string { return item.Name })),
	)

	return lo.Map(results, func(item *model.Tag, index int) int64 { return item.ID }), nil
}

// deduplicateBySubCategory 将向量搜索结果转换为 MatchedTag
func (s *taggingService) deduplicateBySubCategory(ctx context.Context, candidates []repository.TagEmbeddingWithDistance) []*model.Tag {
	if len(candidates) == 0 {
		return make([]*model.Tag, 0)
	}

	slices.SortFunc(candidates, func(a, b repository.TagEmbeddingWithDistance) int {
		return int((a.Distance - b.Distance) * 1000)
	})

	seen := make(map[string]bool)
	result := lo.FilterMap(candidates, func(item repository.TagEmbeddingWithDistance, index int) (*model.Tag, bool) {
		subcategory := *item.TagEmbedding.Tag.SubCategoryId
		if !seen[subcategory] {
			seen[subcategory] = true
			return item.TagEmbedding.Tag, true
		}
		return nil, false
	})

	tag, err := s.tagRepo.FindMainCategoryTag(ctx, *candidates[0].TagEmbedding.Tag.SourceCategoryId)
	if err == nil && tag != nil {
		result = append([]*model.Tag{tag}, result...)
	}

	// 限制最大返回数量
	if len(result) > defaultMaxTagsPerCategory {
		return result[:defaultMaxTagsPerCategory]
	}

	return result
}

func (s *taggingService) TaggingImage(ctx context.Context, imageID int64, modelName string) error {
	tagIDs, err := s.matchTagIdsForImage(ctx, imageID, modelName)
	if err != nil {
		return fmt.Errorf("匹配标签失败: %v image_id=%d", err, imageID)
	}

	if len(tagIDs) == 0 {
		return nil
	}

	// 创建图片-标签关联
	if err := s.imageRepo.AddImageTags(ctx, imageID, tagIDs); err != nil {
		return fmt.Errorf("创建图片标签关联失败: %v image_id=%d", err, imageID)
	}
	return nil
}
