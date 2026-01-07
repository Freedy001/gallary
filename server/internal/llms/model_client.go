package llms

import (
	"context"
	"gallary/server/internal/model"
	"gallary/server/internal/storage"
	"net/http"
)

// ChatMessage Chat 消息结构
type ChatMessage struct {
	Role    string `json:"role"` // "system", "user", "assistant"
	Content string `json:"content"`
}

// ModelClient 统一模型客户端接口
type ModelClient interface {
	// TestConnection 连接测试
	TestConnection(ctx context.Context, modelName string) error

	// GetConfig 获取模型配置
	GetConfig() *model.ModelConfig
	UpdateConfig(config *model.ModelConfig)
}

type EmbeddingClient interface {
	ModelClient

	// Embedding 嵌入向量计算
	// imageData: 图片二进制数据 (可为 nil)
	// text: 文本内容 (可为空)
	Embedding(ctx context.Context, imageData []byte, text string) ([]float32, error)
}

type LLMSClient interface {
	ModelClient

	// ChatCompletion 执行 Chat Completion 请求
	ChatCompletion(ctx context.Context, messages []ChatMessage) (string, error)
}

type SelfClient interface {
	EmbeddingClient

	// Aesthetics 美学评分
	// imageData: 图片二进制数据 (必须提供)
	Aesthetics(ctx context.Context, imageData []byte) (score float64, err error)

	// ClusterStream 流式聚类
	// 通过 progressChan 发送进度更新，完成后关闭 channel
	ClusterStream(ctx context.Context, req *ClusterStreamRequest, progressChan chan<- *ClusterProgress) error
}

// ================== 聚类相关类型 ==================

// ClusterStreamRequest 聚类流式请求
type ClusterStreamRequest struct {
	Embeddings    [][]float32
	ImageIDs      []int64
	TaskID        int64
	HDBSCANParams *HDBSCANParams
	UMAPParams    *UMAPParams
}

// HDBSCANParams HDBSCAN 算法参数
type HDBSCANParams struct {
	MinClusterSize          int
	MinSamples              *int
	ClusterSelectionEpsilon float32
	ClusterSelectionMethod  string
	Metric                  string
}

// UMAPParams UMAP 降维参数
type UMAPParams struct {
	Enabled     bool
	NComponents int
	NNeighbors  int
	MinDist     float32
}

// ClusterProgress 聚类进度
type ClusterProgress struct {
	TaskID   int64
	Status   string // pending, clustering, completed, failed
	Progress int    // 0-100
	Message  string
	Result   *ClusterResult
	Error    string
}

// ClusterResult 聚类结果
type ClusterResult struct {
	Clusters      []ClusterItem
	NoiseImageIDs []int64
	NClusters     int
	ParamsUsed    map[string]string
}

// ClusterItem 单个聚类项
type ClusterItem struct {
	ClusterID      int
	ImageIDs       []int64
	AvgProbability float32
}

// ================== 客户端工厂 ==================

// CreateModelClient 根据配置创建模型客户端
// provider: 提供商配置
// modelItem: 具体的模型项（包含 ApiModelName 和 ModelId）
func CreateModelClient(provider *model.ModelConfig, modelItem *model.ModelItem, httpClient *http.Client, manager *storage.StorageManager) ModelClient {
	switch provider.Provider {
	case model.SelfHosted:
		return newSelfHostedClient(provider, modelItem, manager)
	case model.OpenAI:
		return newOpenAIClient(provider, modelItem, httpClient)
	case model.AliyunMultimodalEmbedding:
		return newAliyunMultimodalEmbedding(provider, modelItem, httpClient, manager)
	default:
		return nil
	}
}
