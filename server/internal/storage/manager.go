package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"gallary/server/config"
	"gallary/server/pkg/logger"

	"go.uber.org/zap"
)

type storageTypeCtx string

var OverrideStorageType storageTypeCtx = "override_storage_type"

// StorageManager 存储管理器，支持动态切换存储实现
type StorageManager struct {
	mu          sync.RWMutex
	defaultType config.StorageType             // 当前存储类型名称
	storages    map[config.StorageType]Storage // 所有已初始化的存储实例
}

// NewStorageManager 创建存储管理器
// 注意：初始化时只传入 cfg，实际的存储配置应该在 ApplySettings 之后通过 SwitchStorage 切换
func NewStorageManager(cfg *config.StorageConfig) (*StorageManager, error) {
	manager := &StorageManager{storages: make(map[config.StorageType]Storage)}

	// 初始化默认存储，使用 cfg 中已应用的配置
	if err := manager.initStorage(cfg); err != nil {
		return nil, fmt.Errorf("初始化存储失败: %w", err)
	}

	return manager, nil
}

// initStorage 根据类型和配置初始化存储
func (m *StorageManager) initStorage(cfg *config.StorageConfig) error {
	var storages = m.storages
	var storage Storage
	var err error

	storage, err = NewLocalStorage(&cfg.Local)
	if err != nil {
		err = fmt.Errorf("初始化本地存储失败: %w", err)
	} else {
		storages[config.StorageTypeLocal] = storage
	}

	storage, err = NewAliyunPanStorage(&cfg.AliyunPan)
	if err != nil {
		err = errors.Join(err, fmt.Errorf("初始化阿里云盘存储失败: %w", err))
	} else {
		storages[config.StorageTypeAliyunpan] = storage
	}

	if err != nil {
		return err
	}

	switch cfg.Default {
	case config.StorageTypeLocal:
		logger.Info("使用本地存储", zap.String("path", cfg.Local.BasePath))
	case config.StorageTypeAliyunpan:
		logger.Info("使用阿里云盘存储", zap.String("base_path", cfg.AliyunPan.BasePath), zap.String("drive_type", cfg.AliyunPan.DriveType))
	default:
		return fmt.Errorf("不支持的存储类型: %s", cfg.Default)
	}

	m.defaultType = cfg.Default
	return nil
}

// SwitchStorage 切换存储类型
// 调用此方法前，应该先更新 cfg 中对应存储类型的配置
func (m *StorageManager) SwitchStorage(cfg *config.StorageConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果类型相同，也重新初始化（可能配置有变化）
	switch cfg.Default {
	case config.StorageTypeLocal:
		storage, err := NewLocalStorage(&cfg.Local)
		if err != nil {
			return err
		}
		m.storages[config.StorageTypeLocal] = storage
		logger.Info("使用本地存储", zap.String("path", cfg.Local.BasePath))
	case config.StorageTypeAliyunpan:
		storage, err := NewAliyunPanStorage(&cfg.AliyunPan)
		if err != nil {
			return err
		}
		m.storages[config.StorageTypeAliyunpan] = storage
		logger.Info("使用阿里云盘存储", zap.String("base_path", cfg.AliyunPan.BasePath), zap.String("drive_type", cfg.AliyunPan.DriveType))
	default:
		return fmt.Errorf("不支持的存储类型: %s", cfg.Default)
	}

	m.defaultType = cfg.Default
	logger.Info("存储类型已切换", zap.String("type", string(cfg.Default)))
	return nil
}

func (m *StorageManager) getStorage(ctx context.Context) Storage {
	m.mu.RLock()
	st := ctx.Value(OverrideStorageType)
	if st == nil {
		st = m.defaultType
	}
	storage := m.storages[st.(config.StorageType)]
	m.mu.RUnlock()
	return storage
}

// 以下是 Storage 接口的代理实现

// GetType 获取当前存储类型
func (m *StorageManager) GetType(ctx context.Context) config.StorageType {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.defaultType
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

// GetLocalStorage 获取本地存储实例（用于缩略图等始终存储在本地的场景）
func (m *StorageManager) GetLocalStorage() Storage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.storages[config.StorageTypeLocal]
}

// 确保 StorageManager 实现了 Storage 接口
var _ Storage = (*StorageManager)(nil)
