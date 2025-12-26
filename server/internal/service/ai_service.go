package service

import (
	"context"
	"fmt"
	"gallary/server/internal/llms"
	"gallary/server/internal/websocket"
	"io"
	"sync"
	"time"

	"gallary/server/internal/model"
	"gallary/server/internal/repository"
	"gallary/server/pkg/logger"

	"go.uber.org/zap"
)

// AIService AI 服务接口
type AIService interface {
	TestConnection(ctx context.Context, id string) error

	// 队列管理
	GetQueueStatus(ctx context.Context) (*model.AIQueueStatus, error)
	GetQueueDetail(ctx context.Context, queueID int64, page, pageSize int) (*model.AIQueueDetail, error)

	// 单张图片操作
	RetryTaskImage(ctx context.Context, taskImageID int64) error
	IgnoreTaskImage(ctx context.Context, taskImageID int64) error

	// 批量操作
	RetryQueueFailedImages(ctx context.Context, queueID int64) error

	// 处理器
	StartProcessor(ctx context.Context)
	StopProcessor()

	// 搜索
	// 图片搜索（以图搜图，支持图片+文本混合查询）
	SemanticSearchWithinIDs(ctx context.Context, imageData []byte, text string, modelName string, candidateIDs []int64, limit int) ([]*model.Image, error)

	// 获取可用的嵌入模型列表
	GetEmbeddingModels(ctx context.Context) ([]string, error)
}

type aiService struct {
	taskRepo       repository.AITaskRepository
	embeddingRepo  repository.EmbeddingRepository
	imageRepo      repository.ImageRepository
	settingService SettingService

	loadBalancer *llms.ModelLoadBalancer
	notifier     websocket.Notifier

	processorCtx    context.Context
	processorCancel context.CancelFunc
	processorWg     sync.WaitGroup
	running         bool
	runningMu       sync.Mutex
}

// NewAIService 创建 AI 服务实例
func NewAIService(
	taskRepo repository.AITaskRepository,
	embeddingRepo repository.EmbeddingRepository,
	imageRepo repository.ImageRepository,
	settingService SettingService,
	notifier websocket.Notifier,
) AIService {
	return &aiService{
		taskRepo:       taskRepo,
		embeddingRepo:  embeddingRepo,
		imageRepo:      imageRepo,
		settingService: settingService,
		loadBalancer:   llms.NewModelLoadBalancer(settingService.GetStorageManager()),
		notifier:       notifier,
	}
}

// ================== 连接测试 ==================

// TestConnection 测试连接
func (s *aiService) TestConnection(ctx context.Context, id string) error {
	client, _, err := s.loadBalancer.GetClientByID(id)
	if err != nil {
		return err
	}
	return client.TestConnection(ctx)
}

// ================== 队列管理 ==================

// GetQueueStatus 获取队列状态
func (s *aiService) GetQueueStatus(ctx context.Context) (*model.AIQueueStatus, error) {
	return s.taskRepo.GetQueueStatus(ctx)
}

// GetQueueDetail 获取队列详情（含失败图片列表）
func (s *aiService) GetQueueDetail(ctx context.Context, queueID int64, page, pageSize int) (*model.AIQueueDetail, error) {
	// 获取队列信息
	queue, err := s.taskRepo.FindQueueByID(ctx, queueID)
	if err != nil {
		return nil, fmt.Errorf("队列不存在: %v", err)
	}

	// 获取队列统计
	pending, _, failed, err := s.taskRepo.GetQueueStats(ctx, queueID)
	if err != nil {
		return nil, err
	}

	// 获取失败图片列表
	failedImages, totalFailed, err := s.taskRepo.GetFailedTaskImages(ctx, queueID, page, pageSize)
	if err != nil {
		return nil, err
	}

	// 转换为前端格式
	imageInfos := make([]model.AITaskImageInfo, len(failedImages))
	manager := s.settingService.GetStorageManager()
	for i, img := range failedImages {
		_, thumbnail := manager.ImageUrl(img.Image)
		imageInfos[i] = img.ToInfo(thumbnail)
	}

	return &model.AIQueueDetail{
		Queue: model.AIQueueInfo{
			ID:           queue.ID,
			QueueKey:     queue.QueueKey,
			TaskType:     queue.TaskType,
			ModelName:    queue.ModelName,
			Status:       queue.Status,
			PendingCount: pending,
			FailedCount:  failed,
		},
		FailedImages: imageInfos,
		TotalFailed:  totalFailed,
		Page:         page,
		PageSize:     pageSize,
	}, nil
}

// ================== 单张图片操作 ==================

// RetryTaskImage 重试单张图片
func (s *aiService) RetryTaskImage(ctx context.Context, taskImageID int64) error {
	return s.taskRepo.RetryTaskImage(ctx, taskImageID)
}

// IgnoreTaskImage 忽略单张图片（删除关联）
func (s *aiService) IgnoreTaskImage(ctx context.Context, taskImageID int64) error {
	return s.taskRepo.RemoveTaskImage(ctx, taskImageID)
}

// ================== 批量操作 ==================

// RetryQueueFailedImages 重试队列所有失败图片
func (s *aiService) RetryQueueFailedImages(ctx context.Context, queueID int64) error {
	return s.taskRepo.RetryQueueFailedImages(ctx, queueID)
}

// ================== 处理器 ==================

// StartProcessor 启动处理器
func (s *aiService) StartProcessor(ctx context.Context) {
	s.runningMu.Lock()
	if s.running {
		s.runningMu.Unlock()
		return
	}
	s.running = true
	s.processorCtx, s.processorCancel = context.WithCancel(ctx)
	s.runningMu.Unlock()

	s.processorWg.Add(2)
	go s.processLoop(s.syncQueueImages)
	go s.processLoop(s.processQueueImages)
}

// StopProcessor 停止处理器
func (s *aiService) StopProcessor() {
	s.runningMu.Lock()
	if !s.running {
		s.runningMu.Unlock()
		return
	}
	s.running = false
	s.runningMu.Unlock()

	if s.processorCancel != nil {
		s.processorCancel()
	}
	s.processorWg.Wait()
}

// processLoop 处理循环
func (s *aiService) processLoop(processor func()) {
	defer s.processorWg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.processorCtx.Done():
			return
		case <-ticker.C:
			processor()
		}
	}
}

// syncQueueImages 同步队列图片（检测未处理的图片并添加到队列）
// 按 ModelName 分组，为每个唯一的 ModelName 创建一个队列
// 同时清理已删除模型对应的队列
func (s *aiService) syncQueueImages() {
	ctx := context.Background()

	// 获取所有可用的嵌入模型
	models, err := s.loadBalancer.GetAllEmbeddingModels()
	if err != nil {
		logger.Error("获取 AI 配置失败", zap.Error(err))
		return
	}

	// 将可用模型放入 map 便于查找
	validModels := make(map[string]bool)
	for _, modelName := range models {
		validModels[modelName] = true
	}

	// 获取所有现有队列
	allQueues, err := s.taskRepo.GetAllQueues(ctx)
	if err != nil {
		logger.Error("获取所有队列失败", zap.Error(err))
	} else {
		// 清理已删除模型对应的队列
		for _, queue := range allQueues {
			if !validModels[queue.ModelName] {
				logger.Info("检测到无效模型队列，准备清理",
					zap.String("model_name", queue.ModelName),
					zap.Int64("queue_id", queue.ID))

				if err := s.taskRepo.DeleteQueueWithImages(ctx, queue.ID); err != nil {
					logger.Error("删除队列失败",
						zap.String("model_name", queue.ModelName),
						zap.Int64("queue_id", queue.ID),
						zap.Error(err))
				} else {
					logger.Info("成功删除无效模型队列",
						zap.String("model_name", queue.ModelName),
						zap.Int64("queue_id", queue.ID))
				}
			}
		}
	}

	// 为每个可用模型同步图片
	for _, modelName := range models {
		// 查找或创建队列
		queue, err := s.taskRepo.FindOrCreateQueue(ctx, model.AITaskTypeEmbedding, modelName)
		if err != nil {
			logger.Error("查找或创建队列失败", zap.String("model_name", modelName), zap.Error(err))
			continue
		}

		// 查询未计算向量的图片
		imageIDs, err := s.embeddingRepo.FindImagesWithoutEmbedding(ctx, modelName, 1000)
		if err != nil {
			logger.Error("查询未处理图片失败", zap.String("model_name", modelName), zap.Error(err))
			continue
		}

		if len(imageIDs) == 0 {
			continue
		}

		// 向队列添加图片（去重）
		added, err := s.taskRepo.AddImagesToQueue(ctx, queue.ID, queue.QueueKey, imageIDs)
		if err != nil {
			logger.Error("添加图片到队列失败", zap.String("model_name", modelName), zap.Error(err))
		} else if added > 0 {
			logger.Info("向队列添加图片",
				zap.String("model_name", modelName),
				zap.Int("added_count", added))
		}
	}
}

// processQueueImages 处理队列中的图片
func (s *aiService) processQueueImages() {
	ctx := context.Background()

	// 获取有待处理图片的队列
	queues, err := s.taskRepo.FindQueuesWithPendingImages(ctx, 1)
	if err != nil {
		logger.Error("获取待处理队列失败", zap.Error(err))
		return
	}

	if len(queues) == 0 {
		return
	}

	queue := queues[0]

	// 更新队列状态为处理中
	queue.Status = model.AIQueueStatusProcessing
	if err := s.taskRepo.UpdateQueue(ctx, queue); err != nil {
		logger.Error("更新队列状态失败", zap.Error(err))
		return
	}

	// 处理队列中的图片
	s.processQueue(ctx, queue)

	// 检查队列是否还有待处理图片
	pending, _, _, err := s.taskRepo.GetQueueStats(ctx, queue.ID)
	if err == nil && pending == 0 {
		// 队列空闲
		queue.Status = model.AIQueueStatusIdle
		_ = s.taskRepo.UpdateQueue(ctx, queue)
	}
}

// processQueue 处理单个队列
func (s *aiService) processQueue(ctx context.Context, queue *model.AIQueue) {
	// 获取待处理的图片
	taskImages, err := s.taskRepo.GetPendingTaskImages(ctx, queue.ID, 10)
	if err != nil {
		logger.Error("获取队列图片失败", zap.Error(err))
		return
	}

	// 验证 ModelName 可以获取 client
	_, _, err = s.loadBalancer.GetClientByName(queue.ModelName)
	if err != nil {
		logger.Error("获取模型客户端失败", zap.String("model_name", queue.ModelName), zap.Error(err))
		// 将所有待处理图片标记为失败
		errMsg := fmt.Sprintf("模型客户端不可用: %v", err)
		for _, taskImage := range taskImages {
			s.failTaskImage(ctx, taskImage, errMsg)
		}
		return
	}

	logger.Info("开始处理队列图片",
		zap.Int64("queue_id", queue.ID),
		zap.String("model_name", queue.ModelName),
		zap.Int("image_count", len(taskImages)))

	for _, taskImage := range taskImages {
		select {
		case <-s.processorCtx.Done():
			return
		default:
		}

		s.processTaskImage(ctx, queue, taskImage)
	}
}

// processTaskImage 处理单张图片
func (s *aiService) processTaskImage(ctx context.Context, queue *model.AIQueue, taskImage *model.AITaskImage) {
	image := taskImage.Image
	if image == nil {
		image, _ = s.imageRepo.FindByID(ctx, taskImage.ImageID)
	}

	if image == nil {
		s.failTaskImage(ctx, taskImage, "图片不存在")
		return
	}

	var err error
	switch queue.TaskType {
	case model.AITaskTypeEmbedding:
		err = s.processImageEmbeddingWithRetry(ctx, queue.ModelName, image)
	case model.AITaskTypeDescription:
		err = fmt.Errorf("描述生成功能尚未实现")
	default:
		err = fmt.Errorf("未知的任务类型: %s", queue.TaskType)
	}

	if err != nil {
		s.failTaskImage(ctx, taskImage, err.Error())
	} else {
		s.completeTaskImage(ctx, taskImage)
	}
}

// processImageEmbeddingWithRetry 处理图片嵌入（带重试，尝试所有提供商）
func (s *aiService) processImageEmbeddingWithRetry(ctx context.Context, modelName string, image *model.Image) error {
	return s.loadBalancer.TryAllProviders(modelName, func(client llms.ModelClient, modelConfig *model.ModelConfig) error {
		return s.processImageEmbedding(ctx, image, client, modelConfig)
	})
}

// processImageEmbedding 处理图片嵌入
func (s *aiService) processImageEmbedding(ctx context.Context, image *model.Image, client llms.ModelClient, modelConfig *model.ModelConfig) error {
	if !client.SupportEmbedding() {
		return fmt.Errorf("模型 %s 不支持嵌入", modelConfig.ModelName)
	}

	// 从存储读取图片数据
	imageData, err := s.readImageData(ctx, image)
	if err != nil {
		return fmt.Errorf("读取图片数据失败: %v", err)
	}

	// 检查是否支持同时计算（自托管模型）
	if client.SupportsEmbeddingWithAesthetics() {
		embedding, score, err := client.EmbeddingWithAesthetics(ctx, imageData)
		if err != nil {
			return err
		}

		// 保存嵌入向量
		embeddingModel := &model.ImageEmbedding{
			ImageID:   image.ID,
			ModelID:   modelConfig.ID,
			ModelName: modelConfig.ModelName,
			Dimension: len(embedding),
			Embedding: model.Vector(embedding),
		}
		if err := s.embeddingRepo.Save(ctx, embeddingModel); err != nil {
			return err
		}

		// 更新图片评分
		image.AIScore = &score

		return s.imageRepo.Update(ctx, image)
	}

	// 普通嵌入处理
	embedding, err := client.Embedding(ctx, imageData, "")
	if err != nil {
		return err
	}

	// 保存嵌入
	embeddingModel := &model.ImageEmbedding{
		ImageID:   image.ID,
		ModelID:   modelConfig.ID,
		ModelName: modelConfig.ModelName,
		Dimension: len(embedding),
		Embedding: model.Vector(embedding),
	}

	return s.embeddingRepo.Save(ctx, embeddingModel)
}

// readImageData 从存储读取图片数据
func (s *aiService) readImageData(ctx context.Context, image *model.Image) ([]byte, error) {
	storageManager := s.settingService.GetStorageManager()
	reader, err := storageManager.Download(ctx, image.StorageId, image.StoragePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// ================== 语义搜索 ==================

// ImageSearchWithinIDs 通过图片（可选文本）进行语义搜索
func (s *aiService) SemanticSearchWithinIDs(ctx context.Context, imageData []byte, text string, modelName string, candidateIDs []int64, limit int) ([]*model.Image, error) {
	if candidateIDs != nil && len(candidateIDs) == 0 {
		return []*model.Image{}, nil
	}

	// 通过 ModelName 获取 client（支持负载均衡）
	client, modelConfig, err := s.loadBalancer.GetClientByName(modelName)
	if err != nil {
		return nil, err
	}

	// 获取查询向量（图片+文本混合）
	queryEmbedding, err := client.Embedding(ctx, imageData, text)
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %v", err)
	}

	// 在指定ID范围内执行向量搜索
	results, err := s.embeddingRepo.VectorSearchWithinIDs(ctx, modelConfig.ModelName, queryEmbedding, candidateIDs, limit)
	if err != nil {
		return nil, err
	}

	// 提取图片
	images := make([]*model.Image, len(results))
	for i, r := range results {
		images[i] = r.Embedding.Image
	}

	return images, nil
}

// ================== 辅助方法 ==================

func (s *aiService) failTaskImage(ctx context.Context, taskImage *model.AITaskImage, errMsg string) {
	logger.Info("图片 AI 任务失败: " + errMsg)

	taskImage.Status = model.AITaskImageStatusFailed
	taskImage.Error = &errMsg
	err := s.taskRepo.UpdateTaskImage(ctx, taskImage)
	if err != nil {
		logger.Error("操作数据库失败", zap.Error(err))
	}

	// 通知状态变化
	s.notifyStatusChange(ctx)
}

func (s *aiService) completeTaskImage(ctx context.Context, taskImage *model.AITaskImage) {
	// 成功后删除关联记录
	err := s.taskRepo.RemoveTaskImage(ctx, taskImage.ID)
	if err != nil {
		logger.Error("删除任务图片关联失败", zap.Error(err))
	}

	// 通知状态变化
	s.notifyStatusChange(ctx)
}

// notifyStatusChange 通知队列状态变化
func (s *aiService) notifyStatusChange(ctx context.Context) {
	if s.notifier == nil {
		return
	}

	// 获取最新状态并推送
	status, err := s.GetQueueStatus(ctx)
	if err == nil && status != nil {
		s.notifier.NotifyAIQueueStatus(status)
	}
}

// GetEmbeddingModels 获取可用的嵌入模型列表
func (s *aiService) GetEmbeddingModels(ctx context.Context) ([]string, error) {
	return s.loadBalancer.GetAllEmbeddingModels()
}
