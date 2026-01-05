package repository

import (
	"context"
	"encoding/json"
	"time"

	"gallary/server/internal/model"
	"gallary/server/pkg/database"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AITaskRepository AI 任务/队列仓库接口
type AITaskRepository interface {
	// 队列管理
	FindOrCreateQueue(ctx context.Context, taskType model.TaskType, modelName string) (*model.AIQueue, error)
	FindQueueByID(ctx context.Context, id int64) (*model.AIQueue, error)
	UpdateQueue(ctx context.Context, queue *model.AIQueue) error
	GetAllQueues(ctx context.Context) ([]*model.AIQueue, error)
	DeleteQueueWithItems(ctx context.Context, queueID int64) error

	// 队列任务项管理（通用）
	AddItemsToQueue(ctx context.Context, queueID int64, queueKey string, itemIDs []int64, taskType model.TaskType) (int, error)
	RemoveTaskItem(ctx context.Context, taskItemID int64) error
	GetPendingTaskItems(ctx context.Context, queueID int64, limit int) ([]*model.AITaskItem, error)
	UpdateTaskItem(ctx context.Context, taskItem *model.AITaskItem) error

	// 统计
	GetQueueStats(ctx context.Context, queueID int64) (pending, processing, failed int, err error)
	GetQueueStatus(ctx context.Context) (*model.AIQueueStatus, error)

	// 队列详情
	GetFailedTaskItems(ctx context.Context, queueID int64, page, pageSize int) ([]*model.AITaskItem, int64, error)

	// 重试相关
	RetryTaskItem(ctx context.Context, taskItemID int64) error
	RetryQueueFailedItems(ctx context.Context, queueID int64) error

	// 查找有待处理项目的队列
	FindQueuesWithPendingItems(ctx context.Context, limit int) ([]*model.AIQueue, error)

	// 智能相册任务管理
	CreateSmartAlbumTask(ctx context.Context, extra *model.AITaskItemExtra) (*model.AITaskItem, error)
	UpdateTaskExtra(ctx context.Context, taskID int64, extra *model.AITaskItemExtra) error
	GetPendingSmartAlbumTasks(ctx context.Context, limit int) ([]*model.AITaskItem, error)
	GetSmartAlbumTaskByID(ctx context.Context, taskID int64) (*model.AITaskItem, error)
}

type aiTaskRepository struct{}

// NewAITaskRepository 创建 AI 任务仓库实例
func NewAITaskRepository() AITaskRepository {
	return &aiTaskRepository{}
}

// ================== 队列管理 ==================

// FindOrCreateQueue 查找或创建队列
func (r *aiTaskRepository) FindOrCreateQueue(ctx context.Context, taskType model.TaskType, modelName string) (*model.AIQueue, error) {
	queueKey := model.GenerateQueueKey(taskType, modelName)

	var queue model.AIQueue
	err := database.GetDB(ctx).WithContext(ctx).
		Where("queue_key = ?", queueKey).
		First(&queue).Error

	if err == nil {
		return &queue, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 创建新队列
	queue = model.AIQueue{
		QueueKey:  queueKey,
		TaskType:  taskType,
		ModelName: modelName,
		Status:    model.AIQueueStatusIdle,
	}

	err = database.GetDB(ctx).WithContext(ctx).Create(&queue).Error
	if err != nil {
		// 可能是并发创建，再查一次
		var existingQueue model.AIQueue
		if findErr := database.GetDB(ctx).WithContext(ctx).
			Where("queue_key = ?", queueKey).
			First(&existingQueue).Error; findErr == nil {
			return &existingQueue, nil
		}
		return nil, err
	}

	return &queue, nil
}

// FindQueueByID 根据 ID 查找队列
func (r *aiTaskRepository) FindQueueByID(ctx context.Context, id int64) (*model.AIQueue, error) {
	var queue model.AIQueue
	err := database.GetDB(ctx).WithContext(ctx).First(&queue, id).Error
	if err != nil {
		return nil, err
	}
	return &queue, nil
}

// UpdateQueue 更新队列
func (r *aiTaskRepository) UpdateQueue(ctx context.Context, queue *model.AIQueue) error {
	return database.GetDB(ctx).WithContext(ctx).Save(queue).Error
}

// GetAllQueues 获取所有队列
func (r *aiTaskRepository) GetAllQueues(ctx context.Context) ([]*model.AIQueue, error) {
	var queues []*model.AIQueue
	err := database.GetDB(ctx).WithContext(ctx).
		Order("created_at ASC").
		Find(&queues).Error
	return queues, err
}

// DeleteQueueWithItems 删除队列及其关联的任务项
func (r *aiTaskRepository) DeleteQueueWithItems(ctx context.Context, queueID int64) error {
	db := database.GetDB(ctx).WithContext(ctx)

	return db.Transaction(func(tx *gorm.DB) error {
		// 先删除所有关联的任务项
		if err := tx.Where("task_id = ?", queueID).Delete(&model.AITaskItem{}).Error; err != nil {
			return err
		}

		// 再删除队列
		if err := tx.Delete(&model.AIQueue{}, queueID).Error; err != nil {
			return err
		}

		return nil
	})
}

// DeleteQueuesByModelName 删除指定模型的所有队列及其关联的任务项

// ================== 队列任务项管理 ==================

// AddItemsToQueue 向队列添加任务项（去重）
// 返回实际添加的数量
func (r *aiTaskRepository) AddItemsToQueue(ctx context.Context, queueID int64, queueKey string, itemIDs []int64, taskType model.TaskType) (int, error) {
	if len(itemIDs) == 0 {
		return 0, nil
	}

	taskItems := make([]model.AITaskItem, len(itemIDs))
	for i, itemID := range itemIDs {
		taskItems[i] = model.AITaskItem{
			TaskID:   queueID,
			ItemID:   itemID,
			TaskType: taskType,
			QueueKey: queueKey,
			Status:   model.AITaskItemStatusPending,
		}
	}

	// 使用 ON CONFLICT DO NOTHING 来忽略重复
	result := database.GetDB(ctx).WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "item_id"}, {Name: "queue_key"}},
			DoNothing: true,
		}).
		CreateInBatches(taskItems, 100)

	if result.Error != nil {
		return 0, result.Error
	}

	return int(result.RowsAffected), nil
}

// RemoveTaskItem 删除任务项关联（处理成功后调用）
func (r *aiTaskRepository) RemoveTaskItem(ctx context.Context, taskItemID int64) error {
	return database.GetDB(ctx).WithContext(ctx).
		Delete(&model.AITaskItem{}, taskItemID).Error
}

// GetPendingTaskItems 获取队列中待处理的任务项
func (r *aiTaskRepository) GetPendingTaskItems(ctx context.Context, queueID int64, limit int) ([]*model.AITaskItem, error) {
	var taskItems []*model.AITaskItem
	err := database.GetDB(ctx).WithContext(ctx).
		Where("task_id = ? AND status = ?", queueID, model.AITaskItemStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&taskItems).Error
	return taskItems, err
}

// UpdateTaskItem 更新任务项状态
func (r *aiTaskRepository) UpdateTaskItem(ctx context.Context, taskItem *model.AITaskItem) error {
	return database.GetDB(ctx).WithContext(ctx).Save(taskItem).Error
}

// ================== 统计 ==================

// GetQueueStats 获取队列统计（动态计算）
func (r *aiTaskRepository) GetQueueStats(ctx context.Context, queueID int64) (pending, processing, failed int, err error) {
	db := database.GetDB(ctx).WithContext(ctx)

	var pendingCount, failedCount int64

	db.Model(&model.AITaskItem{}).
		Where("task_id = ? AND status = ?", queueID, model.AITaskItemStatusPending).
		Count(&pendingCount)

	db.Model(&model.AITaskItem{}).
		Where("task_id = ? AND status = ?", queueID, model.AITaskItemStatusFailed).
		Count(&failedCount)

	return int(pendingCount), 0, int(failedCount), nil
}

// GetQueueStatus 获取所有队列状态汇总
func (r *aiTaskRepository) GetQueueStatus(ctx context.Context) (*model.AIQueueStatus, error) {
	db := database.GetDB(ctx).WithContext(ctx)

	// 获取所有队列
	var queues []*model.AIQueue
	if err := db.Order("created_at ASC").Find(&queues).Error; err != nil {
		return nil, err
	}

	// 获取每个队列的统计
	queueInfos := make([]model.AIQueueInfo, 0, len(queues))
	var totalPending, totalFailed int

	for _, queue := range queues {
		pending, _, failed, err := r.GetQueueStats(ctx, queue.ID)
		if err != nil {
			return nil, err
		}

		// 只返回有任务项的队列
		if pending == 0 && failed == 0 {
			continue
		}

		queueInfos = append(queueInfos, model.AIQueueInfo{
			ID:           queue.ID,
			QueueKey:     queue.QueueKey,
			TaskType:     queue.TaskType,
			ModelName:    queue.ModelName,
			Status:       queue.Status,
			PendingCount: pending,
			FailedCount:  failed,
		})

		totalPending += pending
		totalFailed += failed
	}

	return &model.AIQueueStatus{
		Queues:       queueInfos,
		TotalPending: totalPending,
		TotalFailed:  totalFailed,
	}, nil
}

// ================== 队列详情 ==================

// GetFailedTaskItems 获取队列中的失败任务项列表（分页）
func (r *aiTaskRepository) GetFailedTaskItems(ctx context.Context, queueID int64, page, pageSize int) ([]*model.AITaskItem, int64, error) {
	db := database.GetDB(ctx).WithContext(ctx)

	// 统计总数
	var total int64
	if err := db.Model(&model.AITaskItem{}).
		Where("task_id = ? AND status = ?", queueID, model.AITaskItemStatusFailed).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	var taskItems []*model.AITaskItem
	offset := (page - 1) * pageSize
	if err := db.Where("task_id = ? AND status = ?", queueID, model.AITaskItemStatusFailed).
		Order("updated_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&taskItems).Error; err != nil {
		return nil, 0, err
	}

	return taskItems, total, nil
}

// ================== 重试相关 ==================

// RetryTaskItem 重试单个任务项（重置状态为 pending）
func (r *aiTaskRepository) RetryTaskItem(ctx context.Context, taskItemID int64) error {
	return database.GetDB(ctx).WithContext(ctx).
		Model(&model.AITaskItem{}).
		Where("id = ? AND status = ?", taskItemID, model.AITaskItemStatusFailed).
		Updates(map[string]interface{}{
			"status":     model.AITaskItemStatusPending,
			"error":      gorm.Expr("NULL"),
			"updated_at": time.Now(),
		}).Error
}

// RetryQueueFailedItems 重试队列中所有失败的任务项
func (r *aiTaskRepository) RetryQueueFailedItems(ctx context.Context, queueID int64) error {
	return database.GetDB(ctx).WithContext(ctx).
		Model(&model.AITaskItem{}).
		Where("task_id = ? AND status = ?", queueID, model.AITaskItemStatusFailed).
		Updates(map[string]interface{}{
			"status":     model.AITaskItemStatusPending,
			"error":      gorm.Expr("NULL"),
			"updated_at": time.Now(),
		}).Error
}

// ================== 查找队列 ==================

// FindQueuesWithPendingItems 查找有待处理任务项的队列
func (r *aiTaskRepository) FindQueuesWithPendingItems(ctx context.Context, limit int) ([]*model.AIQueue, error) {
	db := database.GetDB(ctx).WithContext(ctx)

	// 子查询：获取有待处理任务项的队列 ID
	subQuery := db.Model(&model.AITaskItem{}).
		Select("DISTINCT task_id").
		Where("status = ?", model.AITaskItemStatusPending)

	var queues []*model.AIQueue
	err := db.Where("id IN (?)", subQuery).
		Order("created_at ASC").
		Limit(limit).
		Find(&queues).Error

	return queues, err
}

// ================== 智能相册任务管理 ==================

// CreateSmartAlbumTask 创建智能相册任务
func (r *aiTaskRepository) CreateSmartAlbumTask(ctx context.Context, extra *model.AITaskItemExtra) (*model.AITaskItem, error) {
	// 序列化 Extra 数据
	extraJSON, err := json.Marshal(extra)
	if err != nil {
		return nil, err
	}

	// 创建任务项（ItemID 为 0 表示这是一个全局任务，不关联特定实体）
	taskItem := &model.AITaskItem{
		TaskID:   0, // 智能相册任务不需要队列，设为 0
		ItemID:   0, // 不关联特定图片
		TaskType: model.SmartAlbumTaskType,
		QueueKey: string(model.SmartAlbumTaskType),
		Status:   model.AITaskItemStatusPending,
		Extra:    extraJSON,
	}

	err = database.GetDB(ctx).WithContext(ctx).Create(taskItem).Error
	if err != nil {
		return nil, err
	}

	return taskItem, nil
}

// UpdateTaskExtra 更新任务的 Extra 信息
func (r *aiTaskRepository) UpdateTaskExtra(ctx context.Context, taskID int64, extra *model.AITaskItemExtra) error {
	extraJSON, err := json.Marshal(extra)
	if err != nil {
		return err
	}

	return database.GetDB(ctx).WithContext(ctx).
		Model(&model.AITaskItem{}).
		Where("id = ?", taskID).
		Update("extra", extraJSON).
		Update("updated_at", time.Now()).
		Error
}

// GetPendingSmartAlbumTasks 获取待处理的智能相册任务
func (r *aiTaskRepository) GetPendingSmartAlbumTasks(ctx context.Context, limit int) ([]*model.AITaskItem, error) {
	var taskItems []*model.AITaskItem
	err := database.GetDB(ctx).WithContext(ctx).
		Where("task_type = ? AND status = ?", model.SmartAlbumTaskType, model.AITaskItemStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&taskItems).Error
	return taskItems, err
}

// GetSmartAlbumTaskByID 根据 ID 获取智能相册任务
func (r *aiTaskRepository) GetSmartAlbumTaskByID(ctx context.Context, taskID int64) (*model.AITaskItem, error) {
	var taskItem model.AITaskItem
	err := database.GetDB(ctx).WithContext(ctx).
		Where("id = ? AND task_type = ?", taskID, model.SmartAlbumTaskType).
		First(&taskItem).Error
	if err != nil {
		return nil, err
	}
	return &taskItem, nil
}
