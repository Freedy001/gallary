package llms

import (
	"context"
	"gallary/server/internal/model"
	"gallary/server/internal/storage"
	"net/http"
)

// ModelClient 统一模型客户端接口
type ModelClient interface {
	// SupportEmbedding 是否支持向量嵌入
	SupportEmbedding() bool
	// SupportAesthetics 是否支持美学评分
	SupportAesthetics() bool

	// Embedding 嵌入向量计算
	// imageData: 图片二进制数据 (可为 nil)
	// text: 文本内容 (可为空)
	Embedding(ctx context.Context, imageData []byte, text string) ([]float32, error)
	// Aesthetics 美学评分
	// imageData: 图片二进制数据 (必须提供)
	Aesthetics(ctx context.Context, imageData []byte) (score float64, err error)

	// TestConnection 连接测试
	TestConnection(ctx context.Context) error
	// GetConfig 获取模型配置
	GetConfig() *model.ModelConfig
	UpdateConfig(config *model.ModelConfig)
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
