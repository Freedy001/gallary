package storage

import (
	"context"
	"gallary/server/config"
	"io"
)

// StorageStats 存储统计信息
type StorageStats struct {
	UsedBytes  uint64 `json:"used_bytes"`  // 已使用空间（字节）
	TotalBytes uint64 `json:"total_bytes"` // 总容量（字节）
}

// URLResult 批量获取 URL 的结果
type URLResult struct {
	Path  string // 文件路径
	URL   string // 访问 URL
	Error error  // 错误信息
}

// DeleteResult 批量删除的结果
type DeleteResult struct {
	Path  string // 文件路径
	Error error  // 错误信息
}

// Storage 存储接口
type Storage interface {
	GetType(ctx context.Context) config.StorageType

	// Upload 上传文件
	// path: 存储路径
	// 返回: 存储后的完整路径, 错误
	Upload(ctx context.Context, file io.Reader, path string) (string, error)

	// Download 下载文件
	// path: 文件路径
	// 返回: 文件读取器, 错误
	Download(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete 删除文件
	// path: 文件路径
	Delete(ctx context.Context, path string) error

	// DeleteBatch 批量删除文件
	// paths: 文件路径列表
	// 返回: 每个文件的删除结果
	DeleteBatch(ctx context.Context, paths []string) []DeleteResult

	// GetURL 获取文件访问URL
	// path: 文件路径
	// 返回: 访问URL, 错误
	GetURL(ctx context.Context, path string) (string, error)

	// GetURLBatch 批量获取文件访问URL
	// paths: 文件路径列表
	// 返回: 每个文件的 URL 结果
	GetURLBatch(ctx context.Context, paths []string) []URLResult

	// Exists 检查文件是否存在
	// path: 文件路径
	// 返回: 是否存在, 错误
	Exists(ctx context.Context, path string) (bool, error)

	// GetStats 获取存储统计信息
	// 返回: 存储统计信息, 错误
	GetStats(ctx context.Context) (*StorageStats, error)
}
