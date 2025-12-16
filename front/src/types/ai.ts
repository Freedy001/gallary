// ================== 模型配置类型 ==================

// Provider 模型提供商
export type Provider = 'openAI' | 'selfHosted' | 'alyunMultimodalEmbedding'

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

// AI 配置（与后端 AIPo 对应）
export interface AIConfig {
  models: ModelConfig[]  // 模型配置列表
}

// ================== 队列相关类型 ==================

// AI 队列状态汇总
export interface AIQueueStatus {
  queues: AIQueueInfo[]      // 所有队列信息
  total_pending: number      // 总待处理数
  total_processing: number   // 总处理中数
  total_failed: number       // 总失败数
}

// 单个队列信息
export interface AIQueueInfo {
  id: number
  queue_key: string
  task_type: 'embedding' | 'description'
  model_name?: string
  status: 'idle' | 'processing'   // 队列状态
  pending_count: number
  processing_count: number
  failed_count: number
}

// 队列详情（含失败图片列表）
export interface AIQueueDetail {
  queue: AIQueueInfo
  failed_images: AITaskImageInfo[]
  total_failed: number
  page: number
  page_size: number
}

// 任务图片信息
export interface AITaskImageInfo {
  id: number
  image_id: number
  image_path: string
  thumbnail?: string
  status: 'pending' | 'processing' | 'failed'
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
export const TaskTypeLabels: Record<string, string> = {
  embedding: '向量嵌入',
  description: 'AI 描述'
}

// 队列状态显示名称
export const QueueStatusLabels: Record<string, string> = {
  idle: '空闲',
  processing: '处理中'
}

// 任务图片状态显示名称
export const TaskImageStatusLabels: Record<string, string> = {
  pending: '待处理',
  processing: '处理中',
  failed: '失败'
}

// ================== 辅助函数 ==================

// 获取所有启用的模型
export function getEnabledModels(config: AIConfig): ModelConfig[] {
  return config.models.filter(model => model.enabled)
}

// 获取队列显示名称
export function getQueueDisplayName(queue: AIQueueInfo): string {
  const typeLabel = TaskTypeLabels[queue.task_type] || queue.task_type
  if (queue.model_name) {
    return `${typeLabel} - ${queue.model_name}`
  }
  return typeLabel
}
