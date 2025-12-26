package llms

import (
	"context"
	"fmt"
	"gallary/server/internal/model"
	"net/http"
)

// ================== OpenAI 兼容模型客户端 ==================

// OpenAIClient OpenAI 兼容模型客户端
type OpenAIClient struct {
	config     *model.ModelConfig
	httpClient *http.Client
}

func (c *OpenAIClient) UpdateConfig(config *model.ModelConfig) {
	c.config = config
}

// NewOpenAIClient 创建 OpenAI 兼容模型客户端
func NewOpenAIClient(config *model.ModelConfig, httpClient *http.Client) *OpenAIClient {
	return &OpenAIClient{
		config:     config,
		httpClient: httpClient,
	}
}

func (c *OpenAIClient) SupportEmbedding() bool {
	return false
}

// SupportsTextEmbedding OpenAI 客户端当前不支持文本嵌入
func (c *OpenAIClient) SupportsTextEmbedding() bool {
	return false
}

// SupportsCombined OpenAI 客户端不支持同时计算
func (c *OpenAIClient) SupportsEmbeddingWithAesthetics() bool {
	return false
}

// Embedding 计算嵌入向量
func (c *OpenAIClient) Embedding(ctx context.Context, imageData []byte, text string) ([]float32, error) {
	return nil, fmt.Errorf("OpenAI 客户端不支持嵌入")
}

// EmbeddingWithAesthetics OpenAI 客户端不支持同时计算
func (c *OpenAIClient) EmbeddingWithAesthetics(ctx context.Context, imageData []byte) ([]float32, float64, error) {
	embedding, err := c.Embedding(ctx, imageData, "")
	return embedding, 0, err
}

// TestConnection 测试连接
func (c *OpenAIClient) TestConnection(ctx context.Context) error {
	url := c.config.Endpoint + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务返回错误状态: %d", resp.StatusCode)
	}

	return nil
}

// GetConfig 获取模型配置
func (c *OpenAIClient) GetConfig() *model.ModelConfig {
	return c.config
}
