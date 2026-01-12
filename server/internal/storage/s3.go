package storage

import (
	"context"
	"fmt"
	"gallary/server/internal/model"
	"gallary/server/pkg/logger"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"go.uber.org/zap"
)

// S3Storage S3 兼容存储实现
// 支持 AWS S3、MinIO、阿里云 OSS、腾讯云 COS、七牛等 S3 兼容服务
type S3Storage struct {
	client   *s3.Client
	bucket   string
	basePath string
	config   *model.S3StorageConfig
}

// NewS3Storage 创建 S3 存储实例
func NewS3Storage(cfg *model.S3StorageConfig) (*S3Storage, error) {
	if cfg == nil {
		return nil, fmt.Errorf("S3 配置不能为空")
	}

	if cfg.Bucket == "" {
		return nil, fmt.Errorf("S3 Bucket 不能为空")
	}

	if cfg.AccessKeyId == "" || cfg.SecretAccessKey == "" {
		return nil, fmt.Errorf("S3 访问凭证不能为空")
	}

	// 构建端点 URL
	endpoint := cfg.Endpoint
	if endpoint != "" {
		if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			scheme := "https"
			if !cfg.UseSSL {
				scheme = "http"
			}
			endpoint = fmt.Sprintf("%s://%s", scheme, endpoint)
		}
	}

	// 创建 AWS 配置
	awsCfg := aws.Config{
		Region: cfg.Region,
		Credentials: credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyId,
			cfg.SecretAccessKey,
			"",
		),
	}

	// 如果配置了代理，创建自定义 HTTP 客户端
	if cfg.ProxyURL != "" {
		proxyURL, err := url.Parse(cfg.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("代理地址格式错误: %w", err)
		}

		httpClient := awshttp.NewBuildableClient().WithTransportOptions(func(tr *http.Transport) {
			tr.Proxy = http.ProxyURL(proxyURL)
		})
		awsCfg.HTTPClient = httpClient

		logger.Info("S3 存储使用代理",
			zap.String("proxy", cfg.ProxyURL),
		)
	}

	// 创建 S3 客户端选项
	clientOptions := func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
		o.UsePathStyle = cfg.ForcePathStyle
	}

	client := s3.NewFromConfig(awsCfg, clientOptions)

	storage := &S3Storage{
		client:   client,
		bucket:   cfg.Bucket,
		basePath: strings.TrimPrefix(cfg.BasePath, "/"),
		config:   cfg,
	}

	logger.Info("S3 存储初始化成功",
		zap.String("name", cfg.Name),
		zap.String("bucket", cfg.Bucket),
		zap.String("endpoint", endpoint),
		zap.String("region", cfg.Region),
		zap.Bool("forcePathStyle", cfg.ForcePathStyle),
		zap.Bool("useSSL", cfg.UseSSL),
		zap.String("proxyURL", cfg.ProxyURL),
	)

	return storage, nil
}

// GetType 获取存储类型
func (s *S3Storage) GetType(ctx context.Context) model.StorageId {
	return s.config.Id
}

// fullPath 获取完整的对象路径
func (s *S3Storage) fullPath(relativePath string) string {
	if s.basePath == "" {
		return strings.TrimPrefix(relativePath, "/")
	}
	return path.Join(s.basePath, strings.TrimPrefix(relativePath, "/"))
}

// Upload 上传文件
func (s *S3Storage) Upload(ctx context.Context, file io.Reader, filePath string) (string, error) {
	key := s.fullPath(filePath)

	// 检测内容类型
	contentType := detectContentType(filePath)

	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return "", fmt.Errorf("上传文件到 S3 失败: %w", err)
	}

	return filePath, nil
}

// Download 下载文件
func (s *S3Storage) Download(ctx context.Context, filePath string) (io.ReadCloser, error) {
	key := s.fullPath(filePath)

	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	output, err := s.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("从 S3 下载文件失败: %w", err)
	}

	return output.Body, nil
}

// Delete 删除文件
func (s *S3Storage) Delete(ctx context.Context, filePath string) error {
	key := s.fullPath(filePath)

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	_, err := s.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("从 S3 删除文件失败: %w", err)
	}

	return nil
}

// DeleteBatch 批量删除文件
func (s *S3Storage) DeleteBatch(ctx context.Context, paths []string) []DeleteResult {
	results := make([]DeleteResult, len(paths))

	if len(paths) == 0 {
		return results
	}

	// 构建删除对象列表
	objects := make([]types.ObjectIdentifier, len(paths))
	for i, p := range paths {
		key := s.fullPath(p)
		objects[i] = types.ObjectIdentifier{
			Key: aws.String(key),
		}
	}

	input := &s3.DeleteObjectsInput{
		Bucket: aws.String(s.bucket),
		Delete: &types.Delete{
			Objects: objects,
			Quiet:   aws.Bool(false),
		},
	}

	output, err := s.client.DeleteObjects(ctx, input)

	// 填充成功结果
	for i, p := range paths {
		results[i] = DeleteResult{Path: p}
	}

	if err != nil {
		// 如果整体请求失败，所有文件都标记为失败
		for i := range results {
			results[i].Error = err
		}
		return results
	}

	// 处理部分失败的情况
	if output.Errors != nil {
		errorMap := make(map[string]error)
		for _, e := range output.Errors {
			if e.Key != nil && e.Message != nil {
				errorMap[*e.Key] = fmt.Errorf(*e.Message)
			}
		}
		for i, p := range paths {
			key := s.fullPath(p)
			if e, ok := errorMap[key]; ok {
				results[i].Error = e
			}
		}
	}

	return results
}

// GetURL 获取文件访问 URL
func (s *S3Storage) GetURL(ctx context.Context, filePath string) (string, error) {
	// 如果配置了自定义 URL 前缀（如 CDN），使用自定义前缀
	if s.config.UrlPrefix != "" {
		prefix := strings.TrimSuffix(s.config.UrlPrefix, "/")
		return fmt.Sprintf("%s/%s", prefix, s.fullPath(filePath)), nil
	}

	// 生成预签名 URL
	key := s.fullPath(filePath)

	presignClient := s3.NewPresignClient(s.client)

	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	presignedURL, err := presignClient.PresignGetObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = time.Hour * 24 // URL 有效期 24 小时
	})
	if err != nil {
		return "", fmt.Errorf("生成 S3 预签名 URL 失败: %w", err)
	}

	return presignedURL.URL, nil
}

// GetURLBatch 批量获取文件访问 URL
func (s *S3Storage) GetURLBatch(ctx context.Context, paths []string) []URLResult {
	results := make([]URLResult, len(paths))

	for i, p := range paths {
		url, err := s.GetURL(ctx, p)
		results[i] = URLResult{
			Path:  p,
			URL:   url,
			Error: err,
		}
	}

	return results
}

// Exists 检查文件是否存在
func (s *S3Storage) Exists(ctx context.Context, filePath string) (bool, error) {
	key := s.fullPath(filePath)

	input := &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	_, err := s.client.HeadObject(ctx, input)
	if err != nil {
		// 检查是否是 404 错误
		var responseError *awshttp.ResponseError
		if ok := isNotFoundError(err); ok {
			return false, nil
		}
		if responseError != nil && responseError.Response.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, fmt.Errorf("检查 S3 文件是否存在失败: %w", err)
	}

	return true, nil
}

// GetStats 获取存储统计信息
func (s *S3Storage) GetStats(ctx context.Context) (*StorageStats, error) {
	// S3 不直接提供存储统计，需要遍历对象计算
	// 这里返回一个基本的统计，实际使用中可能需要使用 CloudWatch 或其他方式获取
	var totalSize uint64

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(s.basePath),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("获取 S3 存储统计失败: %w", err)
		}

		for _, obj := range page.Contents {
			if obj.Size != nil {
				totalSize += uint64(*obj.Size)
			}
		}
	}

	return &StorageStats{
		UsedBytes:  totalSize,
		TotalBytes: 0,
	}, nil
}

// Move 移动文件到新路径
func (s *S3Storage) Move(ctx context.Context, oldPath, newPath string) error {
	oldKey := s.fullPath(oldPath)
	newKey := s.fullPath(newPath)

	// S3 没有直接的移动操作，需要复制后删除
	// 1. 复制对象
	copySource := fmt.Sprintf("%s/%s", s.bucket, oldKey)
	copyInput := &s3.CopyObjectInput{
		Bucket:     aws.String(s.bucket),
		CopySource: aws.String(copySource),
		Key:        aws.String(newKey),
	}

	_, err := s.client.CopyObject(ctx, copyInput)
	if err != nil {
		return fmt.Errorf("S3 复制文件失败: %w", err)
	}

	// 2. 删除原对象
	deleteInput := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(oldKey),
	}

	_, err = s.client.DeleteObject(ctx, deleteInput)
	if err != nil {
		return fmt.Errorf("S3 删除原文件失败: %w", err)
	}

	return nil
}

// GetConfig 获取 S3 配置
func (s *S3Storage) GetConfig() *model.S3StorageConfig {
	return s.config
}

// GetPresignedUploadURL 生成 S3 预签名上传 URL
func (s *S3Storage) GetPresignedUploadURL(ctx context.Context, filePath string, contentType string, expires time.Duration) (string, error) {
	key := s.fullPath(filePath)

	presignClient := s3.NewPresignClient(s.client)

	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}

	presignedURL, err := presignClient.PresignPutObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = expires
	})
	if err != nil {
		return "", fmt.Errorf("生成 S3 预签名上传 URL 失败: %w", err)
	}

	return presignedURL.URL, nil
}

// SupportsPresignedUpload 是否支持预签名上传
func (s *S3Storage) SupportsPresignedUpload() bool {
	return true
}

// TestConnection 测试 S3 连接
func (s *S3Storage) TestConnection(ctx context.Context) (bool, error) {
	// 使用 HeadBucket 来验证 bucket 存在且有访问权限
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return false, fmt.Errorf("无法访问 Bucket: %w", err)
	}
	return true, nil
}

// detectContentType 根据文件扩展名检测 MIME 类型
func detectContentType(filePath string) string {
	ext := strings.ToLower(path.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".heic", ".heif":
		return "image/heic"
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	case ".avi":
		return "video/x-msvideo"
	default:
		return "application/octet-stream"
	}
}

// isNotFoundError 检查是否是 404 错误
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// 检查错误消息是否包含 NotFound
	return strings.Contains(err.Error(), "NotFound") ||
		strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "NoSuchKey")
}
