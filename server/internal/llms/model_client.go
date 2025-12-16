package llms

import (
	"context"
	"gallary/server/internal/model"
	"gallary/server/internal/storage"
	"net/http"
)

// ModelClient 统一模型客户端接口
type ModelClient interface {
	//  SupportEmbedding 是否支持向量
	SupportEmbedding() bool
	// SupportsCombined 是否支持同时计算向量和评分
	SupportsEmbeddingWithAesthetics() bool

	// Embedding 嵌入向量计算
	Embedding(ctx context.Context, image *model.Image, text string) ([]float32, error)
	// EmbeddingWithAesthetics 同时计算嵌入和评分 (仅自托管模型支持)
	EmbeddingWithAesthetics(ctx context.Context, image *model.Image) (embedding []float32, score float64, err error)

	// TestConnection 连接测试
	TestConnection(ctx context.Context) error
	// GetConfig 获取模型配置
	GetConfig() *model.ModelConfig
}

// ================== 客户端工厂 ==================

// CreateModelClient 根据配置创建模型客户端
func CreateModelClient(config *model.ModelConfig, httpClient *http.Client, manager *storage.StorageManager) ModelClient {
	switch config.Provider {
	case model.SelfHosted:
		return NewSelfHostedClient(config, httpClient, manager)
	case model.OpenAI:
		return NewOpenAIClient(config, httpClient)
	case model.AliyunMultimodalEmbedding:
		return NewAliyunMultimodalEmbedding(config, httpClient, manager)
	default:
		return nil
	}
}
