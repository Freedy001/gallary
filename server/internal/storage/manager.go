package storage

import (
	"context"
	"errors"
	"fmt"
	"gallary/server/internal/model"
	"io"
	"sync"

	"gallary/server/pkg/logger"

	"go.uber.org/zap"
)

type storageTypeCtx string

var OverrideStorageType storageTypeCtx = "override_storage_type"

// StorageManager 存储管理器，支持动态切换存储实现
type StorageManager struct {
	mu        sync.RWMutex
	defaultId model.StorageId             // 当前存储类型名称
	storages  map[model.StorageId]Storage // 所有已初始化的存储实例
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

	logger.Info(string("使用" + cfg.DefaultId + "存储"))
	m.defaultId = cfg.DefaultId
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

// DefaultType 获取当前存储类型
func (m *StorageManager) DefaultType() model.StorageId {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.defaultId
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

		// 获取统计信息
		if storageStats, err := storage.GetStats(ctx); err == nil && storageStats != nil {
			providerStats.UsedBytes = storageStats.UsedBytes
			providerStats.TotalBytes = storageStats.TotalBytes
		}

		stats.Providers = append(stats.Providers, providerStats)
	}

	return stats
}

// ImageUrl 获取文件访问URL
func (m *StorageManager) ImageUrl(image *model.Image) (string, string) {
	thumbnail := "/static/" + image.ThumbnailPath
	if image.StorageId == model.StorageTypeLocal {
		return "/static/" + image.StoragePath, thumbnail
	}

	//fallback use hash to download
	return fmt.Sprintf("/resouse/%s/file", image.FileHash), thumbnail
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
