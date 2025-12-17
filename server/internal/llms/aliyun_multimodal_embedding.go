package llms

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gallary/server/internal/model"
	"gallary/server/internal/storage"
	"gallary/server/pkg/logger"
	"image"
	"image/jpeg"
	_ "image/png" // 注册 PNG 解码器
	"io"
	"net/http"

	"github.com/nfnt/resize"
	"go.uber.org/zap"
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

// getImageBase64 从存储读取图片并转换为 Base64 Data URL
// 如果图片过大，会自动压缩以满足阿里云 API 限制（Base64 后 < 3MB）
func (c *AliyunMultimodalEmbedding) getImageBase64(img *model.Image) (string, error) {
	reader, err := c.manager.Download(context.Background(), img.StorageId, img.StoragePath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	// 阿里云 multimodal-embedding-v1 要求图片 ≤3MB
	// Base64 编码会增加约 33% 的大小，所以原始数据应该 < 2.25MB
	// 为安全起见，我们将阈值设为 2MB
	const maxOriginalSize = 2 * 1024 * 1024 // 2MB

	var imageData []byte
	var mimeType string

	if len(data) <= maxOriginalSize {
		imageData = data
		// 检测 MIME 类型
		mimeType = http.DetectContentType(data)
	} else {
		// 图片过大，需要压缩
		logger.Info("图片过大，进行压缩",
			zap.Int64("image_id", img.ID),
			zap.Int("original_size", len(data)))

		compressedData, err := c.compressImage(data, maxOriginalSize)
		if err != nil {
			return "", fmt.Errorf("压缩图片失败: %v", err)
		}

		logger.Info("图片压缩完成",
			zap.Int64("image_id", img.ID),
			zap.Int("compressed_size", len(compressedData)))

		imageData = compressedData
		mimeType = "image/jpeg" // 压缩后总是 JPEG
	}

	// 返回 Data URL 格式，阿里云 API 需要这种格式
	base64Data := base64.StdEncoding.EncodeToString(imageData)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data), nil
}

// compressImage 压缩图片到指定大小以下
func (c *AliyunMultimodalEmbedding) compressImage(data []byte, maxSize int) ([]byte, error) {
	// 解码图片
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("解码图片失败: %v", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 逐步缩小图片直到满足大小要求
	// 每次缩小到原来的 80%
	scaleFactor := 1.0
	var result []byte

	for attempts := 0; attempts < 10; attempts++ {
		newWidth := uint(float64(width) * scaleFactor)
		newHeight := uint(float64(height) * scaleFactor)

		// 确保最小尺寸
		if newWidth < 100 || newHeight < 100 {
			return nil, fmt.Errorf("图片缩放后尺寸过小")
		}

		// 缩放图片
		var resizedImg image.Image
		if scaleFactor < 1.0 {
			resizedImg = resize.Resize(newWidth, newHeight, img, resize.Lanczos3)
		} else {
			resizedImg = img
		}

		// 尝试不同的 JPEG 质量
		for quality := 85; quality >= 50; quality -= 10 {
			var buf bytes.Buffer
			err := jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: quality})
			if err != nil {
				continue
			}

			if buf.Len() <= maxSize {
				result = buf.Bytes()
				logger.Debug("压缩参数",
					zap.Float64("scale", scaleFactor),
					zap.Int("quality", quality),
					zap.Uint("width", newWidth),
					zap.Uint("height", newHeight))
				return result, nil
			}
		}

		// 继续缩小
		scaleFactor *= 0.8
	}

	if result != nil {
		return result, nil
	}

	return nil, fmt.Errorf("无法将图片压缩到目标大小")
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
