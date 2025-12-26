package llms

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gallary/server/internal/model"
	"gallary/server/internal/storage"
	"io"
	"net/http"
)

// ================== 自托管模型客户端 ==================

// promptOptimizerConfig 提示词优化器配置（用于 API 请求）
type promptOptimizerConfig struct {
	Enabled      bool   `json:"enabled"`
	SystemPrompt string `json:"system_prompt"`
}

// extraConfig 额外配置结构（用于解析 ExtraConfig 字段）
type extraConfig struct {
	PromptOptimizer *promptOptimizerConfig `json:"prompt_optimizer,omitempty"`
}

// SelfHostedClient 自托管模型客户端
type SelfHostedClient struct {
	config     *model.ModelConfig
	httpClient *http.Client
	manager    *storage.StorageManager
}

func (c *SelfHostedClient) UpdateConfig(config *model.ModelConfig) {
	c.config = config
}

// NewSelfHostedClient 创建自托管模型客户端
func NewSelfHostedClient(config *model.ModelConfig, httpClient *http.Client, manager *storage.StorageManager) *SelfHostedClient {
	return &SelfHostedClient{
		config:     config,
		httpClient: httpClient,
		manager:    manager,
	}
}

func (c *SelfHostedClient) SupportEmbedding() bool {
	return true
}

// SupportsTextEmbedding 自托管模型支持文本嵌入
func (c *SelfHostedClient) SupportsTextEmbedding() bool {
	return true
}

// SupportsEmbeddingWithAesthetics 自托管模型支持同时计算向量和美学评分
func (c *SelfHostedClient) SupportsEmbeddingWithAesthetics() bool {
	return true
}

// Embedding 使用阿里云兼容格式计算嵌入向量
func (c *SelfHostedClient) Embedding(ctx context.Context, imageData []byte, text string) ([]float32, error) {
	contents := make([]map[string]string, 0)

	if len(imageData) > 0 {
		base64Data := base64.StdEncoding.EncodeToString(imageData)
		contents = append(contents, map[string]string{"image": base64Data})
	}

	if text != "" {
		contents = append(contents, map[string]string{"text": text})
	}

	if len(contents) == 0 {
		return nil, fmt.Errorf("必须提供图片或文本")
	}

	// 文本查询时传递提示词优化器配置
	var promptOptimizer *promptOptimizerConfig
	if text != "" {
		promptOptimizer = c.getPromptOptimizerConfig()
	}

	return c.callMultimodalEmbedding(ctx, contents, promptOptimizer)
}

// EmbeddingWithAesthetics 同时计算嵌入和美学评分
// 使用传统的 aesthetics API 以获取美学评分
func (c *SelfHostedClient) EmbeddingWithAesthetics(ctx context.Context, imageData []byte) ([]float32, float64, error) {
	if len(imageData) == 0 {
		return nil, 0, fmt.Errorf("必须提供图片")
	}

	base64Data := base64.StdEncoding.EncodeToString(imageData)

	reqBody := struct {
		Input            string `json:"input"`
		ReturnEmbeddings bool   `json:"return_embeddings"`
	}{
		Input:            base64Data,
		ReturnEmbeddings: true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, 0, err
	}

	url := c.config.Endpoint + "/v1/aesthetics"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("请求自托管服务失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("自托管服务返回错误: %d, %s", resp.StatusCode, string(body))
	}

	var response struct {
		Data []struct {
			Index     int       `json:"index"`
			Score     float64   `json:"score"`
			Level     string    `json:"level"`
			Embedding []float32 `json:"embedding,omitempty"`
		} `json:"data"`
		Model string `json:"model"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, 0, fmt.Errorf("解析响应失败: %v", err)
	}

	if len(response.Data) == 0 {
		return nil, 0, fmt.Errorf("响应数据为空")
	}

	return response.Data[0].Embedding, response.Data[0].Score, nil
}

// getPromptOptimizerConfig 从 ExtraConfig 中解析提示词优化器配置
func (c *SelfHostedClient) getPromptOptimizerConfig() *promptOptimizerConfig {
	if c.config.ExtraConfig == "" {
		return nil
	}

	var extra extraConfig
	if err := json.Unmarshal([]byte(c.config.ExtraConfig), &extra); err != nil {
		return nil
	}

	return extra.PromptOptimizer
}

// callMultimodalEmbedding 调用阿里云兼容的多模态嵌入 API
func (c *SelfHostedClient) callMultimodalEmbedding(ctx context.Context, contents []map[string]string, promptOptimizer *promptOptimizerConfig) ([]float32, error) {
	// 构建阿里云兼容的请求格式
	reqBody := struct {
		Model string `json:"model"`
		Input struct {
			Contents []map[string]string `json:"contents"`
		} `json:"input"`
		PromptOptimizer *promptOptimizerConfig `json:"prompt_optimizer,omitempty"`
	}{
		Model:           c.config.ApiModelName,
		PromptOptimizer: promptOptimizer,
	}
	reqBody.Input.Contents = contents

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := c.config.Endpoint + "/v1/multimodal-embedding"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求自托管服务失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("自托管服务返回错误: %d, %s", resp.StatusCode, string(body))
	}

	// 解析阿里云兼容的响应格式
	var response struct {
		Output struct {
			Embeddings []struct {
				Index     int       `json:"index"`
				Embedding []float32 `json:"embedding"`
				Type      string    `json:"type"`
			} `json:"embeddings"`
		} `json:"output"`
		Usage struct {
			InputTokens int `json:"input_tokens"`
			ImageTokens int `json:"image_tokens"`
		} `json:"usage"`
		Model string `json:"model"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if len(response.Output.Embeddings) == 0 {
		return nil, fmt.Errorf("响应数据为空")
	}

	return response.Output.Embeddings[0].Embedding, nil
}

// TestConnection 测试连接
func (c *SelfHostedClient) TestConnection(ctx context.Context) error {
	url := c.config.Endpoint + "/health"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
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
func (c *SelfHostedClient) GetConfig() *model.ModelConfig {
	return c.config
}
