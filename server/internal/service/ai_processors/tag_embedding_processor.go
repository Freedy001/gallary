package ai_processors

import (
	"context"
	"fmt"
	"gallary/server/internal"
	"gallary/server/internal/llms"
	"gallary/server/internal/model"
	"gallary/server/internal/repository"
	"gallary/server/internal/service"
	"gallary/server/pkg/logger"

	"go.uber.org/zap"
)

// AITaskTypeTagEmbedding 标签向量嵌入任务类型
const AITaskTypeTagEmbedding = "tag_embedding"

// TagEmbeddingProcessor 标签向量嵌入处理器
type TagEmbeddingProcessor struct {
	service.BaseProcessor[llms.EmbeddingClient]
	tagRepo          repository.TagRepository
	taggingService   service.TaggingService
	tagEmbeddingRepo repository.TagEmbeddingRepository
}

// NewTagEmbeddingProcessor 创建标签向量嵌入处理器
func NewTagEmbeddingProcessor(
	tagRepo repository.TagRepository,
	taggingService service.TaggingService,
	tagEmbeddingRepo repository.TagEmbeddingRepository,
) *TagEmbeddingProcessor {
	return &TagEmbeddingProcessor{
		tagRepo:          tagRepo,
		taggingService:   taggingService,
		tagEmbeddingRepo: tagEmbeddingRepo,
	}
}

func (p *TagEmbeddingProcessor) TaskType() model.TaskType {
	return model.TagEmbeddingTaskType
}

func (p *TagEmbeddingProcessor) FindPendingItems(ctx context.Context, modelName string, limit int) ([]int64, error) {
	err := p.taggingService.SyncTagsIfChanged(ctx)
	if err != nil {
		return nil, err
	}

	// 查找需要生成向量的标签
	tags, err := p.tagRepo.FindTagsNeedingEmbedding(ctx, modelName)
	if err != nil {
		return nil, err
	}

	if len(tags) == 0 {
		p.tagImage(ctx, limit)
		return []int64{}, nil
	}

	arrLen := min(limit, len(tags))
	// 限制返回数量
	ids := make([]int64, 0, arrLen)
	for _, tag := range tags[0:arrLen] {
		ids = append(ids, tag.ID)
	}
	return ids, nil
}

func (p *TagEmbeddingProcessor) tagImage(ctx context.Context, limit int) {
	tagModelName := internal.PlatConfig.AIPo.GetDefaultTagModelName()
	if tagModelName == "" {
		return
	}

	ids, err := p.tagRepo.FindImagesWithoutTags(ctx, tagModelName, limit)
	if err != nil {
		logger.Error("获取未打标记的图片失败", zap.Error(err))
	}

	if len(ids) == 0 {
		return
	}

	logger.Info("开始给遗留的图片打标记", zap.Int("image_len", len(ids)))
	// 只对默认打标模型执行自动打标
	for _, imageID := range ids {
		err := p.taggingService.TaggingImage(ctx, imageID, tagModelName)
		if err != nil {
			logger.Error("添加图片时，自动打标签失败", zap.Error(err))
		}
	}
	return
}

func (p *TagEmbeddingProcessor) ProcessItem(ctx context.Context, itemID int64, client llms.ModelClient, config *model.ModelConfig, modelItem *model.ModelItem) error {
	// 1. 获取标签
	tag, err := p.tagRepo.FindByID(ctx, itemID)
	if err != nil {
		return fmt.Errorf("获取标签失败: %v", err)
	}
	if tag == nil {
		return fmt.Errorf("标签不存在: %d", itemID)
	}

	if tag.VectorDescription == nil || *tag.VectorDescription == "" {
		return fmt.Errorf("标签没有向量描述: %s", tag.Name)
	}

	// 2. 使用文本生成向量
	embedding, err := p.Cast(client).Embedding(ctx, nil, *tag.VectorDescription)
	if err != nil {
		return fmt.Errorf("生成标签向量失败: %v", err)
	}

	tagEmbedding := &model.TagEmbedding{
		TagID:     tag.ID,
		ModelName: modelItem.ModelName,
		ModelId:   string(model.CreateModelId(config.ID, modelItem.ApiModelName)),
		Dimension: len(embedding),
		Embedding: model.Vector(embedding),
	}

	return p.tagEmbeddingRepo.Save(ctx, tagEmbedding)
}
