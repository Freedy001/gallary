package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"gallary/server/config"
)

// LocalStorage 本地存储实现
type LocalStorage struct {
	basePath  string
	urlPrefix string
}

// NewLocalStorage 创建本地存储实例
func NewLocalStorage(cfg *config.LocalStorageConfig) (*LocalStorage, error) {
	// 确保存储目录存在
	if err := os.MkdirAll(cfg.BasePath, 0755); err != nil {
		return nil, fmt.Errorf("创建存储目录失败: %w", err)
	}

	return &LocalStorage{
		basePath:  cfg.BasePath,
		urlPrefix: cfg.URLPrefix,
	}, nil
}

func (s *LocalStorage) GetType(ctx context.Context) config.StorageType {
	return config.StorageTypeLocal
}

// Upload 上传文件到本地
func (s *LocalStorage) Upload(ctx context.Context, file io.Reader, path string) (string, error) {
	fullPath := filepath.Join(s.basePath, path)

	// 确保目标目录存在
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 创建文件
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()

	// 复制数据
	if _, err := io.Copy(dst, file); err != nil {
		os.Remove(fullPath) // 清理失败的文件
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	return path, nil
}

// Download 从本地下载文件
func (s *LocalStorage) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, path)

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("文件不存在: %s", path)
		}
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}

	return file, nil
}

// Delete 删除本地文件
func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(s.basePath, path)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在，视为删除成功
		}
		return fmt.Errorf("删除文件失败: %w", err)
	}

	return nil
}

// DeleteBatch 批量删除本地文件
func (s *LocalStorage) DeleteBatch(ctx context.Context, paths []string) []DeleteResult {
	results := make([]DeleteResult, len(paths))
	for i, path := range paths {
		results[i] = DeleteResult{
			Path:  path,
			Error: s.Delete(ctx, path),
		}
	}
	return results
}

// GetURL 获取文件访问URL
func (s *LocalStorage) GetURL(ctx context.Context, path string) (string, error) {
	return fmt.Sprintf("%s/%s", s.urlPrefix, path), nil
}

// GetURLBatch 批量获取文件访问URL
func (s *LocalStorage) GetURLBatch(ctx context.Context, paths []string) []URLResult {
	results := make([]URLResult, len(paths))
	for i, path := range paths {
		url, err := s.GetURL(ctx, path)
		results[i] = URLResult{
			Path:  path,
			URL:   url,
			Error: err,
		}
	}
	return results
}

// Exists 检查文件是否存在
func (s *LocalStorage) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(s.basePath, path)

	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("检查文件失败: %w", err)
	}

	return true, nil
}

// GetStats 获取存储统计信息
func (s *LocalStorage) GetStats(ctx context.Context) (*StorageStats, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(s.basePath, &stat); err != nil {
		return nil, fmt.Errorf("获取存储统计失败: %w", err)
	}

	// 计算总容量和已用空间
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	used := total - free

	return &StorageStats{
		UsedBytes:  used,
		TotalBytes: total,
	}, nil
}
