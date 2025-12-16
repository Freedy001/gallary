package storage

import (
	"context"
	"gallary/server/internal/model"
	"io"
)

// StorageStats 存储统计信息
type StorageStats struct {
	UsedBytes  uint64 `json:"used_bytes"`  // 已使用空间（字节）
	TotalBytes uint64 `json:"total_bytes"` // 总容量（字节）
}

// ProviderStats 单个存储提供者的统计信息
type ProviderStats struct {
	Id         model.StorageId `json:"id"`          // 存储类型: local, aliyunpan, oss...
	Name       string          `json:"name"`        // 显示名称
	UsedBytes  uint64          `json:"used_bytes"`  // 已使用空间（字节）
	TotalBytes uint64          `json:"total_bytes"` // 总容量（字节）
	IsActive   bool            `json:"is_active"`   // 是否为当前激活的存储
}

// MultiStorageStats 多存储提供者统计信息
type MultiStorageStats struct {
	Providers []ProviderStats `json:"providers"` // 各提供者统计
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

// MoveResult 移动文件的结果
type MoveResult struct {
	OldPath string // 原文件路径
	NewPath string // 新文件路径
	Error   error  // 错误信息
}

// Storage 存储接口
type Storage interface {
	GetType(ctx context.Context) model.StorageId

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

	// Move 移动文件到新路径（用于迁移）
	// oldPath: 原文件路径（相对路径）
	// newPath: 目标文件路径（相对路径）
	// 返回: 错误
	Move(ctx context.Context, oldPath, newPath string) error
}
