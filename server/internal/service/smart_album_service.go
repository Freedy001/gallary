package service

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"gallary/server/internal/llms"
	"gallary/server/internal/model"
	"gallary/server/internal/repository"
	"gallary/server/internal/websocket"
	"gallary/server/pkg/logger"

	"go.uber.org/zap"
)

// SmartAlbumService 智能相册服务接口
type SmartAlbumService interface {
	// SubmitSmartAlbumTask 提交智能相册任务（异步接口）
	SubmitSmartAlbumTask(ctx context.Context, req *GenerateSmartAlbumsRequest) (*model.SmartAlbumProgressVO, error)

	// GetCurrentTaskStatus 获取当前任务状态
	GetCurrentTaskStatus() *model.SmartAlbumProgressVO
}

// GenerateSmartAlbumsRequest 生成智能相册请求
type GenerateSmartAlbumsRequest struct {
	ModelName     string                  `json:"model_name" binding:"required"`
	Algorithm     string                  `json:"algorithm" binding:"required"` // 目前仅支持 "hdbscan"
	HDBSCANParams *model.HDBSCANParamsDTO `json:"hdbscan_params"`
}

// SmartAlbumTask 内存中的智能相册任务
type SmartAlbumTask struct {
	ID           int64
	Status       string // pending, collecting, clustering, creating, completed, failed
	Progress     int
	Message      string
	Error        *string
	AlbumIDs     []int64
	ClusterCount int
	NoiseCount   int
	TotalImages  int
	Params       *GenerateSmartAlbumsRequest
	CreatedAt    time.Time
}

type smartAlbumService struct {
	albumRepo     repository.AlbumRepository
	embeddingRepo repository.ImageEmbeddingRepository
	loadBalancer  *llms.ModelLoadBalancer
	notifier      websocket.Notifier

	// 内存任务管理
	taskMu      sync.RWMutex
	currentTask *SmartAlbumTask
	taskIDSeq   int64
}

// NewSmartAlbumService 创建智能相册服务实例
func NewSmartAlbumService(
	albumRepo repository.AlbumRepository,
	embeddingRepo repository.ImageEmbeddingRepository,
	loadBalancer *llms.ModelLoadBalancer,
	notifier websocket.Notifier,
) SmartAlbumService {
	return &smartAlbumService{
		albumRepo:     albumRepo,
		embeddingRepo: embeddingRepo,
		loadBalancer:  loadBalancer,
		notifier:      notifier,
	}
}

// SubmitSmartAlbumTask 提交智能相册任务（异步接口）
func (s *smartAlbumService) SubmitSmartAlbumTask(ctx context.Context, req *GenerateSmartAlbumsRequest) (*model.SmartAlbumProgressVO, error) {
	// 验证算法
	if req.Algorithm != "hdbscan" {
		return nil, fmt.Errorf("不支持的算法: %s，目前仅支持 hdbscan", req.Algorithm)
	}

	// 设置默认参数
	if req.HDBSCANParams == nil {
		req.HDBSCANParams = &model.HDBSCANParamsDTO{
			MinClusterSize:          5,
			ClusterSelectionEpsilon: 0.0,
			ClusterSelectionMethod:  "eom",
			Metric:                  "cosine",
			UMAPEnabled:             false,
			UMAPComponents:          50,
			UMAPNeighbors:           15,
		}
	}

	// 检查是否有正在执行的任务
	s.taskMu.Lock()
	if s.currentTask != nil && s.currentTask.Status != "completed" && s.currentTask.Status != "failed" {
		s.taskMu.Unlock()
		return nil, fmt.Errorf("已有任务正在执行中，请等待完成后再试")
	}

	// 创建新任务
	taskID := atomic.AddInt64(&s.taskIDSeq, 1)
	task := &SmartAlbumTask{
		ID:        taskID,
		Status:    "pending",
		Progress:  0,
		Message:   "任务已创建",
		Params:    req,
		CreatedAt: time.Now(),
	}
	s.currentTask = task
	s.taskMu.Unlock()

	logger.Info("智能相册任务已创建",
		zap.Int64("task_id", task.ID),
		zap.String("model_name", req.ModelName))

	// 推送初始状态
	s.notifyProgress(task)

	// 异步执行任务
	go s.processTask(context.Background(), task)

	return s.taskToVO(task), nil
}

// GetCurrentTaskStatus 获取当前任务状态
func (s *smartAlbumService) GetCurrentTaskStatus() *model.SmartAlbumProgressVO {
	s.taskMu.RLock()
	defer s.taskMu.RUnlock()

	if s.currentTask == nil {
		return nil
	}
	return s.taskToVO(s.currentTask)
}

// processTask 处理任务（后台执行）
func (s *smartAlbumService) processTask(ctx context.Context, task *SmartAlbumTask) {
	defer func() {
		if r := recover(); r != nil {
			errMsg := fmt.Sprintf("任务执行发生 panic: %v", r)
			s.updateTaskError(task, errMsg)
			logger.Error("智能相册任务 panic", zap.Int64("task_id", task.ID), zap.Any("panic", r))
		}
	}()

	// 1. 收集向量 (0-20%)
	s.updateTask(task, "collecting", 5, "正在收集图片向量...")
	embeddings, err := s.collectEmbeddings(ctx, task.Params.ModelName)
	if err != nil {
		s.updateTaskError(task, fmt.Sprintf("收集向量失败: %v", err))
		return
	}

	if len(embeddings) < 2 {
		s.updateTaskError(task, "图片数量不足，至少需要 2 张图片才能进行聚类")
		return
	}

	task.TotalImages = len(embeddings)
	s.updateTask(task, "collecting", 20, fmt.Sprintf("已收集 %d 张图片向量", len(embeddings)))

	// 2. 执行聚类 (20-80%)
	s.updateTask(task, "clustering", 25, "正在执行聚类分析...")
	result, err := s.executeClustering(ctx, task, embeddings)
	if err != nil {
		s.updateTaskError(task, fmt.Sprintf("聚类失败: %v", err))
		return
	}

	if result == nil || len(result.Clusters) == 0 {
		s.updateTaskError(task, "聚类结果为空，无法创建相册")
		return
	}

	task.ClusterCount = result.NClusters
	task.NoiseCount = len(result.NoiseImageIDs)
	s.updateTask(task, "clustering", 80, fmt.Sprintf("聚类完成，发现 %d 个分组", result.NClusters))

	// 3. 创建相册 (80-100%)
	s.updateTask(task, "creating", 85, "正在创建相册...")
	albumIDs, err := s.createAlbumsFromClusters(ctx, result)
	if err != nil {
		s.updateTaskError(task, fmt.Sprintf("创建相册失败: %v", err))
		return
	}

	// 4. 完成
	task.AlbumIDs = albumIDs
	s.updateTask(task, "completed", 100, fmt.Sprintf("已创建 %d 个智能相册", len(albumIDs)))

	logger.Info("智能相册任务完成",
		zap.Int64("task_id", task.ID),
		zap.Int("album_count", len(albumIDs)),
		zap.Int("noise_count", task.NoiseCount))
}

// collectEmbeddings 收集图片向量
func (s *smartAlbumService) collectEmbeddings(ctx context.Context, modelName string) ([]*model.ImageEmbedding, error) {
	// 获取所有该模型的图片向量
	embeddings, err := s.embeddingRepo.GetAllEmbeddingsByModel(ctx, modelName)
	if err != nil {
		return nil, fmt.Errorf("查询向量失败: %w", err)
	}
	return embeddings, nil
}

// executeClustering 执行聚类（使用 gRPC 流式调用）
func (s *smartAlbumService) executeClustering(ctx context.Context, task *SmartAlbumTask, embeddings []*model.ImageEmbedding) (*llms.ClusterResult, error) {
	// 获取 SelfClient
	client, err := s.loadBalancer.GetClientByName(task.Params.ModelName)
	if err != nil {
		return nil, fmt.Errorf("获取模型客户端失败: %w", err)
	}

	selfClient, ok := client.(llms.SelfClient)
	if !ok {
		return nil, fmt.Errorf("模型 %s 不支持聚类功能", task.Params.ModelName)
	}

	// 构建请求
	req := s.buildClusterRequest(task, embeddings)

	// 创建进度通道
	progressChan := make(chan *llms.ClusterProgress, 10)

	var result *llms.ClusterResult
	var wg sync.WaitGroup

	// 启动 goroutine 处理进度
	wg.Add(1)
	go func() {
		defer wg.Done()
		for progress := range progressChan {
			// 映射进度到 25-80% 区间
			mappedProgress := 25 + (progress.Progress * 55 / 100)
			s.updateTask(task, "clustering", mappedProgress, progress.Message)

			if progress.Result != nil {
				result = progress.Result
			}
		}
	}()

	// 调用 gRPC 流式聚类
	err = selfClient.ClusterStream(ctx, req, progressChan)

	// 等待进度处理完成
	wg.Wait()

	if err != nil {
		return nil, err
	}

	return result, nil
}

// buildClusterRequest 构建聚类请求
func (s *smartAlbumService) buildClusterRequest(task *SmartAlbumTask, embeddings []*model.ImageEmbedding) *llms.ClusterStreamRequest {
	// 转换嵌入向量
	embeddingVectors := make([][]float32, len(embeddings))
	imageIDs := make([]int64, len(embeddings))
	for i, e := range embeddings {
		embeddingVectors[i] = e.Embedding
		imageIDs[i] = e.ImageID
	}

	params := task.Params.HDBSCANParams

	return &llms.ClusterStreamRequest{
		Embeddings: embeddingVectors,
		ImageIDs:   imageIDs,
		TaskID:     task.ID,
		HDBSCANParams: &llms.HDBSCANParams{
			MinClusterSize:          params.MinClusterSize,
			MinSamples:              params.MinSamples,
			ClusterSelectionEpsilon: float32(params.ClusterSelectionEpsilon),
			ClusterSelectionMethod:  params.ClusterSelectionMethod,
			Metric:                  params.Metric,
		},
		UMAPParams: &llms.UMAPParams{
			Enabled:     params.UMAPEnabled,
			NComponents: params.UMAPComponents,
			NNeighbors:  params.UMAPNeighbors,
			MinDist:     0.1, // 默认值
		},
	}
}

// createAlbumsFromClusters 根据聚类结果创建相册
func (s *smartAlbumService) createAlbumsFromClusters(ctx context.Context, result *llms.ClusterResult) ([]int64, error) {
	if result == nil || len(result.Clusters) == 0 {
		return nil, fmt.Errorf("聚类结果为空")
	}

	albumIDs := make([]int64, 0, len(result.Clusters))

	// 获取起始编号
	startNumber, err := s.getNextSmartAlbumNumber(ctx)
	if err != nil {
		startNumber = 1
	}

	for i, cluster := range result.Clusters {
		if len(cluster.ImageIDs) == 0 {
			continue
		}

		// 创建相册 (使用 model.Tag，因为 AlbumRepository 使用 Tag)
		albumName := fmt.Sprintf("智能相册 #%d", startNumber+i)
		var coverImageID *int64
		if len(cluster.ImageIDs) > 0 {
			coverImageID = &cluster.ImageIDs[0]
		}

		album := &model.Tag{
			Name: albumName,
			Type: model.TagTypeAlbum,
			Metadata: &model.AlbumMetadata{
				IsSmartAlbum: true,
				CoverImageID: coverImageID,
			},
		}

		err := s.albumRepo.Create(ctx, album)
		if err != nil {
			logger.Warn("创建智能相册失败",
				zap.String("name", albumName),
				zap.Error(err))
			continue
		}

		// 添加图片到相册
		if err := s.albumRepo.AddImages(ctx, album.ID, cluster.ImageIDs); err != nil {
			logger.Warn("添加图片到相册失败",
				zap.Int64("album_id", album.ID),
				zap.Error(err))
		}

		albumIDs = append(albumIDs, album.ID)
	}

	return albumIDs, nil
}

// getNextSmartAlbumNumber 获取下一个智能相册编号
func (s *smartAlbumService) getNextSmartAlbumNumber(ctx context.Context) (int, error) {
	// 查询现有智能相册的最大编号
	isSmartTrue := true
	albums, _, err := s.albumRepo.List(ctx, 1, 1000, &isSmartTrue)
	if err != nil {
		return 1, nil
	}

	maxNumber := 0
	re := regexp.MustCompile(`智能相册 #(\d+)`)

	for _, album := range albums {
		matches := re.FindStringSubmatch(album.Name)
		if len(matches) == 2 {
			if num, err := strconv.Atoi(matches[1]); err == nil && num > maxNumber {
				maxNumber = num
			}
		}
	}

	return maxNumber + 1, nil
}

// updateTask 更新任务状态
func (s *smartAlbumService) updateTask(task *SmartAlbumTask, status string, progress int, message string) {
	s.taskMu.Lock()
	task.Status = status
	task.Progress = progress
	task.Message = message
	s.taskMu.Unlock()

	s.notifyProgress(task)
}

// updateTaskError 更新任务错误状态
func (s *smartAlbumService) updateTaskError(task *SmartAlbumTask, errMsg string) {
	s.taskMu.Lock()
	task.Status = "failed"
	task.Error = &errMsg
	task.Message = errMsg
	s.taskMu.Unlock()

	s.notifyProgress(task)

	logger.Error("智能相册任务失败",
		zap.Int64("task_id", task.ID),
		zap.String("error", errMsg))
}

// notifyProgress 通过 WebSocket 推送进度
func (s *smartAlbumService) notifyProgress(task *SmartAlbumTask) {
	if s.notifier == nil {
		return
	}

	s.notifier.NotifySmartAlbumProgress(s.taskToVO(task))
}

// taskToVO 将任务转换为 VO
func (s *smartAlbumService) taskToVO(task *SmartAlbumTask) *model.SmartAlbumProgressVO {
	return &model.SmartAlbumProgressVO{
		TaskID:       task.ID,
		Status:       task.Status,
		Progress:     task.Progress,
		Message:      task.Message,
		Error:        task.Error,
		AlbumIDs:     task.AlbumIDs,
		ClusterCount: task.ClusterCount,
		NoiseCount:   task.NoiseCount,
		TotalImages:  task.TotalImages,
	}
}
