package llms

import (
	"fmt"
	"gallary/server/internal"
	"gallary/server/internal/model"
	"gallary/server/internal/storage"
	"maps"
	"slices"
	"time"

	"net/http"
	"sync"
	"sync/atomic"

	"gallary/server/pkg/logger"

	"go.uber.org/zap"
)

// ModelLoadBalancer 模型负载均衡器
type ModelLoadBalancer struct {
	storageManager *storage.StorageManager
	httpClient     *http.Client

	// 模型客户端缓存（按 ID 缓存）
	modelClients map[string]ModelClient
	modelMu      sync.RWMutex

	// 负载均衡计数器（按 ModelName 分组）
	loadBalanceCounters map[string]*uint64
	lbMu                sync.RWMutex
}

// NewModelLoadBalancer 创建模型负载均衡器
func NewModelLoadBalancer(manager *storage.StorageManager) *ModelLoadBalancer {
	return &ModelLoadBalancer{
		storageManager:      manager,
		httpClient:          &http.Client{Timeout: 120 * time.Second},
		modelClients:        make(map[string]ModelClient),
		loadBalanceCounters: make(map[string]*uint64),
	}
}

// getOrCreateClient 获取或创建模型客户端
func (lb *ModelLoadBalancer) getOrCreateClient(modelConfig *model.ModelConfig) ModelClient {
	lb.modelMu.RLock()
	if client, exists := lb.modelClients[modelConfig.ID]; exists {
		if client.GetConfig().Hash() != modelConfig.Hash() {
			client.UpdateConfig(modelConfig)
		}
		lb.modelMu.RUnlock()
		return client
	}
	lb.modelMu.RUnlock()

	lb.modelMu.Lock()
	defer lb.modelMu.Unlock()

	// 双重检查
	if client, exists := lb.modelClients[modelConfig.ID]; exists {
		return client
	}

	// 创建新的客户端
	client := CreateModelClient(modelConfig, lb.httpClient, lb.storageManager)
	lb.modelClients[modelConfig.ID] = client
	return client
}

// selectModelByRoundRobin 使用轮询算法选择模型
func (lb *ModelLoadBalancer) selectModelByRoundRobin(modelName string, models []*model.ModelConfig) *model.ModelConfig {
	if len(models) == 1 {
		return models[0]
	}

	lb.lbMu.RLock()
	counter, exists := lb.loadBalanceCounters[modelName]
	lb.lbMu.RUnlock()

	if !exists {
		lb.lbMu.Lock()
		// 双重检查
		if counter, exists = lb.loadBalanceCounters[modelName]; !exists {
			var zero uint64
			counter = &zero
			lb.loadBalanceCounters[modelName] = counter
		}
		lb.lbMu.Unlock()
	}

	// 原子递增并取模
	idx := atomic.AddUint64(counter, 1) % uint64(len(models))
	return models[idx]
}

func (lb *ModelLoadBalancer) GetAllEmbeddingModels() ([]string, error) {
	config := internal.PlatConfig.AIPo
	// 获取所有启用的模型
	enabledModels := config.GetEnabledModels()
	if len(enabledModels) == 0 {
		return nil, nil
	}

	// 按 ModelName 分组，每个 ModelName 只需要一个队列
	modelNames := make(map[string]bool)
	for _, modelConfig := range enabledModels {
		client := lb.getOrCreateClient(modelConfig)
		if client == nil || !client.SupportEmbedding() {
			continue
		}
		// 所有启用的模型都可能支持嵌入，在实际处理时会检查
		modelNames[modelConfig.ModelName] = true
	}

	return slices.Collect(maps.Keys(modelNames)), nil
}

// GetClientByName 根据模型名称获取客户端（支持负载均衡）
func (lb *ModelLoadBalancer) GetClientByName(modelName string) (ModelClient, *model.ModelConfig, error) {
	config := internal.PlatConfig.AIPo

	// 获取该模型名称对应的所有启用的模型配置
	models := config.FindModelsByName(modelName)
	if len(models) == 0 {
		return nil, nil, fmt.Errorf("未找到模型配置: %s", modelName)
	}

	// 负载均衡：轮询选择
	modelConfig := lb.selectModelByRoundRobin(modelName, models)

	client := lb.getOrCreateClient(modelConfig)
	if client == nil {
		return nil, nil, fmt.Errorf("无法获取模型客户端: %s", modelConfig.ID)
	}

	return client, modelConfig, nil
}

// GetClientByID 根据模型ID获取客户端
func (lb *ModelLoadBalancer) GetClientByID(modelID string) (ModelClient, *model.ModelConfig, error) {
	config := internal.PlatConfig.AIPo

	// 查找模型配置
	modelConfig := config.FindModelById(modelID)
	if modelConfig == nil {
		return nil, nil, fmt.Errorf("未找到模型配置: %s", modelID)
	}

	client := lb.getOrCreateClient(modelConfig)
	if client == nil {
		return nil, nil, fmt.Errorf("无法获取模型客户端: %s", modelID)
	}

	return client, modelConfig, nil
}

// getAllEnabledModelsByName 获取指定模型名称的所有启用的模型配置
func (lb *ModelLoadBalancer) getAllEnabledModelsByName(modelName string) ([]*model.ModelConfig, error) {
	config := internal.PlatConfig.AIPo

	models := config.FindModelsByName(modelName)
	if len(models) == 0 {
		return nil, fmt.Errorf("未找到模型配置: %s", modelName)
	}

	return models, nil
}

// TryAllProviders 尝试所有提供商执行操作，如果都失败则返回错误
// operation 是一个接受 client 和 modelConfig 的函数，返回错误表示失败
// 使用负载均衡选择起始提供商，确保请求分散到不同的提供商
func (lb *ModelLoadBalancer) TryAllProviders(modelName string, operation func(ModelClient, *model.ModelConfig) error) error {
	models, err := lb.getAllEnabledModelsByName(modelName)
	if err != nil {
		return err
	}

	if len(models) == 0 {
		return fmt.Errorf("未找到可用的提供商")
	}

	// 使用负载均衡选择起始索引
	startModel := lb.selectModelByRoundRobin(modelName, models)
	var startIdx int
	for i, m := range models {
		if m.ID == startModel.ID {
			startIdx = i
			break
		}
	}

	var lastErr error
	// 从负载均衡选择的提供商开始，依次尝试所有提供商
	for i := 0; i < len(models); i++ {
		idx := (startIdx + i) % len(models)
		modelConfig := models[idx]

		client := lb.getOrCreateClient(modelConfig)
		if client == nil {
			lastErr = fmt.Errorf("无法获取模型客户端: %s", modelConfig.ID)
			logger.Warn("跳过模型提供商",
				zap.String("model_name", modelName),
				zap.String("provider", string(modelConfig.Provider)),
				zap.Error(lastErr))
			continue
		}

		err := operation(client, modelConfig)
		if err == nil {
			// 成功
			if i > 0 {
				logger.Info("使用备用提供商成功",
					zap.String("model_name", modelName),
					zap.String("provider", string(modelConfig.Provider)),
					zap.Int("tried_count", i+1))
			}
			return nil
		}

		// 记录失败并尝试下一个提供商
		lastErr = err
		logger.Warn("模型提供商操作失败，尝试下一个",
			zap.String("model_name", modelName),
			zap.String("provider", string(modelConfig.Provider)),
			zap.String("model_id", modelConfig.ID),
			zap.Error(err))
	}

	// 所有提供商都失败
	if lastErr != nil {
		return fmt.Errorf("所有提供商都失败，最后错误: %v", lastErr)
	}

	return fmt.Errorf("未找到可用的提供商")
}
