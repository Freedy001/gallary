package storage

import (
	"context"
	"errors"
	"fmt"
	"gallary/server/internal/model"
	"io"
	"sync"
	"time"

	"gallary/server/pkg/logger"

	"go.uber.org/zap"
)

type storageTypeCtx string

var OverrideStorageType storageTypeCtx = "override_storage_type"

// StorageManager 存储管理器，支持动态切换存储实现
type StorageManager struct {
	mu                 sync.RWMutex
	defaultId          model.StorageId             // 原图默认存储类型
	thumbnailStorageId model.StorageId             // 缩略图默认存储类型
	storages           map[model.StorageId]Storage // 所有已初始化的存储实例
}

// NewStorageManager 创建存储管理器
// 注意：初始化时只传入 cfg，实际的存储配置应该在 ApplySettings 之后通过 SwitchStorage 切换
func NewStorageManager(cfg *model.StorageConfigPO) *StorageManager {
	manager := &StorageManager{storages: make(map[model.StorageId]Storage)}

	// 初始化默认存储，使用 cfg 中已应用的配置
	if err := manager.InitStorage(cfg); err != nil {
		logger.Error("存在失败的存储管理器", zap.Error(err))
	}

	return manager
}

// InitStorage 根据类型和配置初始化存储
func (m *StorageManager) InitStorage(cfg *model.StorageConfigPO) error {
	var storages = m.storages
	var storage Storage
	var err error

	storage, err = NewLocalStorage(cfg.LocalConfig)
	if err != nil {
		err = fmt.Errorf("初始化本地存储失败: %w", err)
	} else {
		storages[model.StorageTypeLocal] = storage
	}

	for _, alConfig := range cfg.AliyunpanConfig {
		storage, err := NewAliyunPanStorage(alConfig, cfg.AliyunpanGlobal)
		if err != nil {
			err = errors.Join(err, fmt.Errorf("初始化阿里云盘存储失败: %w", err))
		} else {
			storages[model.AliyunpanStorageId(storage.GetUserInfo().UserId)] = storage
		}
	}

	for _, s3Config := range cfg.S3Config {
		storage, initErr := NewS3Storage(s3Config)
		if initErr != nil {
			err = errors.Join(err, fmt.Errorf("初始化S3存储失败 [%s]: %w", s3Config.Name, initErr))
		} else {
			storages[s3Config.Id] = storage
		}
	}

	logger.Info(string("使用" + *cfg.DefaultId + "存储"))
	m.defaultId = *cfg.DefaultId

	// 设置缩略图存储，默认使用本地存储
	if cfg.ThumbnailStorageId != nil && *cfg.ThumbnailStorageId != "" {
		m.thumbnailStorageId = *cfg.ThumbnailStorageId
		logger.Info(string("缩略图使用" + *cfg.ThumbnailStorageId + "存储"))
	} else {
		m.thumbnailStorageId = model.StorageTypeLocal
		logger.Info("缩略图使用本地存储")
	}

	return err
}

func (m *StorageManager) getStorage(ctx context.Context) Storage {
	m.mu.RLock()
	st := ctx.Value(OverrideStorageType)
	if st == nil {
		st = m.defaultId
	}
	storage := m.storages[st.(model.StorageId)]
	m.mu.RUnlock()
	return storage
}

// 以下是 Storage 接口的代理实现

// DefaultStorageType 获取当前存储类型
func (m *StorageManager) DefaultStorageType() model.StorageId {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.defaultId
}

// ThumbnailStorageType 获取当前缩略图存储类型
func (m *StorageManager) ThumbnailStorageType() model.StorageId {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.thumbnailStorageId
}

// UploadToDefaultStorage 上传文件
func (m *StorageManager) UploadToDefaultStorage(ctx context.Context, file io.Reader, path string) (string, error) {
	storage := m.getStorage(ctx)
	if storage == nil {
		return "", fmt.Errorf("存储未初始化")
	}

	return storage.Upload(ctx, file, path)
}

// Download 下载文件
func (m *StorageManager) Download(ctx context.Context, storageId model.StorageId, path string) (io.ReadCloser, error) {
	storage := m.storages[storageId]
	if storage == nil {
		return nil, fmt.Errorf("%s存储未初始化", storageId)
	}
	return storage.Download(ctx, path)
}

// Delete 删除文件
func (m *StorageManager) Delete(ctx context.Context, storageId model.StorageId, path string) error {
	storage := m.storages[storageId]
	if storage == nil {
		return fmt.Errorf("%s存储未初始化", storageId)
	}
	return storage.Delete(ctx, path)
}

// DeleteBatch 批量删除文件
func (m *StorageManager) DeleteBatch(ctx context.Context, storageId model.StorageId, paths []string) ([]DeleteResult, error) {
	storage := m.storages[storageId]
	if storage == nil {
		return nil, fmt.Errorf("%s存储未初始化", storageId)
	}

	return storage.DeleteBatch(ctx, paths), nil
}

// Move 移动文件到新路径
func (m *StorageManager) Move(ctx context.Context, storageId model.StorageId, oldPath, newPath string) error {
	storage := m.storages[storageId]
	if storage == nil {
		return fmt.Errorf("%s存储未初始化", storageId)
	}
	return storage.Move(ctx, oldPath, newPath)
}

// GetLocalStorage 获取本地存储实例（用于缩略图等始终存储在本地的场景）
func (m *StorageManager) GetLocalStorage() Storage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.storages[model.StorageTypeLocal]
}

// GetThumbnailStorage 获取缩略图存储实例
func (m *StorageManager) GetThumbnailStorage() Storage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.storages[m.thumbnailStorageId]
}

// GetAliyunPanStorage 获取阿里云盘存储实例
func (m *StorageManager) GetAliyunPanStorage() []*AliyunPanStorage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var ret []*AliyunPanStorage

	for _, val := range m.storages {
		storage, ok := val.(*AliyunPanStorage)
		if ok {
			ret = append(ret, storage)
		}
	}
	return ret
}

// GetS3Storage 获取所有 S3 存储实例
func (m *StorageManager) GetS3Storage() []*S3Storage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var ret []*S3Storage

	for _, val := range m.storages {
		storage, ok := val.(*S3Storage)
		if ok {
			ret = append(ret, storage)
		}
	}
	return ret
}

// GetMultiStorageStats 获取所有存储提供者的统计信息
func (m *StorageManager) GetMultiStorageStats(ctx context.Context) *MultiStorageStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &MultiStorageStats{
		Providers: make([]ProviderStats, 0),
	}

	// 遍历所有已初始化的存储
	for storageType, storage := range m.storages {
		providerStats := ProviderStats{
			Id:       storageType,
			Name:     storageType.DriverName(),
			IsActive: storageType == m.defaultId,
		}

		storageStats, err := storage.GetStats(ctx)
		// 获取统计信息
		if err == nil && storageStats != nil {
			providerStats.UsedBytes = storageStats.UsedBytes
			providerStats.TotalBytes = storageStats.TotalBytes
		} else {
			providerStats.UsedBytes = 0
			providerStats.TotalBytes = 0
			logger.Error("获取存储发生异常", zap.String("type", string(storageType)), zap.Error(err))
		}

		stats.Providers = append(stats.Providers, providerStats)
	}

	return stats
}

// ImageUrl 获取文件访问URL
func (m *StorageManager) ImageUrl(image *model.Image) (string, string) {
	return m.Url(image.StorageId, image.StoragePath), m.Url(image.ThumbnailStorageId, image.ThumbnailPath)
}

func (m *StorageManager) Url(id model.StorageId, path string) string {
	storage := m.storages[id]
	if storage == nil {
		return "" ///没有存储信息
	}

	url, err := storage.GetURL(context.Background(), path)
	if err != nil {
		return fmt.Sprintf("/resouse/%s/%s", id, path)
	}

	return url
}

// ToVO 将 Image 转换为 ImageVO（包含URL）
func (m *StorageManager) ToVO(image *model.Image) *model.ImageVO {
	if image == nil {
		return nil
	}

	return image.ToVO(m.ImageUrl(image))
}

// ToVOList 批量将 Image 转换为 ImageVO（使用批量获取URL）
func (m *StorageManager) ToVOList(images []*model.Image) []*model.ImageVO {
	if len(images) == 0 {
		return []*model.ImageVO{}
	}

	var result []*model.ImageVO

	for _, image := range images {
		if image == nil {
			continue
		}

		result = append(result, m.ToVO(image))
	}

	return result
}

// UploadCredential 上传凭证
type UploadCredential struct {
	Type      string            `json:"type"`                 // "presigned" | "backend"
	URL       string            `json:"url"`                  // 上传 URL
	Method    string            `json:"method"`               // "PUT" | "POST"
	Headers   map[string]string `json:"headers,omitempty"`    // 需要携带的额外 headers
	ExpiresAt *time.Time        `json:"expires_at,omitempty"` // 过期时间
}

// GetUploadCredential 获取上传凭证
func (m *StorageManager) GetUploadCredential(ctx context.Context, storageId model.StorageId, path string, contentType string) (*UploadCredential, error) {
	storage := m.storages[storageId]
	if storage == nil {
		return nil, fmt.Errorf("存储未初始化")
	}

	if storage.SupportsPresignedUpload() {
		url, err := storage.GetPresignedUploadURL(ctx, path, contentType, 15*time.Minute)
		if err != nil {
			return nil, err
		}
		expiresAt := time.Now().Add(15 * time.Minute)
		return &UploadCredential{
			Type:      "presigned",
			URL:       url,
			Method:    "PUT",
			Headers:   map[string]string{"Content-Type": contentType},
			ExpiresAt: &expiresAt,
		}, nil
	}

	// 不支持预签名，返回后端代理 URL
	return &UploadCredential{
		Type:   "backend",
		Method: "PUT",
	}, nil
}

// Exists 检查文件是否存在
func (m *StorageManager) Exists(ctx context.Context, storageId model.StorageId, path string) (bool, error) {
	storage := m.storages[storageId]
	if storage == nil {
		return false, fmt.Errorf("%s存储未初始化", storageId)
	}
	return storage.Exists(ctx, path)
}

// GetImageSource 根据存储类型获取图片来源
// 对于本地存储，返回二进制数据；对于远程存储，返回 URL（避免二次传输）
func (m *StorageManager) GetImageSource(ctx context.Context, image *model.Image) (*model.ImageSource, error) {
	// 本地存储：直接读取二进制数据
	if image.StorageId == model.StorageTypeLocal {
		imageData, err := m.ReadImageData(ctx, image)
		if err != nil {
			return nil, err
		}
		return &model.ImageSource{Data: imageData}, nil
	}

	// 远程存储：获取 URL，让 AI 服务直接下载
	url := m.Url(image.StorageId, image.StoragePath)
	if url == "" {
		// 如果无法获取 URL，回退到二进制数据方式
		imageData, err := m.ReadImageData(ctx, image)
		if err != nil {
			return nil, err
		}
		return &model.ImageSource{Data: imageData}, nil
	}

	return &model.ImageSource{URL: url}, nil
}

// ReadImageData 读取图片二进制数据
func (m *StorageManager) ReadImageData(ctx context.Context, image *model.Image) ([]byte, error) {
	reader, err := m.Download(ctx, image.StorageId, image.StoragePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}
