package model

import (
	"fmt"
	"time"
)

// AI 任务类型常量
type TaskType string

const (
	ImageEmbeddingTaskType   TaskType = "image-embedding"   // 向量嵌入
	TagEmbeddingTaskType     TaskType = "tag-embedding"     // LLM 描述
	AestheticScoringTaskType TaskType = "aesthetic-scoring" // 美学评分
)

// AI 队列状态常量（队列永久存在，只有 idle 和 processing 两种状态）
const (
	AIQueueStatusIdle       = "idle"       // 空闲（无待处理图片）
	AIQueueStatusProcessing = "processing" // 处理中
)

// AI 任务项状态常量
const (
	AITaskItemStatusPending = "pending" // 待处理
	AITaskItemStatusFailed  = "failed"  // 失败
)

// AIQueue AI 队列（永久存在，一个模型名称一个队列）
// 注：表名为 ai_queue
type AIQueue struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	QueueKey  string    `gorm:"type:varchar(200);uniqueIndex" json:"queue_key"` // 队列唯一标识：embedding:{model_name} 或 description
	TaskType  TaskType  `gorm:"type:varchar(20);not null" json:"task_type"`     // embedding, description
	ModelName string    `gorm:"type:varchar(100)" json:"model_name"`            // 模型名称（embedding 类型需要）
	Status    string    `gorm:"type:varchar(20);default:idle;index" json:"status"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	// 关联
	TaskItems []AITaskItem `gorm:"foreignKey:TaskID" json:"task_items,omitempty"`
}

// TableName 指定表名（保持向后兼容）
func (*AIQueue) TableName() string {
	return "ai_queue"
}

// GenerateQueueKey 生成队列标识
func GenerateQueueKey(taskType TaskType, modelName string) string {
	return fmt.Sprintf("%s:%s", taskType, modelName)
}

// AITaskItem AI 任务关联的项目（通用）
// 成功处理后删除记录，失败则保留
// 支持任意实体类型：图片、标签等
type AITaskItem struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID    int64     `gorm:"not null;index" json:"task_id"`                        // 关联的队列 ID
	ItemID    int64     `gorm:"not null;uniqueIndex:idx_task_item" json:"item_id"`    // 实体 ID（图片ID、标签ID等）
	TaskType  TaskType  `gorm:"type:varchar(20);not null" json:"task_type"`           // 实体类型（image、tag等）
	QueueKey  string    `gorm:"type:varchar(200);uniqueIndex:idx_task_item" json:"-"` // 队列标识（用于去重）
	Status    string    `gorm:"type:varchar(20);default:pending;index" json:"status"` // pending, failed
	Error     *string   `gorm:"type:text" json:"error,omitempty"`                     // 错误信息
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	// 关联
	Queue *AIQueue `gorm:"foreignKey:TaskID" json:"queue,omitempty"`
}

// TableName 指定表名
func (*AITaskItem) TableName() string {
	return "ai_task_items"
}

// ================== 前端展示用的结构体 ==================

// AIQueueStatus AI 队列状态汇总（前端展示）
type AIQueueStatus struct {
	Queues       []AIQueueInfo `json:"queues"`        // 所有队列信息
	TotalPending int           `json:"total_pending"` // 总待处理数
	TotalFailed  int           `json:"total_failed"`  // 总失败数
}

// AIQueueInfo 单个队列信息（前端展示）
type AIQueueInfo struct {
	ID           int64    `json:"id"`
	QueueKey     string   `json:"queue_key"`
	TaskType     TaskType `json:"task_type"`
	ModelName    string   `json:"model_name,omitempty"`
	Status       string   `json:"status"` // idle, processing
	PendingCount int      `json:"pending_count"`
	FailedCount  int      `json:"failed_count"`
}

// AIQueueDetail 队列详情（含失败项目列表）
type AIQueueDetail struct {
	Queue       AIQueueInfo      `json:"queue"`
	FailedItems []AITaskItemInfo `json:"failed_items"`
	TotalFailed int64            `json:"total_failed"`
	Page        int              `json:"page"`
	PageSize    int              `json:"page_size"`
}

// AITaskItemInfo 任务项目信息（前端展示）
type AITaskItemInfo struct {
	ID        int64    `json:"id"`
	ItemID    int64    `json:"item_id"`
	ItemType  TaskType `json:"item_type"`
	ItemName  string   `json:"item_name,omitempty"`  // 项目名称（可选，由调用方填充）
	ItemThumb string   `json:"item_thumb,omitempty"` // 缩略图URL（可选，由调用方填充）
	Status    string   `json:"status"`
	Error     *string  `json:"error,omitempty"`
	CreatedAt string   `json:"created_at"`
}

// ToInfo 转换为前端展示信息
func (ti *AITaskItem) ToInfo() AITaskItemInfo {
	return AITaskItemInfo{
		ID:        ti.ID,
		ItemID:    ti.ItemID,
		ItemType:  ti.TaskType,
		Status:    ti.Status,
		Error:     ti.Error,
		CreatedAt: ti.CreatedAt.Format(time.RFC3339),
	}
}
