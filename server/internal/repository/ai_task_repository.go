package repository

import (
	"context"
	"time"

	"gallary/server/internal/model"
	"gallary/server/pkg/database"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AITaskRepository AI 任务/队列仓库接口
type AITaskRepository interface {
	// 队列管理
	FindOrCreateQueue(ctx context.Context, taskType string, modelName string) (*model.AIQueue, error)
	FindQueueByID(ctx context.Context, id int64) (*model.AIQueue, error)
	UpdateQueue(ctx context.Context, queue *model.AIQueue) error
	GetAllQueues(ctx context.Context) ([]*model.AIQueue, error)

	// 队列图片管理
	AddImagesToQueue(ctx context.Context, queueID int64, queueKey string, imageIDs []int64) (int, error)
	RemoveTaskImage(ctx context.Context, taskImageID int64) error
	GetPendingTaskImages(ctx context.Context, queueID int64, limit int) ([]*model.AITaskImage, error)
	UpdateTaskImage(ctx context.Context, taskImage *model.AITaskImage) error

	// 统计
	GetQueueStats(ctx context.Context, queueID int64) (pending, processing, failed int, err error)
	GetQueueStatus(ctx context.Context) (*model.AIQueueStatus, error)

	// 队列详情
	GetFailedTaskImages(ctx context.Context, queueID int64, page, pageSize int) ([]*model.AITaskImage, int64, error)

	// 重试相关
	RetryTaskImage(ctx context.Context, taskImageID int64) error
	RetryQueueFailedImages(ctx context.Context, queueID int64) error

	// 查找有待处理图片的队列
	FindQueuesWithPendingImages(ctx context.Context, limit int) ([]*model.AIQueue, error)
}

type aiTaskRepository struct{}

// NewAITaskRepository 创建 AI 任务仓库实例
func NewAITaskRepository() AITaskRepository {
	return &aiTaskRepository{}
}

// ================== 队列管理 ==================

// FindOrCreateQueue 查找或创建队列
func (r *aiTaskRepository) FindOrCreateQueue(ctx context.Context, taskType string, modelName string) (*model.AIQueue, error) {
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

// ================== 队列图片管理 ==================

// AddImagesToQueue 向队列添加图片（去重）
// 返回实际添加的数量
func (r *aiTaskRepository) AddImagesToQueue(ctx context.Context, queueID int64, queueKey string, imageIDs []int64) (int, error) {
	if len(imageIDs) == 0 {
		return 0, nil
	}

	taskImages := make([]model.AITaskImage, len(imageIDs))
	for i, imageID := range imageIDs {
		taskImages[i] = model.AITaskImage{
			TaskID:   queueID,
			ImageID:  imageID,
			QueueKey: queueKey,
			Status:   model.AITaskImageStatusPending,
		}
	}

	// 使用 ON CONFLICT DO NOTHING 来忽略重复
	result := database.GetDB(ctx).WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "image_id"}, {Name: "queue_key"}},
			DoNothing: true,
		}).
		CreateInBatches(taskImages, 100)

	if result.Error != nil {
		return 0, result.Error
	}

	return int(result.RowsAffected), nil
}

// RemoveTaskImage 删除任务图片关联（处理成功后调用）
func (r *aiTaskRepository) RemoveTaskImage(ctx context.Context, taskImageID int64) error {
	return database.GetDB(ctx).WithContext(ctx).
		Delete(&model.AITaskImage{}, taskImageID).Error
}

// GetPendingTaskImages 获取队列中待处理的图片
func (r *aiTaskRepository) GetPendingTaskImages(ctx context.Context, queueID int64, limit int) ([]*model.AITaskImage, error) {
	var taskImages []*model.AITaskImage
	err := database.GetDB(ctx).WithContext(ctx).
		Where("task_id = ? AND status = ?", queueID, model.AITaskImageStatusPending).
		Preload("Image").
		Order("created_at ASC").
		Limit(limit).
		Find(&taskImages).Error
	return taskImages, err
}

// UpdateTaskImage 更新任务图片状态
func (r *aiTaskRepository) UpdateTaskImage(ctx context.Context, taskImage *model.AITaskImage) error {
	return database.GetDB(ctx).WithContext(ctx).Save(taskImage).Error
}

// ================== 统计 ==================

// GetQueueStats 获取队列统计（动态计算）
func (r *aiTaskRepository) GetQueueStats(ctx context.Context, queueID int64) (pending, processing, failed int, err error) {
	db := database.GetDB(ctx).WithContext(ctx)

	var pendingCount, processingCount, failedCount int64

	db.Model(&model.AITaskImage{}).
		Where("task_id = ? AND status = ?", queueID, model.AITaskImageStatusPending).
		Count(&pendingCount)

	db.Model(&model.AITaskImage{}).
		Where("task_id = ? AND status = ?", queueID, model.AITaskImageStatusProcessing).
		Count(&processingCount)

	db.Model(&model.AITaskImage{}).
		Where("task_id = ? AND status = ?", queueID, model.AITaskImageStatusFailed).
		Count(&failedCount)

	return int(pendingCount), int(processingCount), int(failedCount), nil
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
	var totalPending, totalProcessing, totalFailed int

	for _, queue := range queues {
		pending, processing, failed, err := r.GetQueueStats(ctx, queue.ID)
		if err != nil {
			return nil, err
		}

		// 只返回有图片的队列
		if pending == 0 && processing == 0 && failed == 0 {
			continue
		}

		queueInfos = append(queueInfos, model.AIQueueInfo{
			ID:              queue.ID,
			QueueKey:        queue.QueueKey,
			TaskType:        queue.TaskType,
			ModelName:       queue.ModelName,
			Status:          queue.Status,
			PendingCount:    pending,
			ProcessingCount: processing,
			FailedCount:     failed,
		})

		totalPending += pending
		totalProcessing += processing
		totalFailed += failed
	}

	return &model.AIQueueStatus{
		Queues:          queueInfos,
		TotalPending:    totalPending,
		TotalProcessing: totalProcessing,
		TotalFailed:     totalFailed,
	}, nil
}

// ================== 队列详情 ==================

// GetFailedTaskImages 获取队列中的失败图片列表（分页）
func (r *aiTaskRepository) GetFailedTaskImages(ctx context.Context, queueID int64, page, pageSize int) ([]*model.AITaskImage, int64, error) {
	db := database.GetDB(ctx).WithContext(ctx)

	// 统计总数
	var total int64
	if err := db.Model(&model.AITaskImage{}).
		Where("task_id = ? AND status = ?", queueID, model.AITaskImageStatusFailed).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	var taskImages []*model.AITaskImage
	offset := (page - 1) * pageSize
	if err := db.Where("task_id = ? AND status = ?", queueID, model.AITaskImageStatusFailed).
		Preload("Image").
		Order("updated_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&taskImages).Error; err != nil {
		return nil, 0, err
	}

	return taskImages, total, nil
}

// ================== 重试相关 ==================

// RetryTaskImage 重试单张图片（重置状态为 pending）
func (r *aiTaskRepository) RetryTaskImage(ctx context.Context, taskImageID int64) error {
	return database.GetDB(ctx).WithContext(ctx).
		Model(&model.AITaskImage{}).
		Where("id = ? AND status = ?", taskImageID, model.AITaskImageStatusFailed).
		Updates(map[string]interface{}{
			"status":     model.AITaskImageStatusPending,
			"error":      gorm.Expr("NULL"),
			"updated_at": time.Now(),
		}).Error
}

// RetryQueueFailedImages 重试队列中所有失败的图片
func (r *aiTaskRepository) RetryQueueFailedImages(ctx context.Context, queueID int64) error {
	return database.GetDB(ctx).WithContext(ctx).
		Model(&model.AITaskImage{}).
		Where("task_id = ? AND status = ?", queueID, model.AITaskImageStatusFailed).
		Updates(map[string]interface{}{
			"status":     model.AITaskImageStatusPending,
			"error":      gorm.Expr("NULL"),
			"updated_at": time.Now(),
		}).Error
}

// ================== 查找队列 ==================

// FindQueuesWithPendingImages 查找有待处理图片的队列
func (r *aiTaskRepository) FindQueuesWithPendingImages(ctx context.Context, limit int) ([]*model.AIQueue, error) {
	db := database.GetDB(ctx).WithContext(ctx)

	// 子查询：获取有待处理图片的队列 ID
	subQuery := db.Model(&model.AITaskImage{}).
		Select("DISTINCT task_id").
		Where("status = ?", model.AITaskImageStatusPending)

	var queues []*model.AIQueue
	err := db.Where("id IN (?)", subQuery).
		Order("created_at ASC").
		Limit(limit).
		Find(&queues).Error

	return queues, err
}

// ================== 兼容旧接口 ==================

// Create 创建任务（兼容旧代码）
func (r *aiTaskRepository) Create(ctx context.Context, task *model.AIQueue) error {
	return database.GetDB(ctx).WithContext(ctx).Create(task).Error
}

// FindByID 根据 ID 查找任务（兼容旧代码）
func (r *aiTaskRepository) FindByID(ctx context.Context, id int64) (*model.AIQueue, error) {
	return r.FindQueueByID(ctx, id)
}

// Update 更新任务（兼容旧代码）
func (r *aiTaskRepository) Update(ctx context.Context, task *model.AIQueue) error {
	return r.UpdateQueue(ctx, task)
}

// Delete 删除任务（兼容旧代码，但在新逻辑中不应使用）
func (r *aiTaskRepository) Delete(ctx context.Context, id int64) error {
	return database.GetDB(ctx).WithContext(ctx).Delete(&model.AIQueue{}, id).Error
}

// CreateTaskImages 批量创建任务图片关联（兼容旧代码）
func (r *aiTaskRepository) CreateTaskImages(ctx context.Context, taskImages []model.AITaskImage) error {
	if len(taskImages) == 0 {
		return nil
	}
	return database.GetDB(ctx).WithContext(ctx).CreateInBatches(taskImages, 100).Error
}

// GetPendingTasks 获取待处理任务（兼容旧代码）
func (r *aiTaskRepository) GetPendingTasks(ctx context.Context, limit int) ([]*model.AIQueue, error) {
	return r.FindQueuesWithPendingImages(ctx, limit)
}

// GetProcessingTasks 获取处理中的任务（兼容旧代码）
func (r *aiTaskRepository) GetProcessingTasks(ctx context.Context) ([]*model.AIQueue, error) {
	var tasks []*model.AIQueue
	err := database.GetDB(ctx).WithContext(ctx).
		Where("status = ?", model.AIQueueStatusProcessing).
		Find(&tasks).Error
	return tasks, err
}

// GetTaskImages 获取任务关联的图片（兼容旧代码）
func (r *aiTaskRepository) GetTaskImages(ctx context.Context, taskID int64, status string) ([]*model.AITaskImage, error) {
	var taskImages []*model.AITaskImage
	query := database.GetDB(ctx).WithContext(ctx).
		Where("task_id = ?", taskID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Preload("Image").Find(&taskImages).Error
	return taskImages, err
}

// GetActiveTasks 获取活跃任务（兼容旧代码）
func (r *aiTaskRepository) GetActiveTasks(ctx context.Context, limit int) ([]*model.AIQueue, error) {
	return r.FindQueuesWithPendingImages(ctx, limit)
}

// CountPendingImages 统计待处理的图片数量（兼容旧代码）
func (r *aiTaskRepository) CountPendingImages(ctx context.Context) (int64, error) {
	var count int64
	err := database.GetDB(ctx).WithContext(ctx).
		Model(&model.AITaskImage{}).
		Where("status = ?", model.AITaskImageStatusPending).
		Count(&count).Error
	return count, err
}

// GetFailedTasks 获取有失败图片的队列（兼容旧代码）
func (r *aiTaskRepository) GetFailedTasks(ctx context.Context) ([]*model.AIQueue, error) {
	db := database.GetDB(ctx).WithContext(ctx)

	// 子查询：获取有失败图片的队列 ID
	subQuery := db.Model(&model.AITaskImage{}).
		Select("DISTINCT task_id").
		Where("status = ?", model.AITaskImageStatusFailed)

	var queues []*model.AIQueue
	err := db.Where("id IN (?)", subQuery).Find(&queues).Error
	return queues, err
}

// ResetFailedTaskImages 重置任务中失败的图片状态（兼容旧代码）
func (r *aiTaskRepository) ResetFailedTaskImages(ctx context.Context, taskID int64) error {
	return r.RetryQueueFailedImages(ctx, taskID)
}

// FindActiveTaskByModelID 查找指定模型的队列（兼容旧代码）
func (r *aiTaskRepository) FindActiveTaskByModelID(ctx context.Context, modelID string) (*model.AIQueue, error) {
	queueKey := model.GenerateQueueKey(model.AITaskTypeEmbedding, modelID)
	var queue model.AIQueue
	err := database.GetDB(ctx).WithContext(ctx).
		Where("queue_key = ?", queueKey).
		First(&queue).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &queue, nil
}

// FindActiveTasksByType 查找指定类型的队列（兼容旧代码）
func (r *aiTaskRepository) FindActiveTasksByType(ctx context.Context, taskType string) ([]*model.AIQueue, error) {
	var queues []*model.AIQueue
	err := database.GetDB(ctx).WithContext(ctx).
		Where("task_type = ?", taskType).
		Find(&queues).Error
	return queues, err
}
