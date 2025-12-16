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

// ================== 阿里云多模态向量客户端 ==================

// 阿里云默认 API 端点
const Endpoint = "https://dashscope.aliyuncs.com/api/v1/services/embeddings/multimodal-embedding/multimodal-embedding"

// AliyunMultimodalEmbedding 阿里云多模态向量客户端
type AliyunMultimodalEmbedding struct {
	config     *model.ModelConfig
	httpClient *http.Client
	manager    *storage.StorageManager
}

// NewAliyunMultimodalEmbedding 创建阿里云客户端
func NewAliyunMultimodalEmbedding(config *model.ModelConfig, httpClient *http.Client, manager *storage.StorageManager) *AliyunMultimodalEmbedding {
	return &AliyunMultimodalEmbedding{
		config:     config,
		httpClient: httpClient,
		manager:    manager,
	}
}

func (c *AliyunMultimodalEmbedding) SupportEmbedding() bool {
	return true
}

// SupportsTextEmbedding 阿里云支持文本嵌入
func (c *AliyunMultimodalEmbedding) SupportsTextEmbedding() bool {
	return true
}

// SupportsEmbeddingWithAesthetics 阿里云不支持美学评分
func (c *AliyunMultimodalEmbedding) SupportsEmbeddingWithAesthetics() bool {
	return false
}

// Embedding 计算嵌入向量
func (c *AliyunMultimodalEmbedding) Embedding(ctx context.Context, image *model.Image, text string) ([]float32, error) {
	contents := make([]map[string]string, 0)

	if image != nil {
		imageData, err := c.getImageBase64(image)
		if err != nil {
			return nil, fmt.Errorf("获取图片数据失败: %v", err)
		}
		contents = append(contents, map[string]string{"image": imageData})
	}

	if text != "" {
		contents = append(contents, map[string]string{"text": text})
	}

	if len(contents) == 0 {
		return nil, fmt.Errorf("必须提供图片或文本")
	}

	return c.callMultimodalEmbedding(ctx, contents)
}

// EmbeddingWithAesthetics 阿里云不支持美学评分，仅返回嵌入向量
func (c *AliyunMultimodalEmbedding) EmbeddingWithAesthetics(ctx context.Context, image *model.Image) ([]float32, float64, error) {
	embedding, err := c.Embedding(ctx, image, "")
	return embedding, 0, err
}

// callMultimodalEmbedding 调用阿里云多模态嵌入 API
func (c *AliyunMultimodalEmbedding) callMultimodalEmbedding(ctx context.Context, contents []map[string]string) ([]float32, error) {
	// 构建请求体
	reqBody := struct {
		Model string `json:"model"`
		Input struct {
			Contents []map[string]string `json:"contents"`
		} `json:"input"`
	}{
		Model: c.config.ApiModelName,
	}
	reqBody.Input.Contents = contents

	// 如果未配置模型名称，使用默认模型
	if reqBody.Model == "" {
		reqBody.Model = "tongyi-embedding-vision-plus"
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	// 使用配置的端点或默认端点
	endpoint := c.config.Endpoint
	if endpoint == "" {
		endpoint = Endpoint
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求阿里云服务失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("阿里云服务返回错误: %d, %s", resp.StatusCode, string(body))
	}

	// 解析响应
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
		RequestId string `json:"request_id"`
		Code      string `json:"code"`
		Message   string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查 API 错误
	if response.Code != "" {
		return nil, fmt.Errorf("阿里云 API 错误: %s - %s", response.Code, response.Message)
	}

	if len(response.Output.Embeddings) == 0 {
		return nil, fmt.Errorf("响应数据为空")
	}

	return response.Output.Embeddings[0].Embedding, nil
}

// getImageBase64 从存储读取图片并转换为 Base64
func (c *AliyunMultimodalEmbedding) getImageBase64(image *model.Image) (string, error) {
	reader, err := c.manager.Download(context.Background(), image.StorageId, image.StoragePath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

// TestConnection 测试连接
func (c *AliyunMultimodalEmbedding) TestConnection(ctx context.Context) error {
	// 发送简单的文本嵌入请求来测试连接
	_, err := c.Embedding(ctx, nil, "测试连接")
	return err
}

// GetConfig 获取模型配置
func (c *AliyunMultimodalEmbedding) GetConfig() *model.ModelConfig {
	return c.config
}
