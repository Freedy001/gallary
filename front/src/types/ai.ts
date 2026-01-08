// ================== 模型配置类型 ==================

// Provider 模型提供商
export type Provider = 'openAI' | 'selfHosted' | 'alyunMultimodalEmbedding'

// 提示词优化器配置
export interface PromptOptimizerConfig {
  enabled: boolean
  system_prompt: string
}

// 额外配置结构
export interface ExtraConfig {
  prompt_optimizer?: PromptOptimizerConfig
}

// 单个模型项（一个提供商可配置多个模型）
export interface ModelItem {
  api_model_name: string  // API 调用时使用的模型名称
  model_name: string      // 内部标识/负载均衡分组
}

// 嵌入模型信息（包含模型名称和供应商ID）
export interface EmbeddingModelInfo {
  model_name: string   // 模型名称（用于负载均衡分组）
  provider_id: string  // 供应商 ID
}

// 通用模型配置（与后端 ModelConfig 对应）
export interface ModelConfig {
  id: string             // 提供商配置唯一标识
  provider: Provider     // 提供商类型
  models: ModelItem[]    // 模型列表（新）
  endpoint: string       // API 端点
  api_key: string        // API Key
  enabled: boolean       // 是否启用
  extra_config?: string  // 额外配置
}

// AI 全局配置
export interface AIGlobalConfig {
  default_search_model_id: string  // 默认搜索模型 ID（组合格式: providerId,apiModelName）
  default_tag_model_id: string     // 默认打标签模型 ID（组合格式: providerId,apiModelName）
  default_prompt_optimize_model_id: string     // 默认打标签模型 ID（组合格式: providerId,apiModelName）
  prompt_optimize_system_prompt: string  // 提示词优化配置
  default_naming_model_id: string  // 默认命名模型 ID（组合格式: providerId,apiModelName）
  naming_system_prompt: string     // 命名提示词
  naming_max_images: number        // 命名最大图片数量（默认3）
}

// AI 配置（与后端 AIPo 对应）
export interface AIConfig {
  models: ModelConfig[]           // 模型配置列表
  global_config?: AIGlobalConfig  // 全局配置
}

// ================== 模型 ID 辅助函数 ==================

// 创建组合ID
export function createModelId(providerId: string, apiModelName: string): string {
  return `${providerId},${apiModelName}`
}


// ================== 队列相关类型 ==================

// 任务类型
export type TaskType = 'image-embedding' | 'tag-embedding' | 'aesthetic-scoring' | 'album-naming'

// AI 队列状态汇总
export interface AIQueueStatus {
  queues: AIQueueInfo[]      // 所有队列信息
  total_pending: number      // 总待处理数
  total_failed: number       // 总失败数
}

// 单个队列信息
export interface AIQueueInfo {
  id: number
  queue_key: string
  task_type: TaskType
  model_name?: string
  status: 'idle' | 'processing'   // 队列状态
  pending_count: number
  failed_count: number
}

// 队列详情（含失败项目列表）
export interface AIQueueDetail {
  queue: AIQueueInfo
  failed_items: AITaskItemInfo[]  // 失败项目列表
  total_failed: number
  page: number
  page_size: number
}

// 任务项目信息（通用）
export interface AITaskItemInfo {
  id: number
  item_id: number              // 实体 ID（图片ID、标签ID等）
  item_type: TaskType          // 实体类型
  item_name?: string           // 项目名称
  item_thumb?: string          // 缩略图URL
  status: 'pending' | 'failed'
  error?: string
  created_at: string
}

// ================== 请求类型 ==================

// 测试连接请求 - 传入完整的模型配置用于临时创建客户端测试
export interface TestConnectionRequest {
  provider: ModelConfig      // 完整的提供商配置
  model?: ModelItem          // 可选的模型项（非自托管模型时需要指定）
}

// ================== 显示标签 ==================

// 任务类型显示名称
export const TaskTypeLabels: Record<TaskType, string> = {
  'image-embedding': '图片向量嵌入',
  'tag-embedding': '标签向量嵌入',
  'album-naming': '相册命名',
  'aesthetic-scoring': '美学评分'
}

// ================== 辅助函数 ==================

// 获取队列显示名称
export function getQueueDisplayName(queue: AIQueueInfo): string {
  const typeLabel = TaskTypeLabels[queue.task_type] || queue.task_type
  if (queue.model_name) {
    return `${typeLabel} - ${queue.model_name}`
  }
  return typeLabel
}
