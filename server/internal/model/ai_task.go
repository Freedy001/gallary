package model

import (
	"fmt"
	"time"
)

// AI 任务类型常量
const (
	AITaskTypeEmbedding   = "embedding"   // 向量嵌入
	AITaskTypeDescription = "description" // LLM 描述
)

// AI 队列状态常量（队列永久存在，只有 idle 和 processing 两种状态）
const (
	AIQueueStatusIdle       = "idle"       // 空闲（无待处理图片）
	AIQueueStatusProcessing = "processing" // 处理中
)

// AI 任务图片状态常量
const (
	AITaskImageStatusPending    = "pending"    // 待处理
	AITaskImageStatusProcessing = "processing" // 处理中
	AITaskImageStatusFailed     = "failed"     // 失败
)

// AIQueue AI 队列（永久存在，一个模型名称一个队列）
// 注：表名为 ai_queue
type AIQueue struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	QueueKey  string    `gorm:"type:varchar(200);uniqueIndex" json:"queue_key"` // 队列唯一标识：embedding:{model_name} 或 description
	TaskType  string    `gorm:"type:varchar(20);not null" json:"task_type"`     // embedding, description
	ModelName string    `gorm:"type:varchar(100)" json:"model_name"`            // 模型名称（embedding 类型需要）
	Status    string    `gorm:"type:varchar(20);default:idle;index" json:"status"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	// 关联
	TaskImages []AITaskImage `gorm:"foreignKey:TaskID" json:"task_images,omitempty"`
}

// TableName 指定表名（保持向后兼容）
func (*AIQueue) TableName() string {
	return "ai_queue"
}

// GenerateQueueKey 生成队列标识
func GenerateQueueKey(taskType string, modelName string) string {
	if taskType == AITaskTypeEmbedding {
		return fmt.Sprintf("embedding:%s", modelName)
	}
	return taskType
}

// AITaskImage AI 任务关联的图片
// 成功处理后删除记录，失败则保留
type AITaskImage struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID    int64     `gorm:"not null;index" json:"task_id"`                         // 关联的队列 ID
	ImageID   int64     `gorm:"not null;uniqueIndex:idx_task_image" json:"image_id"`   // 图片 ID
	QueueKey  string    `gorm:"type:varchar(200);uniqueIndex:idx_task_image" json:"-"` // 队列标识（用于去重）
	Status    string    `gorm:"type:varchar(20);default:pending;index" json:"status"`  // pending, processing, failed
	Error     *string   `gorm:"type:text" json:"error,omitempty"`                      // 错误信息
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	// 关联
	Queue *AIQueue `gorm:"foreignKey:TaskID" json:"queue,omitempty"`
	Image *Image   `gorm:"foreignKey:ImageID" json:"image,omitempty"`
}

// TableName 指定表名
func (*AITaskImage) TableName() string {
	return "ai_queue_images"
}

// ================== 前端展示用的结构体 ==================

// AIQueueStatus AI 队列状态汇总（前端展示）
type AIQueueStatus struct {
	Queues          []AIQueueInfo `json:"queues"`           // 所有队列信息
	TotalPending    int           `json:"total_pending"`    // 总待处理数
	TotalProcessing int           `json:"total_processing"` // 总处理中数
	TotalFailed     int           `json:"total_failed"`     // 总失败数
}

// AIQueueInfo 单个队列信息（前端展示）
type AIQueueInfo struct {
	ID              int64  `json:"id"`
	QueueKey        string `json:"queue_key"`
	TaskType        string `json:"task_type"`
	ModelName       string `json:"model_name,omitempty"`
	Status          string `json:"status"` // idle, processing
	PendingCount    int    `json:"pending_count"`
	ProcessingCount int    `json:"processing_count"`
	FailedCount     int    `json:"failed_count"`
}

// AIQueueDetail 队列详情（含失败图片列表）
type AIQueueDetail struct {
	Queue        AIQueueInfo       `json:"queue"`
	FailedImages []AITaskImageInfo `json:"failed_images"`
	TotalFailed  int64             `json:"total_failed"`
	Page         int               `json:"page"`
	PageSize     int               `json:"page_size"`
}

// AITaskImageInfo 任务图片信息（前端展示）
type AITaskImageInfo struct {
	ID           int64   `json:"id"`
	ImageID      int64   `json:"image_id"`
	ImageName    string  `json:"imageName"`
	ThumbnailUrl string  `json:"thumbnailurl,omitempty"`
	Status       string  `json:"status"`
	Error        *string `json:"error,omitempty"`
	CreatedAt    string  `json:"created_at"`
}

// ToInfo 转换为前端展示信息
func (ti *AITaskImage) ToInfo(thumbnailUrl string) AITaskImageInfo {
	info := AITaskImageInfo{
		ID:        ti.ID,
		ImageID:   ti.ImageID,
		Status:    ti.Status,
		Error:     ti.Error,
		CreatedAt: ti.CreatedAt.Format(time.RFC3339),
	}
	if ti.Image != nil {
		info.ImageName = ti.Image.OriginalName
	}
	info.ThumbnailUrl = thumbnailUrl
	return info
}
