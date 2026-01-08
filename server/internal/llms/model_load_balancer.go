package llms

import (
	"fmt"
	"gallary/server/internal"
	"gallary/server/internal/model"
	"gallary/server/internal/storage"
	"time"

	"net/http"
	"sync"
	"sync/atomic"

	"gallary/server/pkg/logger"

	"github.com/samber/lo"
	"go.uber.org/zap"
)

// ModelLoadBalancer 模型负载均衡器
type ModelLoadBalancer struct {
	storageManager *storage.StorageManager
	httpClient     *http.Client

	// 模型客户端缓存（按 ID 缓存）
	modelClients map[string]ModelClient
	modelMu      sync.RWMutex

	// 负载均衡计数器（按 ModelId 分组）
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
// cacheKey 使用组合ID格式: providerId,apiModelName
func (lb *ModelLoadBalancer) getOrCreateClient(provider *model.ModelConfig, modelItem *model.ModelItem) ModelClient {
	cacheKey := string(model.CreateModelId(provider.ID, modelItem.ApiModelName))

	lb.modelMu.RLock()
	if client, exists := lb.modelClients[cacheKey]; exists {
		if client.GetConfig().Hash() != provider.Hash() {
			client.UpdateConfig(provider)
		}
		lb.modelMu.RUnlock()
		return client
	}
	lb.modelMu.RUnlock()

	lb.modelMu.Lock()
	defer lb.modelMu.Unlock()

	// 双重检查
	if client, exists := lb.modelClients[cacheKey]; exists {
		return client
	}

	// 创建新的客户端
	client := CreateModelClient(provider, modelItem, lb.httpClient, lb.storageManager)
	lb.modelClients[cacheKey] = client
	return client
}

// selectProviderByRoundRobin 使用轮询算法选择提供商
func (lb *ModelLoadBalancer) selectProviderByRoundRobin(modelName string, providers []*model.ProviderWithModelItem) *model.ProviderWithModelItem {
	if len(providers) == 1 {
		return providers[0]
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
	idx := atomic.AddUint64(counter, 1) % uint64(len(providers))
	return providers[idx]
}

func (lb *ModelLoadBalancer) GetAllgModelName() ([]string, error) {
	models, err := lb.selectModelWithProvider(func(client ModelClient) bool { return true })
	if err != nil {
		return make([]string, 0), err
	}
	return lo.Uniq(lo.Map(models, func(item *model.ProviderAndModelName, index int) string { return item.ModelName })), nil
}

// GetAllEmbeddingModelsWithProvider 获取所有嵌入模型信息（包含供应商ID）
func (lb *ModelLoadBalancer) GetAllEmbeddingModelsWithProvider() ([]*model.ProviderAndModelName, error) {
	return lb.selectModelWithProvider(func(client ModelClient) bool { _, ok := client.(EmbeddingClient); return ok })
}

// selectModelWithProvider 选择支持指定功能的模型，返回模型名称和供应商ID
func (lb *ModelLoadBalancer) selectModelWithProvider(support func(client ModelClient) bool) ([]*model.ProviderAndModelName, error) {
	return lo.FlatMap(internal.PlatConfig.AIPo.GetEnabled(), func(provider *model.ModelConfig, index int) []*model.ProviderAndModelName {
		return lo.FilterMap(provider.Models, func(modelItem *model.ModelItem, index int) (*model.ProviderAndModelName, bool) {
			client := lb.getOrCreateClient(provider, modelItem)
			if client == nil || !support(client) {
				return nil, false
			}
			return &model.ProviderAndModelName{
				ModelName:  modelItem.ModelName,
				ProviderID: provider.ID,
			}, true
		})
	}), nil
}

// GetClientByName 根据模型名称获取客户端（支持负载均衡）
// 返回客户端、提供商配置和模型项
func (lb *ModelLoadBalancer) GetClientByName(modelName string) (ModelClient, error) {
	config := internal.PlatConfig.AIPo

	// 获取该模型名称对应的所有启用的提供商配置
	providers := config.FindModelConfigByModelName(modelName)
	if len(providers) == 0 {
		return nil, fmt.Errorf("未找到模型配置: %s", modelName)
	}

	// 负载均衡：轮询选择
	selected := lb.selectProviderByRoundRobin(modelName, providers)

	client := lb.getOrCreateClient(selected.Provider, selected.ModelItem)
	if client == nil {
		return nil, fmt.Errorf("无法获取模型客户端: %s,%s", selected.Provider.ID, selected.ModelItem.ApiModelName)
	}

	return client, nil
}

// GetClientByID 根据组合模型ID获取客户端
// compositeId 格式: "providerId,apiModelName" 或 "providerId"
func (lb *ModelLoadBalancer) GetClientByID(compositeId model.CopositModelId) (ModelClient, error) {
	config := internal.PlatConfig.AIPo

	// 查找模型配置
	provider, modelItem := config.FindById(compositeId)
	if provider == nil || modelItem == nil {
		return nil, fmt.Errorf("未找到模型配置: %s", compositeId)
	}

	client := lb.getOrCreateClient(provider, modelItem)
	if client == nil {
		return nil, fmt.Errorf("无法获取模型客户端: %s", compositeId)
	}

	return client, nil
}

// TryAllProviders 尝试所有提供商执行操作，如果都失败则返回错误
// operation 是一个接受 client、provider 和 modelItem 的函数，返回错误表示失败
// 使用负载均衡选择起始提供商，确保请求分散到不同的提供商
func (lb *ModelLoadBalancer) TryAllProviders(modelName string, operation func(ModelClient, *model.ModelConfig, *model.ModelItem) error) error {
	config := internal.PlatConfig.AIPo

	providers := config.FindModelConfigByModelName(modelName)
	if len(providers) == 0 {
		return fmt.Errorf("未找到可用的提供商: %s", modelName)
	}

	// 使用负载均衡选择起始索引
	startProvider := lb.selectProviderByRoundRobin(modelName, providers)
	var startIdx int
	for i, p := range providers {
		if p.Provider.ID == startProvider.Provider.ID {
			startIdx = i
			break
		}
	}

	var lastErr error
	// 从负载均衡选择的提供商开始，依次尝试所有提供商
	for i := 0; i < len(providers); i++ {
		idx := (startIdx + i) % len(providers)
		p := providers[idx]

		client := lb.getOrCreateClient(p.Provider, p.ModelItem)
		if client == nil {
			lastErr = fmt.Errorf("无法获取模型客户端: %s,%s", p.Provider.ID, p.ModelItem.ApiModelName)
			logger.Warn("跳过模型提供商",
				zap.String("model_name", modelName),
				zap.String("provider", string(p.Provider.Provider)),
				zap.Error(lastErr))
			continue
		}

		err := operation(client, p.Provider, p.ModelItem)
		if err == nil {
			// 成功
			if i > 0 {
				logger.Info("使用备用提供商成功",
					zap.String("model_name", modelName),
					zap.String("provider", string(p.Provider.Provider)),
					zap.Int("tried_count", i+1))
			}
			return nil
		}

		// 记录失败并尝试下一个提供商
		lastErr = err
		logger.Warn("模型提供商操作失败，尝试下一个",
			zap.String("model_name", modelName),
			zap.String("provider", string(p.Provider.Provider)),
			zap.String("provider_id", p.Provider.ID),
			zap.Error(err))
	}

	// 所有提供商都失败
	if lastErr != nil {
		return fmt.Errorf("所有提供商都失败，最后错误: %v", lastErr)
	}

	return fmt.Errorf("未找到可用的提供商")
}

// CreateTemporaryClient 创建临时客户端用于测试连接
// 不缓存客户端，每次调用都创建新的实例
func (lb *ModelLoadBalancer) CreateTemporaryClient(provider *model.ModelConfig, modelItem *model.ModelItem) (ModelClient, error) {
	if provider == nil {
		return nil, fmt.Errorf("provider 配置不能为空")
	}

	// 对于自托管模型，不需要 modelItem
	if provider.Provider == model.SelfHosted {
		// 创建一个默认的 modelItem
		if modelItem == nil {
			modelItem = &model.ModelItem{
				ApiModelName: "self-hosted",
				ModelName:    "self-hosted",
			}
		}
	} else {
		// 非自托管模型必须有 modelItem
		if modelItem == nil {
			return nil, fmt.Errorf("非自托管模型必须指定 model")
		}
	}

	// 直接创建客户端，不缓存
	client := CreateModelClient(provider, modelItem, lb.httpClient, lb.storageManager)
	if client == nil {
		return nil, fmt.Errorf("不支持的提供商类型: %s", provider.Provider)
	}

	return client, nil
}
