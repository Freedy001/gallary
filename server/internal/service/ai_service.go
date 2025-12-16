package service

import (
	"context"
	"fmt"
	"gallary/server/internal/llms"
	"net/http"
	"sync"
	"sync/atomic"
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
	SemanticSearch(ctx context.Context, query string, modelName string, limit int) ([]*model.Image, error)
}

type aiService struct {
	taskRepo       repository.AITaskRepository
	embeddingRepo  repository.EmbeddingRepository
	imageRepo      repository.ImageRepository
	settingService SettingService

	httpClient *http.Client

	// 模型客户端缓存（按 ID 缓存）
	modelClients map[string]llms.ModelClient
	modelMu      sync.RWMutex

	// 负载均衡计数器（按 ModelName 分组）
	loadBalanceCounters map[string]*uint64
	lbMu                sync.RWMutex

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
) AIService {
	return &aiService{
		taskRepo:            taskRepo,
		embeddingRepo:       embeddingRepo,
		imageRepo:           imageRepo,
		settingService:      settingService,
		httpClient:          &http.Client{Timeout: 120 * time.Second},
		modelClients:        make(map[string]llms.ModelClient),
		loadBalanceCounters: make(map[string]*uint64),
	}
}

// ================== 模型客户端管理 ==================

// getModelClient 获取或创建模型客户端
func (s *aiService) getModelClient(modelConfig *model.ModelConfig) llms.ModelClient {
	s.modelMu.RLock()
	if client, exists := s.modelClients[modelConfig.ID]; exists {
		s.modelMu.RUnlock()
		return client
	}
	s.modelMu.RUnlock()

	s.modelMu.Lock()
	defer s.modelMu.Unlock()

	// 双重检查
	if client, exists := s.modelClients[modelConfig.ID]; exists {
		return client
	}

	// 创建新的客户端
	client := llms.CreateModelClient(modelConfig, s.httpClient, s.settingService.GetStorageManager())
	s.modelClients[modelConfig.ID] = client
	return client
}

// getModelClientByName 根据模型名称获取客户端（支持负载均衡）
func (s *aiService) getModelClientByName(ctx context.Context, modelName string) (llms.ModelClient, *model.ModelConfig, error) {
	config, err := s.settingService.GetAIConfig(ctx)
	if err != nil {
		return nil, nil, err
	}

	// 获取该模型名称对应的所有启用的模型配置
	models := config.FindModelsByName(modelName)
	if len(models) == 0 {
		return nil, nil, fmt.Errorf("未找到模型配置: %s", modelName)
	}

	// 负载均衡：轮询选择
	modelConfig := s.selectModelByLoadBalance(modelName, models)

	client := s.getModelClient(modelConfig)
	if client == nil {
		return nil, nil, fmt.Errorf("无法获取模型客户端: %s", modelConfig.ID)
	}

	return client, modelConfig, nil
}

// selectModelByLoadBalance 使用轮询算法选择模型
func (s *aiService) selectModelByLoadBalance(modelName string, models []*model.ModelConfig) *model.ModelConfig {
	if len(models) == 1 {
		return models[0]
	}

	s.lbMu.RLock()
	counter, exists := s.loadBalanceCounters[modelName]
	s.lbMu.RUnlock()

	if !exists {
		s.lbMu.Lock()
		// 双重检查
		if counter, exists = s.loadBalanceCounters[modelName]; !exists {
			var zero uint64
			counter = &zero
			s.loadBalanceCounters[modelName] = counter
		}
		s.lbMu.Unlock()
	}

	// 原子递增并取模
	idx := atomic.AddUint64(counter, 1) % uint64(len(models))
	return models[idx]
}

// ================== 连接测试 ==================

// TestConnection 测试连接
func (s *aiService) TestConnection(ctx context.Context, id string) error {
	config, err := s.settingService.GetAIConfig(ctx)
	if err != nil {
		return err
	}

	// 查找模型配置
	modelConfig := config.FindModelById(id)
	if modelConfig == nil {
		return fmt.Errorf("未找到模型配置: %s", id)
	}

	// 使用模型客户端测试连接
	client := s.getModelClient(modelConfig)
	if client == nil {
		return fmt.Errorf("无法获取模型客户端: %s", id)
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
	pending, processing, failed, err := s.taskRepo.GetQueueStats(ctx, queueID)
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
	for i, img := range failedImages {
		imageInfos[i] = img.ToInfo()
	}

	return &model.AIQueueDetail{
		Queue: model.AIQueueInfo{
			ID:              queue.ID,
			QueueKey:        queue.QueueKey,
			TaskType:        queue.TaskType,
			ModelName:       queue.ModelName,
			Status:          queue.Status,
			PendingCount:    pending,
			ProcessingCount: processing,
			FailedCount:     failed,
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
func (s *aiService) syncQueueImages() {
	ctx := context.Background()

	// 获取 AI 配置
	config, err := s.settingService.GetAIConfig(ctx)
	if err != nil {
		logger.Error("获取 AI 配置失败", zap.Error(err))
		return
	}

	// 获取所有启用的模型
	enabledModels := config.GetEnabledModels()
	if len(enabledModels) == 0 {
		return
	}

	// 按 ModelName 分组，每个 ModelName 只需要一个队列
	modelNames := make(map[string]bool)
	for _, modelConfig := range enabledModels {
		client := s.getModelClient(modelConfig)
		if client == nil || !client.SupportEmbedding() {
			continue
		}
		modelNames[modelConfig.ModelName] = true
	}

	for modelName := range modelNames {
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
	pending, processing, _, err := s.taskRepo.GetQueueStats(ctx, queue.ID)
	if err == nil && pending == 0 && processing == 0 {
		// 队列空闲
		queue.Status = model.AIQueueStatusIdle
		s.taskRepo.UpdateQueue(ctx, queue)
	}
}

// processQueue 处理单个队列
func (s *aiService) processQueue(ctx context.Context, queue *model.AIQueue) {
	// 验证 ModelName 可以获取 client
	_, _, err := s.getModelClientByName(ctx, queue.ModelName)
	if err != nil {
		logger.Error("获取模型客户端失败", zap.String("model_name", queue.ModelName), zap.Error(err))
		return
	}

	// 获取待处理的图片
	taskImages, err := s.taskRepo.GetPendingTaskImages(ctx, queue.ID, 10)
	if err != nil {
		logger.Error("获取队列图片失败", zap.Error(err))
		return
	}

	for _, taskImage := range taskImages {
		select {
		case <-s.processorCtx.Done():
			return
		default:
		}

		// 每次处理图片时重新获取 client，实现请求级负载均衡
		client, modelConfig, err := s.getModelClientByName(ctx, queue.ModelName)
		if err != nil {
			s.failTaskImage(ctx, taskImage, err.Error())
			continue
		}

		s.processTaskImage(ctx, queue, taskImage, client, modelConfig)
	}
}

// processTaskImage 处理单张图片
func (s *aiService) processTaskImage(ctx context.Context, queue *model.AIQueue, taskImage *model.AITaskImage, client llms.ModelClient, modelConfig *model.ModelConfig) {
	// 更新状态为处理中
	taskImage.Status = model.AITaskImageStatusProcessing
	_ = s.taskRepo.UpdateTaskImage(ctx, taskImage)

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
		err = s.processImageEmbedding(ctx, image, client, modelConfig)
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

// processImageEmbedding 处理图片嵌入
func (s *aiService) processImageEmbedding(ctx context.Context, image *model.Image, client llms.ModelClient, modelConfig *model.ModelConfig) error {
	if !client.SupportEmbedding() {
		return fmt.Errorf("模型 %s 不支持嵌入", modelConfig.ModelName)
	}

	// 检查是否支持同时计算（自托管模型）
	if client.SupportsEmbeddingWithAesthetics() {
		embedding, score, err := client.EmbeddingWithAesthetics(ctx, image)
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
	embedding, err := client.Embedding(ctx, image, "")
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

// ================== 语义搜索 ==================

// SemanticSearch 语义搜索
func (s *aiService) SemanticSearch(ctx context.Context, query string, modelName string, limit int) ([]*model.Image, error) {
	// 通过 ModelName 获取 client（支持负载均衡）
	client, modelConfig, err := s.getModelClientByName(ctx, modelName)
	if err != nil {
		return nil, err
	}

	// 获取查询向量
	queryEmbedding, err := client.Embedding(ctx, nil, query)
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %v", err)
	}

	// 执行向量搜索（使用 ModelName 而不是 ModelID）
	results, err := s.embeddingRepo.VectorSearchByModelName(ctx, modelConfig.ModelName, queryEmbedding, limit)
	if err != nil {
		return nil, err
	}

	// 提取图片
	images := make([]*model.Image, len(results))
	for i, r := range results {
		images[i] = r.Image
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
}

func (s *aiService) completeTaskImage(ctx context.Context, taskImage *model.AITaskImage) {
	// 成功后删除关联记录
	err := s.taskRepo.RemoveTaskImage(ctx, taskImage.ID)
	if err != nil {
		logger.Error("删除任务图片关联失败", zap.Error(err))
	}
}
