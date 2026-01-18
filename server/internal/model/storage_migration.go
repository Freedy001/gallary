package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// StorageMigrationStatus 存储迁移状态
type StorageMigrationStatus string

const (
	StorageMigrationPending   StorageMigrationStatus = "pending"
	StorageMigrationRunning   StorageMigrationStatus = "running"
	StorageMigrationPaused    StorageMigrationStatus = "paused"
	StorageMigrationCompleted StorageMigrationStatus = "completed"
	StorageMigrationFailed    StorageMigrationStatus = "failed"
	StorageMigrationCancelled StorageMigrationStatus = "cancelled"
)

// MigrationType 迁移类型
type MigrationType string

const (
	MigrationTypeOriginal  MigrationType = "original"  // 原图
	MigrationTypeThumbnail MigrationType = "thumbnail" // 缩略图
)

// StorageMigrationTask 存储迁移任务模型
type StorageMigrationTask struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	// 迁移类型：原图或缩略图（一个任务只处理一种类型）
	MigrationType MigrationType          `gorm:"type:varchar(20);not null" json:"migration_type"`
	Status        StorageMigrationStatus `gorm:"type:varchar(20);not null;default:pending" json:"status"`

	// 统一的源/目标存储配置
	SourceStorageId StorageId `gorm:"type:varchar(64);not null" json:"source_storage_id"`
	TargetStorageId StorageId `gorm:"type:varchar(64);not null" json:"target_storage_id"`

	// 筛选条件 (JSON)
	FilterConditions *MigrationFilterConditions `gorm:"type:jsonb" json:"filter_conditions,omitempty"`

	// 总文件数（创建任务时统计，不变）
	TotalFiles int `gorm:"not null;default:0" json:"total_files"`
	// 配置选项
	DeleteSourceAfterMigration bool `gorm:"not null;default:true" json:"delete_source_after_migration"`
	// 错误信息
	ErrorMessage *string `gorm:"type:text" json:"error_message,omitempty"`

	// 时间戳
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	// 实时统计（非数据库字段，查询时填充）
	ProcessedFiles int `gorm:"-" json:"processed_files"`
	FailedFiles    int `gorm:"-" json:"failed_files"`
}

// TableName 指定表名
func (*StorageMigrationTask) TableName() string {
	return "storage_migration_tasks"
}

// IsActive 检查迁移任务是否处于活跃状态
func (m *StorageMigrationTask) IsActive() bool {
	return m.Status == StorageMigrationPending || m.Status == StorageMigrationRunning || m.Status == StorageMigrationPaused
}

// GetProgress 获取迁移进度百分比
func (m *StorageMigrationTask) GetProgress() float64 {
	if m.TotalFiles == 0 {
		return 0
	}
	return float64(m.ProcessedFiles) / float64(m.TotalFiles) * 100
}

// FlexibleDate 支持多种日期格式的自定义时间类型
type FlexibleDate struct {
	time.Time
}

// UnmarshalJSON 自定义 JSON 反序列化，支持多种日期格式
func (fd *FlexibleDate) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "null" || s == "" {
		return nil
	}

	// 尝试多种日期格式
	formats := []string{
		"2006-01-02",           // 日期格式
		time.RFC3339,           // 2006-01-02T15:04:05Z07:00
		"2006-01-02T15:04:05Z", // UTC 格式
		"2006-01-02T15:04:05",  // 无时区格式
		"2006-01-02 15:04:05",  // 空格分隔格式
	}

	var err error
	for _, format := range formats {
		fd.Time, err = time.Parse(format, s)
		if err == nil {
			return nil
		}
	}

	return errors.New("invalid date format: " + s)
}

// MarshalJSON 自定义 JSON 序列化
func (fd FlexibleDate) MarshalJSON() ([]byte, error) {
	if fd.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + fd.Time.Format(time.RFC3339) + `"`), nil
}

// MigrationFilterConditions 迁移筛选条件
type MigrationFilterConditions struct {
	AlbumIDs    []int64       `json:"album_ids,omitempty"`
	StartDate   *FlexibleDate `json:"start_date,omitempty"`
	EndDate     *FlexibleDate `json:"end_date,omitempty"`
	MinFileSize *int64        `json:"min_file_size,omitempty"`
	MaxFileSize *int64        `json:"max_file_size,omitempty"`
}

// Value 实现 driver.Valuer 接口，用于将结构体转换为数据库存储的值
func (m MigrationFilterConditions) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan 实现 sql.Scanner 接口，用于将数据库值转换为结构体
func (m *MigrationFilterConditions) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("failed to scan MigrationFilterConditions: invalid type")
	}

	return json.Unmarshal(bytes, m)
}

// MigrationFileRecord 迁移文件记录（用于断点续传）
type MigrationFileRecord struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID    int64     `gorm:"not null;index" json:"task_id"`
	ImageID   int64     `gorm:"not null;index" json:"image_id"`
	Image     *Image    `gorm:"foreignKey:ImageID" json:"image,omitempty"`               // 关联的图片对象
	Status    string    `gorm:"type:varchar(20);not null;default:pending" json:"status"` // pending, success, failed
	ErrorMsg  *string   `gorm:"type:text" json:"error_msg,omitempty"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName 指定表名
func (MigrationFileRecord) TableName() string {
	return "migration_file_records"
}

// MigrationFileRecordStatus 迁移文件记录状态
const (
	MigrationFileRecordPending    = "pending"
	MigrationFileRecordInProgress = "in_progress"
	MigrationFileRecordSuccess    = "success"
	MigrationFileRecordFailed     = "failed"
)

// MigrationFileRecordVO 迁移文件记录 VO（用于前端展示）
type MigrationFileRecordVO struct {
	ID        int64   `json:"id"`
	TaskID    int64   `json:"task_id"`
	ImageID   int64   `json:"image_id"`
	ImageName string  `json:"image_name"`
	ThumbURL  string  `json:"thumb_url"`
	Status    string  `json:"status"`
	ErrorMsg  *string `json:"error_msg,omitempty"`
	CreatedAt string  `json:"created_at"`
}

// MigrationProgressVO 迁移进度 VO（用于 WebSocket 推送）
type MigrationProgressVO struct {
	TaskID           int64   `json:"task_id"`
	Status           string  `json:"status"`
	MigrationType    string  `json:"migration_type"`
	SourceStorageId  string  `json:"source_storage_id"`
	TargetStorageId  string  `json:"target_storage_id"`
	TotalFiles       int     `json:"total_files"`
	ProcessedFiles   int     `json:"processed_files"`
	FailedFiles      int     `json:"failed_files"`
	ProgressPercent  float64 `json:"progress_percent"`
	Speed            int64   `json:"speed"`             // 传输速度（字节/秒）
	RemainingSeconds int     `json:"remaining_seconds"` // 预计剩余时间（秒）
}

// MigrationStatusVO 迁移状态 VO（包含所有活跃任务）
type MigrationStatusVO struct {
	Tasks        []MigrationProgressVO `json:"tasks"`
	TotalRunning int                   `json:"total_running"`
	TotalPaused  int                   `json:"total_paused"`
}

// ToProgressVO 转换为进度 VO
func (m *StorageMigrationTask) ToProgressVO() MigrationProgressVO {
	return MigrationProgressVO{
		TaskID:          m.ID,
		Status:          string(m.Status),
		MigrationType:   string(m.MigrationType),
		SourceStorageId: string(m.SourceStorageId),
		TargetStorageId: string(m.TargetStorageId),
		TotalFiles:      m.TotalFiles,
		ProcessedFiles:  m.ProcessedFiles,
		FailedFiles:     m.FailedFiles,
		ProgressPercent: m.GetProgress(),
	}
}
