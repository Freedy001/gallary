package model

import (
	"time"
)

// MigrationStatus 迁移状态
type MigrationStatus string

const (
	MigrationStatusPending    MigrationStatus = "pending"     // 等待开始
	MigrationStatusRunning    MigrationStatus = "running"     // 执行中
	MigrationStatusCompleted  MigrationStatus = "completed"   // 已完成
	MigrationStatusFailed     MigrationStatus = "failed"      // 失败
	MigrationStatusRolledBack MigrationStatus = "rolled_back" // 已回滚
	MigrationStatusCancelled  MigrationStatus = "cancelled"   // 已取消
)

// MigrationTask 迁移任务模型
type MigrationTask struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// 状态
	Status MigrationStatus `gorm:"type:varchar(20);not null;default:pending" json:"status"`

	OldStorageType StorageId `gorm:"type:varchar(64);not null" json:"old_storage_type"`

	// 旧配置
	OldBasePath string `gorm:"type:varchar(500);not null" json:"old_base_path"`
	// 新配置
	NewBasePath string `gorm:"type:varchar(500);not null" json:"new_base_path"`

	// 进度信息
	TotalFiles     int `gorm:"not null;default:0" json:"total_files"`
	ProcessedFiles int `gorm:"not null;default:0" json:"processed_files"`

	// 错误信息
	ErrorMessage *string `gorm:"type:text" json:"error_message,omitempty"`

	// 时间戳
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName 指定表名
func (MigrationTask) TableName() string {
	return "migration_tasks"
}

// IsActive 检查迁移任务是否处于活跃状态
func (m *MigrationTask) IsActive() bool {
	return m.Status == MigrationStatusPending || m.Status == MigrationStatusRunning
}

// GetProgress 获取迁移进度百分比
func (m *MigrationTask) GetProgress() float64 {
	if m.TotalFiles == 0 {
		return 0
	}
	return float64(m.ProcessedFiles) / float64(m.TotalFiles) * 100
}
