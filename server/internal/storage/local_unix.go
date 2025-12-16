//go:build !windows
// +build !windows

package storage

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// getStats 获取 Unix/Linux 平台的存储统计信息
func (s *basePath) getStats() (*StorageStats, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(string(*s), &stat); err != nil {
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
