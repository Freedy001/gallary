package ai_processors

import (
	"context"
	"fmt"
	"io"

	"gallary/server/internal/llms"
	"gallary/server/internal/model"
	"gallary/server/internal/repository"
	"gallary/server/internal/storage"
)

// AestheticScoringProcessor 美学评分处理器
type AestheticScoringProcessor struct {
	imageRepo      repository.ImageRepository
	storageManager *storage.StorageManager
}

// NewAestheticScoringProcessor 创建美学评分处理器
func NewAestheticScoringProcessor(
	imageRepo repository.ImageRepository,
	storageManager *storage.StorageManager,
) *AestheticScoringProcessor {
	return &AestheticScoringProcessor{
		imageRepo:      imageRepo,
		storageManager: storageManager,
	}
}

func (p *AestheticScoringProcessor) TaskType() model.TaskType {
	return model.AestheticScoringTaskType
}

func (p *AestheticScoringProcessor) FindPendingItems(ctx context.Context, _ string, limit int) ([]int64, error) {
	// 查找没有 AI 评分的图片
	return p.imageRepo.FindImagesWithoutAIScore(ctx, limit)
}

func (p *AestheticScoringProcessor) SupportedBy(client llms.ModelClient) bool {
	return client.SupportAesthetics()
}

func (p *AestheticScoringProcessor) ProcessItem(ctx context.Context, itemID int64, client llms.ModelClient, _ *model.ModelConfig, _ *model.ModelItem) error {
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

	// 3. 调用美学评分接口
	score, err := client.Aesthetics(ctx, imageData)
	if err != nil {
		return err
	}

	// 4. 更新图片评分
	image.AIScore = &score
	if err := p.imageRepo.Update(ctx, image); err != nil {
		return fmt.Errorf("更新图片评分失败: %v", err)
	}

	return nil
}

func (p *AestheticScoringProcessor) readImageData(ctx context.Context, image *model.Image) ([]byte, error) {
	reader, err := p.storageManager.Download(ctx, image.StorageId, image.StoragePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}
