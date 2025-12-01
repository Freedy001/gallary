package storage

import (
	"context"
	"errors"
	"fmt"
	"gallary/server/internal/model"
	"io"
	"strings"
	"sync"

	"gallary/server/pkg/logger"
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
func NewStorageManager(cfg *model.StorageConfigDTO) (*StorageManager, error) {
	manager := &StorageManager{storages: make(map[model.StorageId]Storage)}

	// 初始化默认存储，使用 cfg 中已应用的配置
	if err := manager.initStorage(cfg); err != nil {
		return nil, fmt.Errorf("初始化存储失败: %w", err)
	}

	return manager, nil
}

// initStorage 根据类型和配置初始化存储
func (m *StorageManager) initStorage(cfg *model.StorageConfigDTO) error {
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
		storage, err := NewAliyunPanStorage(alConfig)
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

// SwitchStorage 切换存储类型
// 调用此方法前，应该先更新 cfg 中对应存储类型的配置
func (m *StorageManager) SwitchStorage(cfg *model.StorageConfigDTO) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return nil
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

// GetType 获取当前存储类型
func (m *StorageManager) GetType(ctx context.Context) model.StorageId {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.defaultId
}

// Upload 上传文件
func (m *StorageManager) Upload(ctx context.Context, file io.Reader, path string) (string, error) {
	storage := m.getStorage(ctx)
	if storage == nil {
		return "", fmt.Errorf("存储未初始化")
	}

	return storage.Upload(ctx, file, path)
}

// Download 下载文件
func (m *StorageManager) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	storage := m.getStorage(ctx)
	if storage == nil {
		return nil, fmt.Errorf("存储未初始化")
	}

	return storage.Download(ctx, path)
}

// Delete 删除文件
func (m *StorageManager) Delete(ctx context.Context, path string) error {
	storage := m.getStorage(ctx)
	if storage == nil {
		return fmt.Errorf("存储未初始化")
	}

	return storage.Delete(ctx, path)
}

// DeleteBatch 批量删除文件
func (m *StorageManager) DeleteBatch(ctx context.Context, paths []string) []DeleteResult {
	storage := m.getStorage(ctx)
	if storage == nil {
		results := make([]DeleteResult, len(paths))
		for i, p := range paths {
			results[i] = DeleteResult{Path: p, Error: fmt.Errorf("存储未初始化")}
		}
		return results
	}

	return storage.DeleteBatch(ctx, paths)
}

// GetURL 获取文件访问URL
func (m *StorageManager) GetURL(ctx context.Context, path string) (string, error) {
	storage := m.getStorage(ctx)
	if storage == nil {
		return "", fmt.Errorf("存储未初始化")
	}

	return storage.GetURL(ctx, path)
}

// GetURLBatch 批量获取文件访问URL
func (m *StorageManager) GetURLBatch(ctx context.Context, paths []string) []URLResult {
	storage := m.getStorage(ctx)
	if storage == nil {
		results := make([]URLResult, len(paths))
		for i, p := range paths {
			results[i] = URLResult{Path: p, Error: fmt.Errorf("存储未初始化")}
		}
		return results
	}

	return storage.GetURLBatch(ctx, paths)
}

// Exists 检查文件是否存在
func (m *StorageManager) Exists(ctx context.Context, path string) (bool, error) {
	storage := m.getStorage(ctx)
	if storage == nil {
		return false, fmt.Errorf("存储未初始化")
	}

	return storage.Exists(ctx, path)
}

// GetStats 获取存储统计信息
func (m *StorageManager) GetStats(ctx context.Context) (*StorageStats, error) {
	storage := m.getStorage(ctx)
	if storage == nil {
		return nil, fmt.Errorf("存储未初始化")
	}

	return storage.GetStats(ctx)
}

// Move 移动文件到新路径
func (m *StorageManager) Move(ctx context.Context, oldPath, newPath string) error {
	storage := m.getStorage(ctx)
	if storage == nil {
		return fmt.Errorf("存储未初始化")
	}

	return storage.Move(ctx, oldPath, newPath)
}

// MoveBatch 批量移动文件
func (m *StorageManager) MoveBatch(ctx context.Context, moves map[string]string) []MoveResult {
	storage := m.getStorage(ctx)
	if storage == nil {
		results := make([]MoveResult, 0, len(moves))
		for oldPath, newPath := range moves {
			results = append(results, MoveResult{
				OldPath: oldPath,
				NewPath: newPath,
				Error:   fmt.Errorf("存储未初始化"),
			})
		}
		return results
	}

	return storage.MoveBatch(ctx, moves)
}

// GetLocalStorage 获取本地存储实例（用于缩略图等始终存储在本地的场景）
func (m *StorageManager) GetLocalStorage() Storage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.storages[model.StorageTypeLocal]
}

// GetAliyunPanStorage 获取阿里云盘存储实例
func (m *StorageManager) GetAliyunPanStorage(accountId string) *AliyunPanStorage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if storage, ok := m.storages[model.AliyunpanStorageId(accountId)]; ok {
		if aliyunPan, ok := storage.(*AliyunPanStorage); ok {
			return aliyunPan
		}
	}
	return nil
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
			IsActive: storageType == m.defaultId,
		}

		if storageType == model.StorageTypeLocal {
			providerStats.Name = "本地存储"
		} else if strings.HasPrefix(string(storageType), "aliyunpan") {
			providerStats.Name = "阿里云盘"
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

// 确保 StorageManager 实现了 Storage 接口
var _ Storage = (*StorageManager)(nil)
