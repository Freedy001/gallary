//go:build windows
// +build windows

package storage

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// getStats 获取 Windows 平台的存储统计信息
func (s *LocalStorage) getStats() (*StorageStats, error) {
	// 获取磁盘的可用空间和总容量
	var freeBytesAvailable uint64
	var totalBytes uint64
	var totalFreeBytes uint64

	// 将路径转换为 Windows API 需要的 UTF16 指针
	pathPtr, err := windows.UTF16PtrFromString(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("路径转换失败: %w", err)
	}

	// 调用 Windows API 获取磁盘空间信息
	err = windows.GetDiskFreeSpaceEx(
		pathPtr,
		&freeBytesAvailable,
		&totalBytes,
		&totalFreeBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("获取磁盘空间失败: %w", err)
	}

	used := totalBytes - freeBytesAvailable

	return &StorageStats{
		UsedBytes:  used,
		TotalBytes: totalBytes,
	}, nil
}

// getStatsUnix 在 Windows 上返回空统计
func (s *LocalStorage) getStatsUnix() (*StorageStats, error) {
	return &StorageStats{
		UsedBytes:  0,
		TotalBytes: 0,
	}, nil
}
