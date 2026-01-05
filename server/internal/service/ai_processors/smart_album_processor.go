package ai_processors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"gallary/server/internal/llms"
	"gallary/server/internal/model"
	"gallary/server/internal/repository"
	"gallary/server/internal/storage"
	"gallary/server/internal/websocket"
	"gallary/server/pkg/database"
	"gallary/server/pkg/logger"

	"go.uber.org/zap"
)

// SmartAlbumProcessor 智能相册处理器
// 注意：此处理器不同于标准 AITaskProcessor，它处理整批数据进行聚类
type SmartAlbumProcessor struct {
	albumRepo     repository.AlbumRepository
	embeddingRepo repository.ImageEmbeddingRepository
	taskRepo      repository.AITaskRepository
	loadBalancer  *llms.ModelLoadBalancer
	storage       *storage.StorageManager
	notifier      websocket.Notifier
	httpClient    *http.Client
}

// NewSmartAlbumProcessor 创建智能相册处理器
func NewSmartAlbumProcessor(
	albumRepo repository.AlbumRepository,
	embeddingRepo repository.ImageEmbeddingRepository,
	taskRepo repository.AITaskRepository,
	loadBalancer *llms.ModelLoadBalancer,
	storage *storage.StorageManager,
	notifier websocket.Notifier,
) *SmartAlbumProcessor {
	return &SmartAlbumProcessor{
		albumRepo:     albumRepo,
		embeddingRepo: embeddingRepo,
		taskRepo:      taskRepo,
		loadBalancer:  loadBalancer,
		storage:       storage,
		notifier:      notifier,
		httpClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (p *SmartAlbumProcessor) TaskType() model.TaskType {
	return model.SmartAlbumTaskType
}

// FindPendingItems 查找待处理的智能相册任务
func (p *SmartAlbumProcessor) FindPendingItems(ctx context.Context, modelName string, limit int) ([]int64, error) {
	tasks, err := p.taskRepo.GetPendingSmartAlbumTasks(ctx, limit)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	return ids, nil
}

// SupportedBy 智能相册任务需要嵌入模型支持
func (p *SmartAlbumProcessor) SupportedBy(client llms.ModelClient) bool {
	return client.SupportEmbedding()
}

// ProcessItem 处理单个智能相册任务
func (p *SmartAlbumProcessor) ProcessItem(ctx context.Context, itemID int64, client llms.ModelClient, config *model.ModelConfig, modelItem *model.ModelItem) error {
	// 获取任务
	taskItem, err := p.taskRepo.GetSmartAlbumTaskByID(ctx, itemID)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	// 解析配置
	var extra model.AITaskItemExtra
	if err := json.Unmarshal(taskItem.Extra, &extra); err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}

	// 更新状态为 collecting
	p.updateProgress(ctx, taskItem, &extra, model.AITaskItemStatusCollecting, 10, "收集向量数据中")

	// 1. 获取向量数据
	embeddings, err := p.embeddingRepo.GetAllEmbeddingsByModel(ctx, extra.ModelName)
	if err != nil {
		return p.failTask(ctx, taskItem, &extra, fmt.Sprintf("获取向量数据失败: %v", err))
	}

	if len(embeddings) < 2 {
		return p.failTask(ctx, taskItem, &extra, fmt.Sprintf("向量数据不足，至少需要 2 张图片，当前仅有 %d 张", len(embeddings)))
	}

	// 更新进度
	p.updateProgress(ctx, taskItem, &extra, model.AITaskItemStatusClustering, 20, "执行聚类分析中")

	// 2. 获取 Python 服务端点
	modelClient, err := p.loadBalancer.GetClientByName(extra.ModelName)
	if err != nil {
		return p.failTask(ctx, taskItem, &extra, fmt.Sprintf("获取模型客户端失败: %v", err))
	}
	endpoint := modelClient.GetConfig().Endpoint

	// 3. 调用 Python 聚类服务（同步方式）
	clusterResult, err := p.callClusteringService(ctx, endpoint, embeddings, extra.HDBSCANParams)
	if err != nil {
		return p.failTask(ctx, taskItem, &extra, fmt.Sprintf("聚类失败: %v", err))
	}

	logger.Info("聚类完成",
		zap.Int64("task_id", taskItem.ID),
		zap.Int("n_clusters", clusterResult.NClusters),
		zap.Int("noise_count", len(clusterResult.NoiseImageIDs)))

	// 更新进度
	p.updateProgress(ctx, taskItem, &extra, model.AITaskItemStatusCreating, 80, "创建相册中")

	// 4. 创建相册
	albums, err := p.createAlbumsFromClusterResult(ctx, &extra, clusterResult)
	if err != nil {
		return p.failTask(ctx, taskItem, &extra, fmt.Sprintf("创建相册失败: %v", err))
	}

	// 5. 保存结果
	albumIDs := make([]int64, len(albums))
	for i, album := range albums {
		albumIDs[i] = album.ID
	}
	extra.AlbumIDs = albumIDs
	extra.ClusterCount = len(albums)
	extra.NoiseCount = len(clusterResult.NoiseImageIDs)
	extra.TotalImages = len(clusterResult.NoiseImageIDs)
	for _, cluster := range clusterResult.Clusters {
		extra.TotalImages += len(cluster.ImageIDs)
	}

	// 标记完成
	p.updateProgress(ctx, taskItem, &extra, model.AITaskItemStatusCompleted, 100, "任务完成")

	logger.Info("智能相册任务完成",
		zap.Int64("task_id", taskItem.ID),
		zap.Int("album_count", len(albums)))

	return nil
}

// ClusteringResult Python 聚类服务返回的结果
type ClusteringResult struct {
	Clusters      []ClusterItem `json:"clusters"`
	NoiseImageIDs []int64       `json:"noise_image_ids"`
	NClusters     int           `json:"n_clusters"`
	ParamsUsed    interface{}   `json:"params_used"`
}

// ClusterItem 单个聚类结果
type ClusterItem struct {
	ClusterID      int     `json:"cluster_id"`
	ImageIDs       []int64 `json:"image_ids"`
	AvgProbability float64 `json:"avg_probability"`
}

// callClusteringService 调用 Python 聚类服务
func (p *SmartAlbumProcessor) callClusteringService(ctx context.Context, endpoint string, embeddings []*model.ImageEmbedding, params *model.HDBSCANParamsDTO) (*ClusteringResult, error) {
	// 构建请求
	embeddingVectors := make([][]float32, len(embeddings))
	imageIDs := make([]int64, len(embeddings))
	for i, e := range embeddings {
		embeddingVectors[i] = []float32(e.Embedding)
		imageIDs[i] = e.ImageID
	}

	reqBody := map[string]interface{}{
		"embeddings": embeddingVectors,
		"image_ids":  imageIDs,
		"hdbscan_params": map[string]interface{}{
			"min_cluster_size":          params.MinClusterSize,
			"min_samples":               params.MinSamples,
			"cluster_selection_epsilon": params.ClusterSelectionEpsilon,
			"cluster_selection_method":  params.ClusterSelectionMethod,
			"metric":                    params.Metric,
		},
		"umap_params": map[string]interface{}{
			"enabled":      params.UMAPEnabled,
			"n_components": params.UMAPComponents,
			"n_neighbors":  params.UMAPNeighbors,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// HTTP POST 到 Python 服务
	url := endpoint + "/v1/clustering"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求聚类服务失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("聚类服务返回错误: %d", resp.StatusCode)
	}

	var result ClusteringResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析聚类结果失败: %w", err)
	}

	return &result, nil
}

// createAlbumsFromClusterResult 从聚类结果创建相册
func (p *SmartAlbumProcessor) createAlbumsFromClusterResult(ctx context.Context, extra *model.AITaskItemExtra, clusterResult *ClusteringResult) ([]*model.Tag, error) {
	if clusterResult.NClusters == 0 {
		return []*model.Tag{}, nil
	}

	// 获取下一个智能相册编号
	nextNumber, err := p.getNextSmartAlbumNumber(ctx)
	if err != nil {
		return nil, err
	}

	albums := make([]*model.Tag, 0, len(clusterResult.Clusters))

	// 在事务中创建相册
	err = database.Transaction0(ctx, func(ctx context.Context) error {
		for i, cluster := range clusterResult.Clusters {
			albumName := fmt.Sprintf("智能相册 #%d", nextNumber+i)

			album := &model.Tag{
				Name: albumName,
				Type: model.TagTypeAlbum,
				Metadata: &model.AlbumMetadata{
					IsSmartAlbum: true,
					SmartAlbumConfig: &model.SmartAlbumConfig{
						ModelName:      extra.ModelName,
						Algorithm:      extra.Algorithm,
						ClusterID:      cluster.ClusterID,
						GeneratedAt:    time.Now(),
						HDBSCANParams:  p.convertHDBSCANParams(extra.HDBSCANParams),
						ImageCount:     len(cluster.ImageIDs),
						AvgProbability: cluster.AvgProbability,
					},
				},
			}

			if err := p.albumRepo.Create(ctx, album); err != nil {
				return fmt.Errorf("创建相册失败: %w", err)
			}

			// 添加图片到相册
			if err := p.albumRepo.AddImages(ctx, album.ID, cluster.ImageIDs); err != nil {
				return fmt.Errorf("添加图片到相册失败: %w", err)
			}

			albums = append(albums, album)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return albums, nil
}

// getNextSmartAlbumNumber 获取下一个智能相册编号
func (p *SmartAlbumProcessor) getNextSmartAlbumNumber(ctx context.Context) (int, error) {
	isSmartTrue := true
	albums, _, err := p.albumRepo.List(ctx, 1, 1000, &isSmartTrue)
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

// convertHDBSCANParams 转换 HDBSCAN 参数
func (p *SmartAlbumProcessor) convertHDBSCANParams(dto *model.HDBSCANParamsDTO) *model.HDBSCANParams {
	if dto == nil {
		return nil
	}
	return &model.HDBSCANParams{
		MinClusterSize:          dto.MinClusterSize,
		MinSamples:              dto.MinSamples,
		ClusterSelectionEpsilon: dto.ClusterSelectionEpsilon,
		ClusterSelectionMethod:  dto.ClusterSelectionMethod,
		Metric:                  dto.Metric,
		UMAPEnabled:             dto.UMAPEnabled,
		UMAPComponents:          dto.UMAPComponents,
		UMAPNeighbors:           dto.UMAPNeighbors,
	}
}

// updateProgress 更新任务进度并推送 WebSocket
func (p *SmartAlbumProcessor) updateProgress(ctx context.Context, taskItem *model.AITaskItem, extra *model.AITaskItemExtra, status string, progress int, message string) {
	taskItem.Status = status
	extra.Progress = progress
	extra.Message = message

	// 保存到数据库
	p.taskRepo.UpdateTaskExtra(ctx, taskItem.ID, extra)
	p.taskRepo.UpdateTaskItem(ctx, taskItem)

	// 推送 WebSocket
	if p.notifier != nil {
		p.notifier.NotifySmartAlbumProgress(&model.SmartAlbumProgressVO{
			TaskID:       taskItem.ID,
			Status:       status,
			Progress:     progress,
			Message:      message,
			AlbumIDs:     extra.AlbumIDs,
			ClusterCount: extra.ClusterCount,
			NoiseCount:   extra.NoiseCount,
			TotalImages:  extra.TotalImages,
		})
	}
}

// failTask 标记任务失败
func (p *SmartAlbumProcessor) failTask(ctx context.Context, taskItem *model.AITaskItem, extra *model.AITaskItemExtra, errMsg string) error {
	taskItem.Status = model.AITaskItemStatusFailed
	taskItem.Error = &errMsg
	extra.Message = errMsg

	p.taskRepo.UpdateTaskExtra(ctx, taskItem.ID, extra)
	p.taskRepo.UpdateTaskItem(ctx, taskItem)

	// 推送 WebSocket
	if p.notifier != nil {
		p.notifier.NotifySmartAlbumProgress(&model.SmartAlbumProgressVO{
			TaskID:   taskItem.ID,
			Status:   model.AITaskItemStatusFailed,
			Progress: extra.Progress,
			Message:  errMsg,
			Error:    &errMsg,
		})
	}

	logger.Error("智能相册任务失败",
		zap.Int64("task_id", taskItem.ID),
		zap.String("error", errMsg))

	return fmt.Errorf(errMsg)
}
