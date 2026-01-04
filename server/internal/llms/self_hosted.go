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

// PromptOptimizerConfig 提示词优化器配置（用于 API 请求）
type PromptOptimizerConfig struct {
	Enabled      bool   `json:"enabled"`
	SystemPrompt string `json:"system_prompt"`
}

// ExtraConfig 额外配置结构（用于解析 ExtraConfig 字段）
type ExtraConfig struct {
	PromptOptimizer *PromptOptimizerConfig `json:"prompt_optimizer,omitempty"`
}

// SelfHostedClient 自托管模型客户端
type SelfHostedClient struct {
	config     *model.ModelConfig
	modelItem  *model.ModelItem
	httpClient *http.Client
	manager    *storage.StorageManager
}

func (c *SelfHostedClient) UpdateConfig(config *model.ModelConfig) {
	c.config = config
}

// NewSelfHostedClient 创建自托管模型客户端
func NewSelfHostedClient(provider *model.ModelConfig, modelItem *model.ModelItem, httpClient *http.Client, manager *storage.StorageManager) *SelfHostedClient {
	return &SelfHostedClient{
		config:     provider,
		modelItem:  modelItem,
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

// SupportAesthetics 自托管模型支持美学评分
func (c *SelfHostedClient) SupportAesthetics() bool {
	return true
}

// SupportChatCompletion 自托管模型不支持 Chat Completion
func (c *SelfHostedClient) SupportChatCompletion() bool {
	return false
}

// ChatCompletion 自托管模型不支持 Chat Completion
func (c *SelfHostedClient) ChatCompletion(ctx context.Context, messages []ChatMessage) (string, error) {
	return "", fmt.Errorf("自托管模型不支持 Chat Completion")
}

// Embedding 使用阿里云兼容格式计算嵌入向量
func (c *SelfHostedClient) Embedding(ctx context.Context, imageData []byte, text string) ([]float32, error) {
	// 文本查询时传递提示词优化器配置
	var promptOptimizer *PromptOptimizerConfig
	if text != "" {
		promptOptimizer = c.getPromptOptimizerConfig()
	}

	return c.EmbeddingWithPromptConfig(ctx, imageData, text, promptOptimizer)
}

func (c *SelfHostedClient) EmbeddingWithPromptConfig(ctx context.Context, imageData []byte, text string, promptOptimizer *PromptOptimizerConfig) ([]float32, error) {
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

	return c.callMultimodalEmbedding(ctx, contents, promptOptimizer)
}

// Aesthetics 美学评分（仅返回评分，不返回嵌入向量）
func (c *SelfHostedClient) Aesthetics(ctx context.Context, imageData []byte) (float64, error) {
	if len(imageData) == 0 {
		return 0, fmt.Errorf("必须提供图片")
	}

	base64Data := base64.StdEncoding.EncodeToString(imageData)

	reqBody := struct {
		Input string `json:"input"`
	}{
		Input: base64Data,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return 0, err
	}

	url := c.config.Endpoint + "/v1/aesthetics"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("请求自托管服务失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("自托管服务返回错误: %d, %s", resp.StatusCode, string(body))
	}

	var response struct {
		Data []struct {
			Index int     `json:"index"`
			Score float64 `json:"score"`
			Level string  `json:"level"`
		} `json:"data"`
		Model string `json:"model"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, fmt.Errorf("解析响应失败: %v", err)
	}

	if len(response.Data) == 0 {
		return 0, fmt.Errorf("响应数据为空")
	}

	return response.Data[0].Score, nil
}

// getPromptOptimizerConfig 从 ExtraConfig 中解析提示词优化器配置
func (c *SelfHostedClient) getPromptOptimizerConfig() *PromptOptimizerConfig {
	if c.config.ExtraConfig == "" {
		return nil
	}

	var extra ExtraConfig
	if err := json.Unmarshal([]byte(c.config.ExtraConfig), &extra); err != nil {
		return nil
	}

	return extra.PromptOptimizer
}

// callMultimodalEmbedding 调用阿里云兼容的多模态嵌入 API
func (c *SelfHostedClient) callMultimodalEmbedding(ctx context.Context, contents []map[string]string, promptOptimizer *PromptOptimizerConfig) ([]float32, error) {
	// 构建阿里云兼容的请求格式
	reqBody := struct {
		Model string `json:"model"`
		Input struct {
			Contents []map[string]string `json:"contents"`
		} `json:"input"`
		PromptOptimizer *PromptOptimizerConfig `json:"prompt_optimizer,omitempty"`
	}{
		Model:           c.modelItem.ApiModelName,
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
