package llms

import (
	"context"
	"fmt"
	"gallary/server/pkg/logger"
	"io"

	pb "gallary/server/grpc"
	"gallary/server/internal/model"
	"gallary/server/internal/storage"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ================== 自托管模型客户端 ==================

// PromptOptimizerConfig 提示词优化器配置（用于 API 请求）
type PromptOptimizerConfig struct {
	Enabled      bool   `json:"enabled"`
	SystemPrompt string `json:"system_prompt"`
}

// ExtraConfig 额外配置结构（用于解析 ExtraConfig 字段）
type ExtraConfig struct {
	PromptOptimizer *PromptOptimizerConfig `json:"prompt_optimizer,omitempty"`
}

// selfHostedClient 自托管模型客户端（使用gRPC）
type selfHostedClient struct {
	config    *model.ModelConfig
	modelItem *model.ModelItem
	manager   *storage.StorageManager
	conn      *grpc.ClientConn
	client    pb.AIServiceClient
}

func (c *selfHostedClient) UpdateConfig(config *model.ModelConfig) {
	c.config = config
	conn, err := grpc.NewClient(config.Endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Error("初始化 grpc 失败", zap.Error(err))
		return
	}
	_ = c.Close()
	c.conn = conn
	c.client = pb.NewAIServiceClient(conn)
}

// newSelfHostedClient 创建自托管模型客户端
func newSelfHostedClient(provider *model.ModelConfig, modelItem *model.ModelItem, manager *storage.StorageManager) *selfHostedClient {
	client := &selfHostedClient{
		modelItem: modelItem,
		manager:   manager,
	}

	client.UpdateConfig(provider)
	return client
}

// Embedding 使用 gRPC 计算嵌入向量
func (c *selfHostedClient) Embedding(ctx context.Context, imageSource *model.ImageSource, text string) ([]float32, error) {
	if c.client == nil {
		return nil, fmt.Errorf("gRPC 客户端未初始化")
	}

	// 构建多模态嵌入请求
	req := &pb.MultimodalEmbeddingRequest{
		Model: c.modelItem.ApiModelName,
	}

	// 添加图片内容
	if imageSource != nil {
		if imageSource.URL != "" {
			// 优先使用 URL（避免二次传输）
			req.Contents = append(req.Contents, &pb.MultimodalContent{
				Content: &pb.MultimodalContent_ImageUrl{
					ImageUrl: imageSource.URL,
				},
			})
		} else if len(imageSource.Data) > 0 {
			// 使用二进制数据
			req.Contents = append(req.Contents, &pb.MultimodalContent{
				Content: &pb.MultimodalContent_Image{
					Image: imageSource.Data,
				},
			})
		}
	}

	// 添加文本内容
	if text != "" {
		req.Contents = append(req.Contents, &pb.MultimodalContent{
			Content: &pb.MultimodalContent_Text{
				Text: text,
			},
		})
	}

	if len(req.Contents) == 0 {
		return nil, fmt.Errorf("必须提供图片或文本")
	}

	// 调用 gRPC 接口
	resp, err := c.client.CreateMultimodalEmbedding(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("gRPC 嵌入请求失败: %v", err)
	}

	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("响应数据为空")
	}

	return resp.Embeddings[0].Embedding, nil
}

// Aesthetics 美学评分（使用 gRPC）
func (c *selfHostedClient) Aesthetics(ctx context.Context, imageSource *model.ImageSource) (float64, error) {
	if c.client == nil {
		return 0, fmt.Errorf("gRPC 客户端未初始化")
	}

	if imageSource == nil || (len(imageSource.Data) == 0 && imageSource.URL == "") {
		return 0, fmt.Errorf("必须提供图片")
	}

	req := &pb.AestheticRequest{
		ReturnDistribution: false,
	}

	// 优先使用新的 ImageInputs 字段
	if imageSource.URL != "" {
		// 使用 URL（避免二次传输）
		req.ImageInputs = []*pb.ImageInput{
			{
				Source: &pb.ImageInput_Url{
					Url: imageSource.URL,
				},
			},
		}
	} else {
		// 使用二进制数据
		req.ImageInputs = []*pb.ImageInput{
			{
				Source: &pb.ImageInput_Data{
					Data: imageSource.Data,
				},
			},
		}
	}

	resp, err := c.client.EvaluateAesthetic(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("gRPC 美学评分请求失败: %v", err)
	}

	if len(resp.Data) == 0 {
		return 0, fmt.Errorf("响应数据为空")
	}

	return float64(resp.Data[0].Score), nil
}

// TestConnection 测试 gRPC 连接
func (c *selfHostedClient) TestConnection(ctx context.Context, model_name string) error {
	if c.client == nil {
		return fmt.Errorf("gRPC 客户端未初始化")
	}

	req := &pb.HealthRequest{}
	_, err := c.client.Health(ctx, req)
	if err != nil {
		return fmt.Errorf("健康检查失败: %v", err)
	}

	return nil
}

// GetConfig 获取模型配置
func (c *selfHostedClient) GetConfig() *model.ModelConfig {
	return c.config
}

// Close 关闭 gRPC 连接
func (c *selfHostedClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ClusterStream 流式聚类
func (c *selfHostedClient) ClusterStream(ctx context.Context, req *ClusterStreamRequest, progressChan chan<- *ClusterProgress) error {
	defer close(progressChan)

	if c.client == nil {
		return fmt.Errorf("gRPC 客户端未初始化")
	}

	// 构建 gRPC 请求
	grpcReq := &pb.ClusteringRequest{
		Embeddings:    convertEmbeddingsToProto(req.Embeddings),
		ImageIds:      req.ImageIDs,
		TaskId:        req.TaskID,
		HdbscanParams: convertHDBSCANParamsToProto(req.HDBSCANParams),
		UmapParams:    convertUMAPParamsToProto(req.UMAPParams),
	}

	// 调用流式 RPC
	stream, err := c.client.ClusterStream(ctx, grpcReq)
	if err != nil {
		return fmt.Errorf("启动聚类流失败: %w", err)
	}

	// 接收流式响应
	for {
		update, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("聚类流错误: %w", err)
		}

		progress := convertProgressUpdateFromProto(update)
		progressChan <- progress

		if progress.Status == "completed" || progress.Status == "failed" {
			break
		}
	}

	return nil
}

// convertEmbeddingsToProto 将嵌入向量转换为 proto 格式
func convertEmbeddingsToProto(embeddings [][]float32) []*pb.Embedding {
	result := make([]*pb.Embedding, len(embeddings))
	for i, emb := range embeddings {
		result[i] = &pb.Embedding{Values: emb}
	}
	return result
}

// convertHDBSCANParamsToProto 将 HDBSCAN 参数转换为 proto 格式
func convertHDBSCANParamsToProto(params *HDBSCANParams) *pb.HDBSCANParams {
	if params == nil {
		return nil
	}
	pbParams := &pb.HDBSCANParams{
		MinClusterSize:          int32(params.MinClusterSize),
		ClusterSelectionEpsilon: params.ClusterSelectionEpsilon,
		ClusterSelectionMethod:  params.ClusterSelectionMethod,
	}
	if params.MinSamples != nil {
		minSamples := int32(*params.MinSamples)
		pbParams.MinSamples = &minSamples
	}
	return pbParams
}

// convertUMAPParamsToProto 将 UMAP 参数转换为 proto 格式
func convertUMAPParamsToProto(params *UMAPParams) *pb.UMAPParams {
	if params == nil {
		return nil
	}
	return &pb.UMAPParams{
		Enabled:     params.Enabled,
		NComponents: int32(params.NComponents),
		NNeighbors:  int32(params.NNeighbors),
		MinDist:     params.MinDist,
	}
}

// convertProgressUpdateFromProto 将 proto 进度更新转换为内部格式
func convertProgressUpdateFromProto(update *pb.ProgressUpdate) *ClusterProgress {
	progress := &ClusterProgress{
		TaskID:   update.TaskId,
		Status:   update.Status,
		Progress: int(update.Progress),
		Message:  update.Message,
		Error:    update.Error,
	}

	if update.Result != nil {
		progress.Result = convertClusteringResponseFromProto(update.Result)
	}

	return progress
}

// convertClusteringResponseFromProto 将 proto 聚类响应转换为内部格式
func convertClusteringResponseFromProto(resp *pb.ClusteringResponse) *ClusterResult {
	clusters := make([]ClusterItem, len(resp.Clusters))
	for i, c := range resp.Clusters {
		clusters[i] = ClusterItem{
			ClusterID:      int(c.ClusterId),
			ImageIDs:       c.ImageIds,
			AvgProbability: c.AvgProbability,
		}
	}

	return &ClusterResult{
		Clusters:      clusters,
		NoiseImageIDs: resp.NoiseImageIds,
		NClusters:     int(resp.NClusters),
		ParamsUsed:    resp.ParamsUsed,
	}
}
