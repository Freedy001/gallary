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

// 通用模型配置（与后端 ModelConfig 对应）
export interface ModelConfig {
  id: string             // 唯一标识
  provider: Provider     // 提供商
  model_name: string     // 模型名称
  api_model_name: string     // 模型名称
  endpoint: string       // API 端点
  api_key: string        // API Key
  enabled: boolean       // 是否启用
  extra_config?: string  // 额外配置
  // 前端扩展字段（用于显示）
  dimension?: number     // 向量维度
}

// AI 全局配置
export interface AIGlobalConfig {
  default_search_model_id: string  // 默认搜索模型 ID
  default_tag_model_id: string     // 默认打标签模型 ID
}

// AI 配置（与后端 AIPo 对应）
export interface AIConfig {
  models: ModelConfig[]           // 模型配置列表
  global_config?: AIGlobalConfig  // 全局配置
}

// ================== 队列相关类型 ==================

// 任务类型
export type TaskType = 'image-embedding' | 'tag-embedding' | 'aesthetic-scoring'

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

// 测试连接请求
export interface TestConnectionRequest {
  id: string  // 模型ID
}

// 语义搜索请求
export interface SemanticSearchRequest {
  query: string
  model_name?: string
  limit?: number
}

// ================== 显示标签 ==================

// 任务类型显示名称
export const TaskTypeLabels: Record<TaskType, string> = {
  'image-embedding': '图片向量嵌入',
  'tag-embedding': '标签向量嵌入',
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
