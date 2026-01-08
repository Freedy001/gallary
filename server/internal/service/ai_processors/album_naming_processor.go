package ai_processors

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"gallary/server/internal"
	"gallary/server/internal/llms"
	"gallary/server/internal/model"
	"gallary/server/internal/repository"
	"gallary/server/internal/service"
	"gallary/server/internal/storage"
	"gallary/server/pkg/logger"

	"go.uber.org/zap"
)

// AlbumNamingProcessor 相册 AI 命名处理器
type AlbumNamingProcessor struct {
	service.BaseProcessor[llms.LLMSClient]
	albumRepo      repository.AlbumRepository
	imageRepo      repository.ImageRepository
	storageManager *storage.StorageManager
}

// NewAlbumNamingProcessor 创建相册命名处理器
func NewAlbumNamingProcessor(
	albumRepo repository.AlbumRepository,
	imageRepo repository.ImageRepository,
	storageManager *storage.StorageManager,
) *AlbumNamingProcessor {
	return &AlbumNamingProcessor{
		albumRepo:      albumRepo,
		imageRepo:      imageRepo,
		storageManager: storageManager,
	}
}

func (p *AlbumNamingProcessor) TaskType() model.TaskType {
	return model.AlbumNamingTaskType
}

// FindPendingItems 相册命名是手动触发的，不自动查找待处理项
// 返回空列表，不会自动添加到队列
func (p *AlbumNamingProcessor) FindPendingItems(_ context.Context, _ string, _ int) ([]int64, error) {
	// 相册命名任务是手动触发的，不需要自动检测
	return []int64{}, nil
}

// ProcessItem 处理单个相册的 AI 命名
func (p *AlbumNamingProcessor) ProcessItem(ctx context.Context, itemID int64, client llms.ModelClient, config *model.ModelConfig, modelItem *model.ModelItem) error {
	// 1. 获取相册
	album, err := p.albumRepo.FindByID(ctx, itemID)
	if err != nil {
		return fmt.Errorf("获取相册失败: %v", err)
	}
	if album == nil {
		return fmt.Errorf("相册不存在: %d", itemID)
	}

	// 2. 选择代表性图片
	representativeImages, err := p.selectRepresentativeImages(ctx, album, modelItem.ModelName)
	if err != nil {
		return fmt.Errorf("选择代表性图片失败: %v", err)
	}
	if len(representativeImages) == 0 {
		return fmt.Errorf("相册中没有图片")
	}

	// 3. 读取图片数据
	imageDataList := make([][]byte, 0, len(representativeImages))
	for _, img := range representativeImages {
		data, err := p.readImageData(ctx, img)
		if err != nil {
			logger.Warn("读取图片失败，跳过", zap.Int64("image_id", img.ID), zap.Error(err))
			continue
		}
		imageDataList = append(imageDataList, data)
	}

	if len(imageDataList) == 0 {
		return fmt.Errorf("无法读取相册中的任何图片")
	}

	// 构建多模态消息
	messages := buildMultimodalMessage(
		imageDataList,
		p.getSystemPrompt(),
		p.buildUserPrompt(album, len(representativeImages)),
	)

	// 4. 调用 LLM 生成名称（多模态）
	name, err := p.Cast(client).ChatCompletion(ctx, messages)
	if err != nil {
		return fmt.Errorf("调用视觉模型失败: %v", err)
	}

	// 5. 清理生成的名称
	name = cleanAlbumName(name)
	if name == "" {
		return fmt.Errorf("生成的名称为空")
	}

	if name == "未识别到图像,请更换视觉模型" {
		return fmt.Errorf(name)
	}

	// 6. 更新相册名称
	album.Name = name
	if err := p.albumRepo.Update(ctx, album); err != nil {
		return fmt.Errorf("更新相册名称失败: %v", err)
	}

	logger.Info("相册命名成功", zap.Int64("album_id", album.ID), zap.String("name", name))
	return nil
}

// selectRepresentativeImages 选择相册的代表性图片
// 策略：优先使用封面图片，其次使用平均向量最接近的图片，最后随机选择
func (p *AlbumNamingProcessor) selectRepresentativeImages(ctx context.Context, album *model.Tag, modelName string) ([]*model.Image, error) {
	// 从配置获取最大图片数量，默认为 3
	maxImages := 3
	if internal.PlatConfig.GlobalConfig != nil && internal.PlatConfig.GlobalConfig.NamingMaxImages > 0 {
		maxImages = internal.PlatConfig.GlobalConfig.NamingMaxImages
	}

	var selectedImages []*model.Image
	selectedIDs := make(map[int64]bool)

	// 1. 优先使用用户设置的封面图片
	if album.Metadata != nil && album.Metadata.CoverImageID != nil {
		coverImage, err := p.albumRepo.GetCoverImage(ctx, *album.Metadata.CoverImageID)
		if err == nil && coverImage != nil {
			selectedImages = append(selectedImages, coverImage)
			selectedIDs[coverImage.ID] = true
		}
	}

	// 2. 尝试通过平均向量选择最接近中心的图片
	if len(selectedImages) < maxImages && modelName != "" {
		bestCoverID, err := p.albumRepo.FindBestCoverByAverageVector(ctx, album.ID, modelName)
		if err == nil && bestCoverID > 0 && !selectedIDs[bestCoverID] {
			images, err := p.imageRepo.FindByIDs(ctx, []int64{bestCoverID})
			if err == nil && len(images) > 0 {
				selectedImages = append(selectedImages, images[0])
				selectedIDs[bestCoverID] = true
			}
		}
	}

	// 3. 获取相册中的更多图片以补充
	if len(selectedImages) < maxImages {
		images, _, err := p.albumRepo.GetImages(ctx, album.ID, 1, 10)
		if err != nil {
			return selectedImages, nil // 返回已有的图片
		}

		// 选择评分最高的图片
		for _, img := range images {
			if len(selectedImages) >= maxImages {
				break
			}
			if !selectedIDs[img.ID] {
				selectedImages = append(selectedImages, img)
				selectedIDs[img.ID] = true
			}
		}
	}

	return selectedImages, nil
}

func (p *AlbumNamingProcessor) readImageData(ctx context.Context, image *model.Image) ([]byte, error) {
	reader, err := p.storageManager.Download(ctx, image.StorageId, image.ThumbnailPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func (p *AlbumNamingProcessor) getSystemPrompt() string {
	// 如果配置了自定义提示词，使用自定义提示词
	if internal.PlatConfig.GlobalConfig != nil && internal.PlatConfig.GlobalConfig.NamingSystemPrompt != "" {
		return internal.PlatConfig.GlobalConfig.NamingSystemPrompt
	}

	// 默认提示词
	return `你是一个专业的相册命名助手。请根据提供的图片内容，为相册生成一个简洁、准确、有意义的中文名称。

规则：
1. 只输出相册名称，不要输出任何其他内容
2. 名称长度控制在 2-8 个字符
3. 名称应该准确描述图片的主题或情感
4. 使用具体、有描述性的词语
5. 避免使用过于笼统的词语如"风景"、"照片"等

示例：
- 海边日落的图片 -> "海边日落"
- 可爱的猫咪图片 -> "萌宠时光"
- 城市夜景图片 -> "都市夜色"
- 家庭聚会图片 -> "温馨家宴"
- 山水风光图片 -> "山水画卷"`
}

func (p *AlbumNamingProcessor) buildUserPrompt(album *model.Tag, imageCount int) string {
	prompt := fmt.Sprintf("这是相册中的 %d 张代表性图片。", imageCount)

	// 如果相册有描述，添加到提示中
	if album.Metadata != nil && album.Metadata.Description != nil && *album.Metadata.Description != "" {
		prompt += fmt.Sprintf("\n相册描述：%s", *album.Metadata.Description)
	}

	prompt += "\n\n请根据这些图片的内容，为相册生成一个合适的中文名称。只输出名称，如果你没有看到任何图片请回复\"未识别到图像,请更换视觉模型\"，不要输出其他内容。"

	return prompt
}

// buildMultimodalMessage 构建多模态消息的辅助方法
// images: 图片二进制数据列表
// systemPrompt: 系统提示词
// userPrompt: 用户提示词
func buildMultimodalMessage(images [][]byte, systemPrompt, userPrompt string) []llms.ChatMessage {
	// 构建 system 消息
	systemMessage := llms.ChatMessage{
		Role:    "system",
		Content: systemPrompt,
	}

	// 构建 user 消息（包含图片和文本）
	userContent := make([]llms.ContentPart, 0, len(images)+1)

	// 添加图片
	for _, imageData := range images {
		if len(imageData) == 0 {
			continue
		}
		// 检测图片类型并转换为 base64 data URL
		mimeType := detectImageMimeType(imageData)
		base64Data := base64.StdEncoding.EncodeToString(imageData)
		dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)

		userContent = append(userContent, llms.ContentPart{
			Type: "image_url",
			ImageURL: &llms.ImageURL{
				URL:    dataURL,
				Detail: "low", // 使用 low 降低 token 消耗
			},
		})
	}

	// 添加文本提示
	userContent = append(userContent, llms.ContentPart{
		Type: "text",
		Text: userPrompt,
	})

	userMessage := llms.ChatMessage{
		Role:    "user",
		Content: userContent,
	}

	return []llms.ChatMessage{systemMessage, userMessage}
}

// detectImageMimeType 检测图片 MIME 类型
func detectImageMimeType(data []byte) string {
	if len(data) < 8 {
		return "image/jpeg"
	}

	// PNG: 89 50 4E 47 0D 0A 1A 0A
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}

	// JPEG: FF D8 FF
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}

	// GIF: 47 49 46 38
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return "image/gif"
	}

	// WebP: 52 49 46 46 ... 57 45 42 50
	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
		return "image/webp"
	}

	return "image/jpeg"
}

// cleanAlbumName 清理生成的相册名称
func cleanAlbumName(name string) string {
	// 去除前后空格
	name = strings.TrimSpace(name)

	// 去除引号
	name = strings.Trim(name, `"'"'""`)

	// 去除常见前缀
	prefixes := []string{"相册名称：", "名称：", "相册：", "Album:", "Name:"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			name = strings.TrimPrefix(name, prefix)
			name = strings.TrimSpace(name)
		}
	}

	// 只取第一行
	if idx := strings.Index(name, "\n"); idx != -1 {
		name = name[:idx]
	}

	// 限制长度（最多 50 个字符）
	if len(name) > 50 {
		name = name[:50]
	}

	return strings.TrimSpace(name)
}
