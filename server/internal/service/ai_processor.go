package service

import (
	"context"
	"sync"

	"gallary/server/internal/llms"
	"gallary/server/internal/model"
)

// AITaskProcessor AI 任务处理器接口（通用）
// 可处理任意类型的实体：图片、标签、文本等
type AITaskProcessor interface {
	// TaskType 返回任务类型标识
	TaskType() model.TaskType
	// FindPendingItems 查找需要处理的项目ID列表
	// modelName: 模型名称（用于按模型分组）
	// limit: 最大返回数量
	// 返回: 项目ID列表（可以是图片ID、标签ID等任意实体ID）
	FindPendingItems(ctx context.Context, modelName string, limit int) ([]int64, error)

	// ProcessItem 处理单个项目
	// itemID: 项目ID（可以是图片ID、标签ID等）
	ProcessItem(ctx context.Context, itemID int64, client llms.ModelClient, config *model.ModelConfig, modelItem *model.ModelItem) error

	// SupportedBy 检查模型客户端是否支持此任务
	SupportedBy(client llms.ModelClient) bool
}

// processorRegistry 全局处理器注册表
var processorRegistry = make(map[model.TaskType]AITaskProcessor)
var processorMu sync.RWMutex

// RegisterProcessor 注册处理器
func RegisterProcessor(processor AITaskProcessor) {
	processorMu.Lock()
	defer processorMu.Unlock()
	processorRegistry[processor.TaskType()] = processor
}

// GetProcessor 获取处理器
func GetProcessor(taskType model.TaskType) (AITaskProcessor, bool) {
	processorMu.RLock()
	defer processorMu.RUnlock()
	p, ok := processorRegistry[taskType]
	return p, ok
}

// GetAllProcessors 获取所有处理器
func GetAllProcessors() []AITaskProcessor {
	processorMu.RLock()
	defer processorMu.RUnlock()
	processors := make([]AITaskProcessor, 0, len(processorRegistry))
	for _, p := range processorRegistry {
		processors = append(processors, p)
	}
	return processors
}
