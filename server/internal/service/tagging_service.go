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

	"go.uber.org/zap"
)

// TaggingService 自动打标签服务接口
type TaggingService interface {
	// SyncTagsIfChanged 同步tag
	SyncTagsIfChanged(ctx context.Context) error

	// MatchTagIDsForImage 返回匹配的标签ID列表（简化版本，用于处理器）
	MatchTagIDsForImage(ctx context.Context, imageID int64, modelName string) ([]int64, error)

	//
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
				SubCategoryId:     mainCategoryMarker, // 使用特殊标记
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
	defaultSimilarityThreshold = 0.3                 // 相似度阈值
	defaultMaxTagsPerCategory  = 10                  // 每个主分类最大标签数
	mainCategoryMarker         = "__main_category__" // 主分类向量的特殊标记
)

// 三个主分类的 category_id
var mainCategories = []string{
	"portrait_photography",   // 人像摄影
	"humanities_documentary", // 人文与社会纪实摄影
	"landscape_photography",  // 风光摄影
}

// MatchedTag 匹配结果
type MatchedTag struct {
	TagID         int64   `json:"tag_id"`          // 关联 Tag 表的 ID
	TagName       string  `json:"tag_name"`        // 中文名称
	TagNameEn     string  `json:"tag_name_en"`     // 英文名称
	CategoryId    string  `json:"category_id"`     // 主分类ID
	SubCategoryId string  `json:"sub_category_id"` // 子分类ID
	Score         float64 `json:"score"`           // 相似度分数
}

// MatchTagsForImage 为图片匹配标签
func (s *taggingService) MatchTagsForImage(ctx context.Context, imageID int64, modelName string) ([]MatchedTag, error) {
	// 1. 获取图片向量
	imageEmbedding, err := s.embeddingRepo.FindByImageAndModel(ctx, imageID, modelName)
	if err != nil {
		return nil, fmt.Errorf("获取图片向量失败: %v", err)
	}
	if imageEmbedding == nil {
		return nil, fmt.Errorf("图片向量不存在: image_id=%d, model=%s", imageID, modelName)
	}
	imageVector := []float32(imageEmbedding.Embedding)

	// 2. 使用数据库向量搜索确定最匹配的主分类
	mainCategory, err := s.findBestMainCategory(ctx, imageVector, modelName)
	if err != nil {
		return nil, err
	}

	logger.Debug("选定主分类",
		zap.Int64("image_id", imageID),
		zap.String("main_category", mainCategory))

	// 3. 使用数据库向量搜索获取匹配的标签
	tagResults, err := s.tagEmbeddingRepo.VectorSearchByCategory(
		ctx, modelName, mainCategory, imageVector,
		defaultSimilarityThreshold, 100, // 先获取足够多的候选
	)
	if err != nil {
		return nil, fmt.Errorf("向量搜索标签失败: %v", err)
	}

	// 4. 转换为 MatchedTag 并按子分类去重
	candidates := s.convertToMatchedTags(tagResults)
	results := s.deduplicateBySubCategory(candidates)

	logger.Info("标签匹配完成",
		zap.Int64("image_id", imageID),
		zap.String("main_category", mainCategory),
		zap.Int("candidates_count", len(candidates)),
		zap.Int("results_count", len(results)))

	return results, nil
}

// findBestMainCategory 使用数据库向量搜索选择最匹配的主分类
func (s *taggingService) findBestMainCategory(ctx context.Context, imageVector []float32, modelName string) (string, error) {
	// 使用数据库搜索主分类向量
	results, err := s.tagEmbeddingRepo.VectorSearchMainCategories(ctx, modelName, imageVector, mainCategories)
	if err != nil {
		return "", fmt.Errorf("搜索主分类向量失败: %v", err)
	}

	if len(results) == 0 {
		return "", fmt.Errorf("无法确定主分类，可能没有主分类向量")
	}

	// 第一个结果就是最相似的（已按距离升序排列）
	bestCategory := results[0].TagEmbedding.Tag.SourceCategoryId

	logger.Debug("主分类向量搜索结果",
		zap.String("best_category", *bestCategory),
		zap.Float64("similarity", results[0].Similarity))

	for _, r := range results {
		logger.Debug("主分类相似度",
			zap.String("category", *r.TagEmbedding.Tag.SourceCategoryId),
			zap.Float64("similarity", r.Similarity),
			zap.Float64("distance", r.Distance))
	}

	return *bestCategory, nil
}

// convertToMatchedTags 将向量搜索结果转换为 MatchedTag
func (s *taggingService) convertToMatchedTags(results []repository.TagEmbeddingWithDistance) []MatchedTag {
	var candidates []MatchedTag

	for _, r := range results {
		if r.TagEmbedding == nil || r.TagEmbedding.Tag == nil {
			continue
		}
		tag := r.TagEmbedding.Tag
		subCategoryId := ""
		if tag.SubCategoryId != nil {
			subCategoryId = *tag.SubCategoryId
		}
		tagNameEn := ""
		if tag.NameEn != nil {
			tagNameEn = *tag.NameEn
		}
		candidates = append(candidates, MatchedTag{
			TagID:         r.TagEmbedding.TagID,
			TagName:       tag.Name,
			TagNameEn:     tagNameEn,
			CategoryId:    *r.TagEmbedding.Tag.SubCategoryId,
			SubCategoryId: subCategoryId,
			Score:         r.Similarity,
		})
	}

	return candidates
}

// deduplicateBySubCategory 按子分类去重，每个子分类只保留最高分
func (s *taggingService) deduplicateBySubCategory(candidates []MatchedTag) []MatchedTag {
	seen := make(map[string]bool)
	var results []MatchedTag

	// candidates 已按分数降序排列，所以第一个遇到的就是该子分类的最高分
	for _, tag := range candidates {
		key := tag.SubCategoryId
		if key == "" {
			// 如果没有子分类，用标签名作为唯一标识（允许所有无子分类的标签）
			key = tag.TagName
		}
		if !seen[key] {
			seen[key] = true
			results = append(results, tag)
		}
	}

	// 限制最大返回数量
	if len(results) > defaultMaxTagsPerCategory {
		return results[:defaultMaxTagsPerCategory]
	}
	return results
}

// MatchTagIDsForImage 返回匹配的标签ID列表（简化版本，用于处理器）
func (s *taggingService) MatchTagIDsForImage(ctx context.Context, imageID int64, modelName string) ([]int64, error) {
	matchedTags, err := s.MatchTagsForImage(ctx, imageID, modelName)
	if err != nil {
		return nil, err
	}

	tagIDs := make([]int64, len(matchedTags))
	for i, tag := range matchedTags {
		tagIDs[i] = tag.TagID
	}
	return tagIDs, nil
}

func (s *taggingService) TaggingImage(ctx context.Context, imageID int64, modelName string) error {
	tagIDs, err := s.MatchTagIDsForImage(ctx, imageID, modelName)
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
