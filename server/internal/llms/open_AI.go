package llms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"gallary/server/internal/model"
	"io"
	"net/http"
	"strings"
)

// ================== OpenAI 兼容模型客户端 ==================

// ChatCompletionRequest OpenAI Chat Completion 请求结构
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

// ChatCompletionResponse OpenAI Chat Completion 响应结构
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// OpenAIClient OpenAI 兼容模型客户端
type OpenAIClient struct {
	config     *model.ModelConfig
	modelItem  *model.ModelItem
	httpClient *http.Client
}

func (c *OpenAIClient) UpdateConfig(config *model.ModelConfig) {
	c.config = config
}

// NewOpenAIClient 创建 OpenAI 兼容模型客户端
func NewOpenAIClient(provider *model.ModelConfig, modelItem *model.ModelItem, httpClient *http.Client) *OpenAIClient {
	return &OpenAIClient{
		config:     provider,
		modelItem:  modelItem,
		httpClient: httpClient,
	}
}

// TestConnection 测试连接
func (c *OpenAIClient) TestConnection(ctx context.Context, model_name string) error {
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

// ChatCompletion 执行 Chat Completion 请求
func (c *OpenAIClient) ChatCompletion(ctx context.Context, messages []ChatMessage) (string, error) {
	reqBody := ChatCompletionRequest{
		Model:       c.modelItem.ApiModelName,
		Messages:    messages,
		Temperature: 0, // 贪婪解码，保持输出稳定
		MaxTokens:   128,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := c.config.Endpoint + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求 OpenAI 服务失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI 服务返回错误: %d, %s", resp.StatusCode, string(body))
	}

	var response ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("响应数据为空")
	}

	return CleanPromptResponse(response.Choices[0].Message.Content), nil
}

// CleanPromptResponse 清理模型响应
func CleanPromptResponse(response string) string {
	// 移除思考标签
	if idx := strings.Index(response, "<think>"); idx != -1 {
		if endIdx := strings.Index(response, "</think>"); endIdx != -1 {
			response = response[endIdx+len("</think>"):]
		} else {
			response = response[:idx]
		}
	}

	response = strings.TrimSpace(response)
	response = strings.Trim(response, "\"'")

	// 移除常见前缀
	prefixes := []string{"Output:", "Optimized:", "Result:", "English:"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(response, prefix) {
			response = strings.TrimPrefix(response, prefix)
			response = strings.TrimSpace(response)
		}
	}

	// 只取第一行
	if idx := strings.Index(response, "\n"); idx != -1 {
		response = response[:idx]
	}

	return strings.TrimSpace(response)
}
