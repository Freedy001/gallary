package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// SmartAlbumService 智能相册服务接口
type SmartAlbumService interface {
	// GenerateSmartAlbums 生成智能相册（同步接口，保持向后兼容）
	GenerateSmartAlbums(ctx context.Context, req *GenerateSmartAlbumsRequest) (*GenerateSmartAlbumsResponse, error)

	// SubmitSmartAlbumTask 提交智能相册任务（异步接口）
	SubmitSmartAlbumTask(ctx context.Context, req *GenerateSmartAlbumsRequest) (*model.SmartAlbumProgressVO, error)
}

// GenerateSmartAlbumsRequest 生成智能相册请求
type GenerateSmartAlbumsRequest struct {
	ModelName     string                  `json:"model_name" binding:"required"`
	Algorithm     string                  `json:"algorithm" binding:"required"` // 目前仅支持 "hdbscan"
	HDBSCANParams *model.HDBSCANParamsDTO `json:"hdbscan_params"`
}

// GenerateSmartAlbumsResponse 生成智能相册响应
type GenerateSmartAlbumsResponse struct {
	Albums      []*model.AlbumVO `json:"albums"`       // 生成的相册列表
	NoiseCount  int              `json:"noise_count"`  // 噪声点数量
	TotalImages int              `json:"total_images"` // 总图片数
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

type smartAlbumService struct {
	albumRepo     repository.AlbumRepository
	embeddingRepo repository.ImageEmbeddingRepository
	taskRepo      repository.AITaskRepository
	loadBalancer  *llms.ModelLoadBalancer
	storage       *storage.StorageManager
	notifier      websocket.Notifier
	httpClient    *http.Client
}

// NewSmartAlbumService 创建智能相册服务实例
func NewSmartAlbumService(
	albumRepo repository.AlbumRepository,
	embeddingRepo repository.ImageEmbeddingRepository,
	taskRepo repository.AITaskRepository,
	loadBalancer *llms.ModelLoadBalancer,
	storage *storage.StorageManager,
	notifier websocket.Notifier,
) SmartAlbumService {
	return &smartAlbumService{
		albumRepo:     albumRepo,
		embeddingRepo: embeddingRepo,
		taskRepo:      taskRepo,
		loadBalancer:  loadBalancer,
		storage:       storage,
		notifier:      notifier,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
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

	// 创建任务
	extra := &model.AITaskItemExtra{
		ModelName:     req.ModelName,
		Algorithm:     req.Algorithm,
		HDBSCANParams: req.HDBSCANParams,
		Progress:      0,
		Message:       "任务已创建",
	}

	taskItem, err := s.taskRepo.CreateSmartAlbumTask(ctx, extra)
	if err != nil {
		return nil, fmt.Errorf("创建任务失败: %w", err)
	}

	logger.Info("智能相册任务已创建",
		zap.Int64("task_id", taskItem.ID),
		zap.String("model_name", req.ModelName))

	// 构建进度 VO
	progressVO := &model.SmartAlbumProgressVO{
		TaskID:   taskItem.ID,
		Status:   taskItem.Status,
		Progress: extra.Progress,
		Message:  extra.Message,
	}

	// 通过 WebSocket 推送任务创建通知
	if s.notifier != nil {
		s.notifier.NotifySmartAlbumProgress(progressVO)
	}

	return progressVO, nil
}

// GenerateSmartAlbums 生成智能相册（同步接口，保持向后兼容）
func (s *smartAlbumService) GenerateSmartAlbums(ctx context.Context, req *GenerateSmartAlbumsRequest) (*GenerateSmartAlbumsResponse, error) {
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

	// 1. 获取所有有向量的图片
	embeddings, err := s.embeddingRepo.GetAllEmbeddingsByModel(ctx, req.ModelName)
	if err != nil {
		return nil, fmt.Errorf("获取向量数据失败: %w", err)
	}

	if len(embeddings) < 2 {
		return nil, fmt.Errorf("向量数据不足，至少需要 2 张图片，当前仅有 %d 张", len(embeddings))
	}

	logger.Info("开始生成智能相册",
		zap.String("model_name", req.ModelName),
		zap.Int("image_count", len(embeddings)))

	// 2. 获取 Python 服务端点
	client, err := s.loadBalancer.GetClientByName(req.ModelName)
	if err != nil {
		return nil, fmt.Errorf("获取模型客户端失败: %w", err)
	}
	endpoint := client.GetConfig().Endpoint

	// 3. 调用 Python 服务进行聚类
	clusterResult, err := s.callClusteringService(ctx, endpoint, embeddings, req.HDBSCANParams)
	if err != nil {
		return nil, fmt.Errorf("聚类失败: %w", err)
	}

	logger.Info("聚类完成",
		zap.Int("n_clusters", clusterResult.NClusters),
		zap.Int("noise_count", len(clusterResult.NoiseImageIDs)))

	if clusterResult.NClusters == 0 {
		return &GenerateSmartAlbumsResponse{
			Albums:      []*model.AlbumVO{},
			NoiseCount:  len(clusterResult.NoiseImageIDs),
			TotalImages: len(embeddings),
		}, nil
	}

	// 4. 获取下一个智能相册编号
	nextNumber, err := s.getNextSmartAlbumNumber(ctx)
	if err != nil {
		return nil, err
	}

	// 5. 在事务中创建相册
	albums := make([]*model.AlbumVO, 0, len(clusterResult.Clusters))

	err = database.Transaction0(ctx, func(ctx context.Context) error {
		for i, cluster := range clusterResult.Clusters {
			albumName := fmt.Sprintf("智能相册 #%d", nextNumber+i)

			album := &model.Tag{
				Name: albumName,
				Type: model.TagTypeAlbum,
				Metadata: &model.AlbumMetadata{
					IsSmartAlbum: true,
					SmartAlbumConfig: &model.SmartAlbumConfig{
						ModelName:      req.ModelName,
						Algorithm:      req.Algorithm,
						ClusterID:      cluster.ClusterID,
						GeneratedAt:    time.Now(),
						HDBSCANParams:  s.convertHDBSCANParams(req.HDBSCANParams),
						ImageCount:     len(cluster.ImageIDs),
						AvgProbability: cluster.AvgProbability,
					},
				},
			}

			if err := s.albumRepo.Create(ctx, album); err != nil {
				return fmt.Errorf("创建相册失败: %w", err)
			}

			// 添加图片到相册
			if err := s.albumRepo.AddImages(ctx, album.ID, cluster.ImageIDs); err != nil {
				return fmt.Errorf("添加图片到相册失败: %w", err)
			}

			// 转换为 VO
			albumVO := album.ToAlbumVO(nil, int64(len(cluster.ImageIDs)))
			albums = append(albums, albumVO)

			logger.Debug("创建智能相册",
				zap.String("name", albumName),
				zap.Int("image_count", len(cluster.ImageIDs)))
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	logger.Info("智能相册生成完成",
		zap.Int("album_count", len(albums)),
		zap.Int("noise_count", len(clusterResult.NoiseImageIDs)))

	return &GenerateSmartAlbumsResponse{
		Albums:      albums,
		NoiseCount:  len(clusterResult.NoiseImageIDs),
		TotalImages: len(embeddings),
	}, nil
}

// callClusteringService 调用 Python 聚类服务
func (s *smartAlbumService) callClusteringService(ctx context.Context, endpoint string, embeddings []*model.ImageEmbedding, params *model.HDBSCANParamsDTO) (*ClusteringResult, error) {
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

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求聚类服务失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("聚类服务返回错误: %d, %s", resp.StatusCode, string(body))
	}

	var result ClusteringResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析聚类结果失败: %w", err)
	}

	return &result, nil
}

// getNextSmartAlbumNumber 获取下一个智能相册编号
func (s *smartAlbumService) getNextSmartAlbumNumber(ctx context.Context) (int, error) {
	// 查询现有智能相册的最大编号（只查智能相册）
	isSmartTrue := true
	albums, _, err := s.albumRepo.List(ctx, 1, 1000, &isSmartTrue)
	if err != nil {
		return 1, nil // 出错时从 1 开始
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
func (s *smartAlbumService) convertHDBSCANParams(dto *model.HDBSCANParamsDTO) *model.HDBSCANParams {
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
