package service

import (
	"context"
	"fmt"
	"gallary/server/internal"
	"gallary/server/internal/llms"
	"gallary/server/internal/websocket"
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

	// 单个任务项操作
	RetryTaskItem(ctx context.Context, taskItemID int64) error
	IgnoreTaskItem(ctx context.Context, taskItemID int64) error

	// 批量操作
	RetryQueueFailedItems(ctx context.Context, queueID int64) error

	// 处理器
	StartProcessor(ctx context.Context)
	StopProcessor()

	// 搜索
	// 图片搜索（以图搜图，支持图片+文本混合查询）
	SemanticSearchWithinIDs(ctx context.Context, imageData []byte, text string, modelName string, candidateIDs []int64, limit int) ([]*model.Image, error)

	// 获取可用的嵌入模型列表
	GetEmbeddingModels(ctx context.Context) ([]string, error)

	// 获取支持 ChatCompletion 的模型列表
	GetChatCompletionModels(ctx context.Context) ([]string, error)

	// 优化提示词
	OptimizePrompt(ctx context.Context, query string) (string, error)
}

type aiService struct {
	taskRepo       repository.AITaskRepository
	embeddingRepo  repository.ImageEmbeddingRepository
	imageRepo      repository.ImageRepository
	tagRepo        repository.TagRepository
	settingService SettingService
	taggingService TaggingService

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
	embeddingRepo repository.ImageEmbeddingRepository,
	imageRepo repository.ImageRepository,
	tagRepo repository.TagRepository,
	settingService SettingService,
	taggingService TaggingService,
	loadBalancer *llms.ModelLoadBalancer,
	notifier websocket.Notifier,
) AIService {
	return &aiService{
		taskRepo:       taskRepo,
		embeddingRepo:  embeddingRepo,
		imageRepo:      imageRepo,
		tagRepo:        tagRepo,
		settingService: settingService,
		taggingService: taggingService,
		loadBalancer:   loadBalancer,
		notifier:       notifier,
	}
}

// ================== 连接测试 ==================

// TestConnection 测试连接
func (s *aiService) TestConnection(ctx context.Context, id string) error {
	client, err := s.loadBalancer.GetClientByID(id)
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

// GetQueueDetail 获取队列详情（含失败项目列表）
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

	// 获取失败项目列表
	failedItems, totalFailed, err := s.taskRepo.GetFailedTaskItems(ctx, queueID, page, pageSize)
	if err != nil {
		return nil, err
	}

	// 转换为前端格式
	itemInfos := make([]model.AITaskItemInfo, len(failedItems))
	for i, item := range failedItems {
		itemInfos[i] = item.ToInfo()
	}

	// 批量填充 ItemName 和 ItemThumb
	s.batchFillTaskItemInfo(ctx, queue.TaskType, itemInfos)

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
		FailedItems: itemInfos,
		TotalFailed: totalFailed,
		Page:        page,
		PageSize:    pageSize,
	}, nil
}

// batchFillTaskItemInfo 批量填充项目的名称和缩略图
func (s *aiService) batchFillTaskItemInfo(ctx context.Context, taskType model.TaskType, items []model.AITaskItemInfo) {
	if len(items) == 0 {
		return
	}

	// 收集所有 ItemID
	itemIDs := make([]int64, len(items))
	for i, item := range items {
		itemIDs[i] = item.ItemID
	}

	switch taskType {
	case model.ImageEmbeddingTaskType, model.AestheticScoringTaskType:
		// 图片类任务：批量加载图片信息
		images, err := s.imageRepo.FindByIDs(ctx, itemIDs)
		if err != nil {
			logger.Debug("批量加载图片信息失败", zap.Error(err))
			return
		}

		// 构建 ID -> Image 映射
		imageMap := make(map[int64]*model.Image, len(images))
		for _, img := range images {
			imageMap[img.ID] = img
		}

		// 获取 storageManager
		storageManager := s.settingService.GetStorageManager()

		// 填充信息
		for i := range items {
			if img, ok := imageMap[items[i].ItemID]; ok {
				items[i].ItemName = img.OriginalName
				if storageManager != nil {
					_, thumbURL := storageManager.ImageUrl(img)
					items[i].ItemThumb = thumbURL
				}
			}
		}

	case model.TagEmbeddingTaskType:
		// 标签类任务：批量加载标签信息
		tags, err := s.tagRepo.FindByIDs(ctx, itemIDs)
		if err != nil {
			logger.Debug("批量加载标签信息失败", zap.Error(err))
			return
		}

		// 构建 ID -> Tag 映射
		tagMap := make(map[int64]*model.Tag, len(tags))
		for _, tag := range tags {
			tagMap[tag.ID] = tag
		}

		// 填充信息
		for i := range items {
			if tag, ok := tagMap[items[i].ItemID]; ok {
				items[i].ItemName = tag.Name
			}
		}
	}
}

// ================== 单个任务项操作 ==================

// RetryTaskItem 重试单个任务项
func (s *aiService) RetryTaskItem(ctx context.Context, taskItemID int64) error {
	return s.taskRepo.RetryTaskItem(ctx, taskItemID)
}

// IgnoreTaskItem 忽略单个任务项（删除关联）
func (s *aiService) IgnoreTaskItem(ctx context.Context, taskItemID int64) error {
	return s.taskRepo.RemoveTaskItem(ctx, taskItemID)
}

// ================== 批量操作 ==================

// RetryQueueFailedItems 重试队列所有失败项目
func (s *aiService) RetryQueueFailedItems(ctx context.Context, queueID int64) error {
	return s.taskRepo.RetryQueueFailedItems(ctx, queueID)
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
	go s.processLoop(s.aiTaskAdder)
	go s.processLoop(s.processQueueItems)
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

// aiTaskAdder 同步队列任务（检测未处理的项目并添加到队列）
// 遍历所有注册的处理器，为每个模型创建队列
// 同时清理已删除模型对应的队列
func (s *aiService) aiTaskAdder() {
	ctx := context.Background()

	// ========== 获取所有可用的嵌入模型 ==========
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

	// ========== 清理无效队列 ==========
	allQueues, err := s.taskRepo.GetAllQueues(ctx)
	if err != nil {
		logger.Error("获取所有队列失败", zap.Error(err))
	} else {
		for _, queue := range allQueues {
			if !validModels[queue.ModelName] {
				logger.Info("检测到无效模型队列，准备清理",
					zap.String("model_name", queue.ModelName),
					zap.Int64("queue_id", queue.ID))

				if err := s.taskRepo.DeleteQueueWithItems(ctx, queue.ID); err != nil {
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
		s.notifyStatusChange(ctx)
	}

	// ========== 4. 遍历所有处理器，为每个模型同步任务项 ==========
	for _, processor := range GetAllProcessors() {
		taskType := processor.TaskType()

		for _, modelName := range models {
			// 检查模型是否支持此任务类型
			client, err := s.loadBalancer.GetClientByName(modelName)
			if err != nil || !processor.SupportedBy(client) {
				continue
			}

			// 查找或创建队列
			queue, err := s.taskRepo.FindOrCreateQueue(ctx, taskType, modelName)
			if err != nil {
				logger.Error("查找或创建队列失败",
					zap.String("task_type", string(taskType)),
					zap.String("model_name", modelName),
					zap.Error(err))
				continue
			}

			// 查询待处理项目
			itemIDs, err := processor.FindPendingItems(ctx, modelName, 1000)
			if err != nil {
				logger.Error("查询待处理项目失败",
					zap.String("task_type", string(taskType)),
					zap.String("model_name", modelName),
					zap.Error(err))
				continue
			}

			if len(itemIDs) == 0 {
				continue
			}

			// 添加到队列
			added, err := s.taskRepo.AddItemsToQueue(ctx, queue.ID, queue.QueueKey, itemIDs, processor.TaskType())
			if err != nil {
				logger.Error("添加项目到队列失败",
					zap.String("task_type", string(taskType)),
					zap.String("model_name", modelName),
					zap.Error(err))
			} else if added > 0 {
				logger.Info("向队列添加项目",
					zap.String("task_type", string(taskType)),
					zap.String("model_name", modelName),
					zap.Int("added_count", added))
				s.notifyStatusChange(ctx)
			}
		}
	}
}

// processQueueItems 处理队列中的任务项
func (s *aiService) processQueueItems() {
	ctx := context.Background()

	// 获取有待处理任务项的队列
	queues, err := s.taskRepo.FindQueuesWithPendingItems(ctx, 1)
	if err != nil {
		logger.Error("获取待处理队列失败", zap.Error(err))
		return
	}

	if len(queues) == 0 {
		return
	}

	queue := queues[0]
	for _, q := range queues {
		if q.TaskType == model.TagEmbeddingTaskType {
			queue = q //优先级最高
			break
		}
	}

	// 更新队列状态为处理中
	queue.Status = model.AIQueueStatusProcessing
	if err := s.taskRepo.UpdateQueue(ctx, queue); err != nil {
		logger.Error("更新队列状态失败", zap.Error(err))
		return
	}

	// 处理队列中的任务项
	s.processQueue(ctx, queue)

	// 检查队列是否还有待处理任务项
	pending, _, _, err := s.taskRepo.GetQueueStats(ctx, queue.ID)
	if err == nil && pending == 0 {
		// 队列空闲
		queue.Status = model.AIQueueStatusIdle
		_ = s.taskRepo.UpdateQueue(ctx, queue)
	}
}

// processQueue 处理单个队列
func (s *aiService) processQueue(ctx context.Context, queue *model.AIQueue) {
	// 获取待处理的任务项
	taskItems, err := s.taskRepo.GetPendingTaskItems(ctx, queue.ID, 1000)
	if err != nil {
		logger.Error("获取队列任务项失败", zap.Error(err))
		return
	}

	// 验证 ModelName 可以获取 client
	_, err = s.loadBalancer.GetClientByName(queue.ModelName)
	if err != nil {
		logger.Error("获取模型客户端失败", zap.String("model_name", queue.ModelName), zap.Error(err))
		// 将所有待处理任务项标记为失败
		errMsg := fmt.Sprintf("模型客户端不可用: %v", err)
		for _, taskItem := range taskItems {
			s.failTaskItem(ctx, taskItem, errMsg)
		}
		return
	}

	logger.Info("开始处理队列任务项",
		zap.Int64("queue_id", queue.ID),
		zap.String("model_name", queue.ModelName),
		zap.Int("item_count", len(taskItems)))

	for _, taskItem := range taskItems {
		select {
		case <-s.processorCtx.Done():
			return
		default:
		}

		// 获取处理器
		processor, ok := GetProcessor(queue.TaskType)
		if !ok {
			s.failTaskItem(ctx, taskItem, fmt.Sprintf("未知的任务类型: %s", queue.TaskType))
			continue
		}

		//根据队列的任务类型获取对应的处理器，并调用处理器的 ProcessItem 方法
		// 执行处理（带重试，尝试所有提供商）
		err := s.loadBalancer.TryAllProviders(queue.ModelName, func(client llms.ModelClient, config *model.ModelConfig, modelItem *model.ModelItem) error {
			if !processor.SupportedBy(client) {
				return fmt.Errorf("模型 %s 不支持任务类型 %s", modelItem.ModelName, queue.TaskType)
			}
			return processor.ProcessItem(ctx, taskItem.ItemID, client, config, modelItem)
		})

		if err != nil {
			s.failTaskItem(ctx, taskItem, err.Error())
		} else {
			s.completeTaskItem(ctx, taskItem)
		}
	}
}

// ================== 语义搜索 ==================

// ImageSearchWithinIDs 通过图片（可选文本）进行语义搜索
func (s *aiService) SemanticSearchWithinIDs(ctx context.Context, imageData []byte, text string, modelName string, candidateIDs []int64, limit int) ([]*model.Image, error) {
	if candidateIDs != nil && len(candidateIDs) == 0 {
		return []*model.Image{}, nil
	}

	// 通过 ModelName 获取 client（支持负载均衡）
	client, err := s.loadBalancer.GetClientByName(modelName)
	if err != nil {
		return nil, err
	}

	// 获取查询向量（图片+文本混合）
	queryEmbedding, err := client.Embedding(ctx, imageData, text)
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %v", err)
	}

	// 在指定ID范围内执行向量搜索（使用 modelItem.ModelName 作为内部标识）
	results, err := s.embeddingRepo.VectorSearchWithinIDs(ctx, modelName, queryEmbedding, candidateIDs, limit)
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
func (s *aiService) failTaskItem(ctx context.Context, taskItem *model.AITaskItem, errMsg string) {
	logger.Info("AI 任务失败: " + errMsg)

	taskItem.Status = model.AITaskItemStatusFailed
	taskItem.Error = &errMsg
	err := s.taskRepo.UpdateTaskItem(ctx, taskItem)
	if err != nil {
		logger.Error("操作数据库失败", zap.Error(err))
	}

	// 通知状态变化
	s.notifyStatusChange(ctx)
}

func (s *aiService) completeTaskItem(ctx context.Context, taskItem *model.AITaskItem) {
	// 成功后删除关联记录
	err := s.taskRepo.RemoveTaskItem(ctx, taskItem.ID)
	if err != nil {
		logger.Error("删除任务项关联失败", zap.Error(err))
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
func (s *aiService) GetEmbeddingModels(_ context.Context) ([]string, error) {
	models, err := s.loadBalancer.GetAllEmbeddingModels()
	if err != nil {
		return models, err
	}

	for i, m := range models {
		if m == internal.PlatConfig.GlobalConfig.DefaultSearchModelId {
			temp := models[i]
			models[i] = models[0]
			models[0] = temp
			break
		}
	}

	return s.loadBalancer.GetAllEmbeddingModels()
}

// GetChatCompletionModels 获取支持 ChatCompletion 的模型列表
func (s *aiService) GetChatCompletionModels(_ context.Context) ([]string, error) {
	return s.loadBalancer.GetAllChatCompletionModels()
}

// OptimizePrompt 优化提示词
func (s *aiService) OptimizePrompt(ctx context.Context, query string) (string, error) {
	if internal.PlatConfig.GlobalConfig == nil {
		return "", fmt.Errorf("未配置模型")
	}

	// 获取支持 ChatCompletion 的模型客户端
	client, err := s.loadBalancer.GetClientByID(internal.PlatConfig.GlobalConfig.DefaultPromptOptimizeModelId)
	if err != nil {
		return "", fmt.Errorf("获取模型失败: %v", err)
	}

	if !client.SupportChatCompletion() {
		return "", fmt.Errorf("模型不支持 ChatCompletion")
	}

	// 构建消息
	messages := []llms.ChatMessage{
		{
			Role:    "system",
			Content: internal.PlatConfig.GlobalConfig.PromptOptimizeSystemPrompt,
		},
		{
			Role:    "user",
			Content: query,
		},
	}

	return client.ChatCompletion(ctx, messages)
}
