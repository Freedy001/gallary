package storage

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"gallary/server/config"
	"gallary/server/pkg/logger"

	"github.com/google/uuid"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan_web"
	"go.uber.org/zap"
)

// AliyunPanStorage 阿里云盘存储实现
type AliyunPanStorage struct {
	client   *aliyunpan_web.WebPanClient
	basePath string // 基础存储路径
	driveId  string // 网盘 ID
	mu       sync.RWMutex
	userInfo *aliyunpan.UserInfo

	// Token 管理
	refreshToken string
	webToken     aliyunpan_web.WebLoginToken
	appConfig    aliyunpan_web.AppConfig
}

// NewAliyunPanStorage 创建阿里云盘存储实例
func NewAliyunPanStorage(cfg *config.AliyunPanStorageConfig) (*AliyunPanStorage, error) {
	if cfg.RefreshToken == "" {
		return nil, fmt.Errorf("阿里云盘 refresh_token 不能为空")
	}

	// 1. 使用 RefreshToken 获取 AccessToken
	webToken, err := aliyunpan_web.GetAccessTokenFromRefreshToken(cfg.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("获取阿里云盘 AccessToken 失败: %w", err)
	}

	// 2. 生成设备 ID
	deviceId := generateDeviceId()

	// 3. 创建 AppConfig
	appConfig := aliyunpan_web.AppConfig{
		AppId:    "25dzX3vbYqktVxyX",
		DeviceId: deviceId,
		UserId:   "",
		Nonce:    0,
	}

	// 4. 创建 WebPanClient
	client := aliyunpan_web.NewWebPanClient(
		*webToken,
		aliyunpan_web.AppLoginToken{},
		appConfig,
		aliyunpan_web.SessionConfig{
			DeviceName: "GalleryServer",
			ModelName:  "Linux服务器",
		},
	)

	// 5. 获取用户信息
	userInfo, apiErr := client.GetUserInfo()
	if apiErr != nil {
		return nil, fmt.Errorf("获取阿里云盘用户信息失败: %s", apiErr.Error())
	}

	// 6. 更新 userId 并创建会话
	appConfig.UserId = userInfo.UserId
	client.UpdateAppConfig(appConfig)

	_, apiErr = client.CreateSession(nil)
	if apiErr != nil {
		return nil, fmt.Errorf("创建阿里云盘会话失败: %s", apiErr.Error())
	}

	// 7. 根据 DriveType 选择网盘 ID
	driveId := userInfo.FileDriveId // 默认使用备份盘
	switch cfg.DriveType {
	case "album":
		driveId = userInfo.AlbumDriveId
	case "resource":
		driveId = userInfo.ResourceDriveId
	case "file", "":
		driveId = userInfo.FileDriveId
	}

	if driveId == "" {
		return nil, fmt.Errorf("无法获取网盘 ID，请检查 drive_type 配置")
	}

	// 8. 确保基础目录存在
	basePath := cfg.BasePath
	if basePath == "" {
		basePath = "/gallery/images"
	}
	basePath = formatPath(basePath)

	_, apiErr = client.MkdirByFullPath(driveId, basePath)
	if apiErr != nil {
		// 目录可能已存在，忽略错误
	}

	storage := &AliyunPanStorage{
		client:       client,
		basePath:     basePath,
		driveId:      driveId,
		userInfo:     userInfo,
		refreshToken: cfg.RefreshToken,
		webToken:     *webToken,
		appConfig:    appConfig,
	}

	return storage, nil
}

// ensureValidToken 确保 Token 有效
func (s *AliyunPanStorage) ensureValidToken() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.webToken.IsAccessTokenExpired() {
		newToken, apiErr := aliyunpan_web.GetAccessTokenFromRefreshToken(s.refreshToken)
		if apiErr != nil {
			return fmt.Errorf("刷新 Token 失败: %s", apiErr.Error())
		}
		s.webToken = *newToken
		s.client.UpdateToken(*newToken)

		// 重新创建会话
		_, apiErr = s.client.CreateSession(nil)
		if apiErr != nil {
			return fmt.Errorf("重新创建会话失败: %s", apiErr.Error())
		}
	}
	return nil
}

func (s *AliyunPanStorage) GetType(ctx context.Context) config.StorageType {
	return config.StorageTypeAliyunpan
}

// Upload 上传文件到阿里云盘
func (s *AliyunPanStorage) Upload(ctx context.Context, file io.Reader, filePath string) (string, error) {
	if err := s.ensureValidToken(); err != nil {
		return "", err
	}

	// 构建完整路径
	fullPath := path.Join(s.basePath, filePath)
	dirPath := path.Dir(fullPath)
	fileName := path.Base(fullPath)

	// 确保目录存在
	dirResult, apiErr := s.client.MkdirByFullPath(s.driveId, dirPath)
	if apiErr != nil {
		return "", fmt.Errorf("创建目录失败: %s", apiErr.Error())
	}
	parentFileId := dirResult.FileId
	if parentFileId == "" {
		parentFileId = aliyunpan.DefaultRootParentFileId
	}

	// 读取文件内容到内存
	fileData, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("读取文件内容失败: %w", err)
	}

	fileSize := int64(len(fileData))

	// 计算 SHA1 哈希
	contentHash := ""
	if fileSize > 0 {
		hash := sha1.New()
		hash.Write(fileData)
		contentHash = strings.ToUpper(hex.EncodeToString(hash.Sum(nil)))
	} else {
		contentHash = aliyunpan.DefaultZeroSizeFileContentHash
	}

	// 尝试创建上传任务
	var createResult *aliyunpan.CreateFileUploadResult

	// 首先尝试正常上传
	createParam := &aliyunpan.CreateFileUploadParam{
		Name:            fileName,
		DriveId:         s.driveId,
		ParentFileId:    parentFileId,
		Size:            fileSize,
		ContentHash:     contentHash,
		ContentHashName: "sha1",
		CheckNameMode:   "auto_rename",
		BlockSize:       aliyunpan.DefaultChunkSize,
	}

	createResult, apiErr = s.client.CreateUploadFile(createParam)

	// 如果遇到 Rapid proof 错误，尝试使用不同的哈希或强制完整上传
	if apiErr != nil && strings.Contains(apiErr.Error(), "Rapid proof needed") {
		logger.Warn("遇到 Rapid proof 错误，尝试强制完整上传", zap.String("file", fileName))

		// 尝试使用空哈希强制完整上传
		createParam.ContentHash = ""
		createParam.ContentHashName = ""

		createResult, apiErr = s.client.CreateUploadFile(createParam)
		if apiErr != nil {
			return "", fmt.Errorf("重新创建上传任务失败: %s", apiErr.Error())
		}
	} else if apiErr != nil {
		return "", fmt.Errorf("创建上传任务失败: %s", apiErr.Error())
	}

	// 检查是否秒传成功
	if createResult.RapidUpload {
		return filePath, nil
	}

	// 分片上传
	if fileSize > 0 {
		chunkSize := aliyunpan.DefaultChunkSize
		offset := int64(0)

		for i, partInfo := range createResult.PartInfoList {
			// 计算当前分片大小
			currentChunkSize := chunkSize
			if offset+currentChunkSize > fileSize {
				currentChunkSize = fileSize - offset
			}

			// 获取分片数据
			chunkData := fileData[offset : offset+currentChunkSize]

			// 上传分片
			uploadData := &aliyunpan.FileUploadChunkData{
				Reader:    bytes.NewReader(chunkData),
				ChunkSize: currentChunkSize,
			}

			apiErr = s.client.UploadDataChunk(partInfo.UploadURL, uploadData)
			if apiErr != nil {
				return "", fmt.Errorf("上传分片 %d 失败: %s", i+1, apiErr.Error())
			}

			offset += currentChunkSize
		}
	}

	// 完成上传
	completeParam := &aliyunpan.CompleteUploadFileParam{
		DriveId:  s.driveId,
		FileId:   createResult.FileId,
		UploadId: createResult.UploadId,
	}

	_, apiErr = s.client.CompleteUploadFile(completeParam)
	if apiErr != nil {
		return "", fmt.Errorf("完成上传失败: %s", apiErr.Error())
	}

	return filePath, nil
}

// Download 从阿里云盘下载文件
func (s *AliyunPanStorage) Download(ctx context.Context, filePath string) (io.ReadCloser, error) {
	if err := s.ensureValidToken(); err != nil {
		return nil, err
	}

	fullPath := path.Join(s.basePath, filePath)

	// 获取文件信息
	fileInfo, apiErr := s.client.FileInfoByPath(s.driveId, fullPath)
	if apiErr != nil {
		return nil, fmt.Errorf("获取文件信息失败: %s", apiErr.Error())
	}

	if fileInfo.IsFolder() {
		return nil, fmt.Errorf("路径是文件夹，不是文件: %s", filePath)
	}

	s.client.DownloadFileData()
	// 获取下载 URL
	downloadResult, apiErr := s.client.GetFileDownloadUrl(&aliyunpan.GetFileDownloadUrlParam{
		DriveId: s.driveId,
		FileId:  fileInfo.FileId,
	})
	if apiErr != nil {
		return nil, fmt.Errorf("获取下载链接失败: %s", apiErr.Error())
	}

	if downloadResult.Url == "" {
		return nil, fmt.Errorf("下载链接为空")
	}

	// 检查是否为非法资源
	if strings.HasPrefix(downloadResult.Url, aliyunpan.IllegalDownloadUrlPrefix) {
		return nil, fmt.Errorf("资源已被屏蔽")
	}

	// 使用 HTTP 下载文件
	req, err := http.NewRequestWithContext(ctx, "GET", downloadResult.Url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建下载请求失败: %w", err)
	}
	req.Header.Set("Referer", "https://www.aliyundrive.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载文件失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		resp.Body.Close()
		return nil, fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// Delete 删除阿里云盘文件（移至回收站）
func (s *AliyunPanStorage) Delete(ctx context.Context, filePath string) error {
	if err := s.ensureValidToken(); err != nil {
		return err
	}

	fullPath := path.Join(s.basePath, filePath)

	// 获取文件信息
	fileInfo, apiErr := s.client.FileInfoByPath(s.driveId, fullPath)
	if apiErr != nil {
		// 如果文件不存在，视为删除成功
		if strings.Contains(apiErr.Error(), "文件不存在") {
			return nil
		}
		return fmt.Errorf("获取文件信息失败: %s", apiErr.Error())
	}

	// 删除文件（移至回收站）
	deleteParam := []*aliyunpan.FileBatchActionParam{
		{
			DriveId: s.driveId,
			FileId:  fileInfo.FileId,
		},
	}

	results, apiErr := s.client.FileDelete(deleteParam)
	if apiErr != nil {
		return fmt.Errorf("删除文件失败: %s", apiErr.Error())
	}

	if len(results) > 0 && !results[0].Success {
		return fmt.Errorf("删除文件失败")
	}

	return nil
}

// DeleteBatch 批量删除阿里云盘文件（并发10线程）
func (s *AliyunPanStorage) DeleteBatch(ctx context.Context, paths []string) []DeleteResult {
	if len(paths) == 0 {
		return []DeleteResult{}
	}

	const maxConcurrency = 10
	results := make([]DeleteResult, len(paths))
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for i, filePath := range paths {
		wg.Add(1)
		go func(index int, p string) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results[index] = DeleteResult{Path: p, Error: ctx.Err()}
				return
			}

			results[index] = DeleteResult{
				Path:  p,
				Error: s.Delete(ctx, p),
			}
		}(i, filePath)
	}

	wg.Wait()
	return results
}

// GetURL 获取文件访问 URL
func (s *AliyunPanStorage) GetURL(ctx context.Context, filePath string) (string, error) {
	if err := s.ensureValidToken(); err != nil {
		return "", err
	}

	fullPath := path.Join(s.basePath, filePath)

	// 获取文件信息
	fileInfo, apiErr := s.client.FileInfoByPath(s.driveId, fullPath)
	if apiErr != nil {
		return "", fmt.Errorf("获取文件信息失败: %s", apiErr.Error())
	}

	if fileInfo.IsFolder() {
		return "", fmt.Errorf("路径是文件夹，不是文件: %s", filePath)
	}

	// 获取下载 URL（有效期约 4 小时）
	downloadResult, apiErr := s.client.GetFileDownloadUrl(&aliyunpan.GetFileDownloadUrlParam{
		DriveId:   s.driveId,
		FileId:    fileInfo.FileId,
		ExpireSec: 14400, // 4 小时
	})
	if apiErr != nil {
		return "", fmt.Errorf("获取下载链接失败: %s", apiErr.Error())
	}

	return downloadResult.Url, nil
}

// GetURLBatch 批量获取文件访问 URL（并发10线程）
func (s *AliyunPanStorage) GetURLBatch(ctx context.Context, paths []string) []URLResult {
	if len(paths) == 0 {
		return []URLResult{}
	}

	const maxConcurrency = 10
	results := make([]URLResult, len(paths))
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for i, filePath := range paths {
		wg.Add(1)
		go func(index int, p string) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results[index] = URLResult{Path: p, Error: ctx.Err()}
				return
			}

			url, err := s.GetURL(ctx, p)
			results[index] = URLResult{
				Path:  p,
				URL:   url,
				Error: err,
			}
		}(i, filePath)
	}

	wg.Wait()
	return results
}

// Exists 检查文件是否存在
func (s *AliyunPanStorage) Exists(ctx context.Context, filePath string) (bool, error) {
	if err := s.ensureValidToken(); err != nil {
		return false, err
	}

	fullPath := path.Join(s.basePath, filePath)

	_, apiErr := s.client.FileInfoByPath(s.driveId, fullPath)
	if apiErr != nil {
		if strings.Contains(apiErr.Error(), "文件不存在") {
			return false, nil
		}
		return false, fmt.Errorf("检查文件失败: %s", apiErr.Error())
	}

	return true, nil
}

// GetStats 获取存储统计信息
func (s *AliyunPanStorage) GetStats(ctx context.Context) (*StorageStats, error) {
	if err := s.ensureValidToken(); err != nil {
		return nil, err
	}

	userInfo, apiErr := s.client.GetUserInfo()
	if apiErr != nil {
		return nil, fmt.Errorf("获取用户信息失败: %s", apiErr.Error())
	}

	return &StorageStats{
		UsedBytes:  userInfo.UsedSize,
		TotalBytes: userInfo.TotalSize,
	}, nil
}

// generateDeviceId 生成唯一设备 ID
func generateDeviceId() string {
	return uuid.New().String()
}

// formatPath 格式化路径
func formatPath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if p != "/" {
		p = strings.TrimSuffix(p, "/")
	}
	return p
}

func (s *AliyunPanStorage) GetClient() *aliyunpan_web.WebPanClient {
	return s.client
}
