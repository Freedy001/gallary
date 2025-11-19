package storage

import (
	"context"
	"io"
)

// Storage 存储接口
type Storage interface {
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

	// GetURL 获取文件访问URL
	// path: 文件路径
	// 返回: 访问URL, 错误
	GetURL(ctx context.Context, path string) (string, error)

	// Exists 检查文件是否存在
	// path: 文件路径
	// 返回: 是否存在, 错误
	Exists(ctx context.Context, path string) (bool, error)
}
