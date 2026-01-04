package ai_processors

import (
	"context"
	"fmt"
	"gallary/server/internal/service"
	"gallary/server/pkg/logger"
	"io"

	"gallary/server/internal"
	"gallary/server/internal/llms"
	"gallary/server/internal/model"
	"gallary/server/internal/repository"
	"gallary/server/internal/storage"

	"go.uber.org/zap"
)

// ImageEmbeddingProcessor 图片向量嵌入处理器
type ImageEmbeddingProcessor struct {
	imageEmbeddingRepository repository.ImageEmbeddingRepository
	imageRepo                repository.ImageRepository
	storageManager           *storage.StorageManager
	taggingService           service.TaggingService
}

// NewEmbeddingProcessor 创建图片向量嵌入处理器
func NewEmbeddingProcessor(
	imageEmbeddingRepository repository.ImageEmbeddingRepository,
	imageRepo repository.ImageRepository,
	storageManager *storage.StorageManager,
	taggingService service.TaggingService,
) *ImageEmbeddingProcessor {
	return &ImageEmbeddingProcessor{
		imageEmbeddingRepository: imageEmbeddingRepository,
		imageRepo:                imageRepo,
		storageManager:           storageManager,
		taggingService:           taggingService,
	}
}

func (p *ImageEmbeddingProcessor) TaskType() model.TaskType {
	return model.ImageEmbeddingTaskType
}

func (p *ImageEmbeddingProcessor) FindPendingItems(ctx context.Context, modelName string, limit int) ([]int64, error) {
	return p.imageEmbeddingRepository.FindImagesWithoutEmbedding(ctx, modelName, limit)
}

func (p *ImageEmbeddingProcessor) SupportedBy(client llms.ModelClient) bool {
	return client.SupportEmbedding()
}

func (p *ImageEmbeddingProcessor) ProcessItem(ctx context.Context, itemID int64, client llms.ModelClient, config *model.ModelConfig, modelItem *model.ModelItem) error {
	// 1. 获取图片
	image, err := p.imageRepo.FindByID(ctx, itemID)
	if err != nil {
		return fmt.Errorf("获取图片失败: %v", err)
	}
	if image == nil {
		return fmt.Errorf("图片不存在: %d", itemID)
	}

	// 2. 读取图片数据
	imageData, err := p.readImageData(ctx, image)
	if err != nil {
		return fmt.Errorf("读取图片数据失败: %v", err)
	}

	// 3. 计算嵌入向量
	embedding, err := client.Embedding(ctx, imageData, "")
	if err != nil {
		return err
	}

	// 4. 保存嵌入向量（使用 modelItem.ModelName 作为内部标识）
	embeddingModel := &model.ImageEmbedding{
		ImageID:   image.ID,
		ModelID:   string(model.CreateModelId(config.ID, modelItem.ApiModelName)),
		ModelName: modelItem.ModelName,
		Dimension: len(embedding),
		Embedding: model.Vector(embedding),
	}
	if err := p.imageEmbeddingRepository.Save(ctx, embeddingModel); err != nil {
		return err
	}

	// 5. 自动打标签（如果是默认标签模型）
	if modelItem.ModelName == internal.PlatConfig.AIPo.GetDefaultTagModelName() {
		err := p.taggingService.TaggingImage(ctx, image.ID, modelItem.ModelName)
		if err != nil {
			logger.Error("添加图片时，自动打标签失败", zap.Error(err))
		}
	}

	return nil
}

func (p *ImageEmbeddingProcessor) readImageData(ctx context.Context, image *model.Image) ([]byte, error) {
	reader, err := p.storageManager.Download(ctx, image.StorageId, image.StoragePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}
