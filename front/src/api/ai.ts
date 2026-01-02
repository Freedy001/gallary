import http from './http'
import type {ApiResponse} from '@/types'
import type {AIConfig, AIQueueDetail, TestConnectionRequest,} from '@/types/ai'

export const aiApi = {
  // 获取 AI 配置（通过 settings API）
  getSettings(): Promise<ApiResponse<AIConfig>> {
    return http.get('/api/settings/ai')
  },

  // 更新 AI 配置（通过 settings API）
  updateSettings(config: AIConfig): Promise<ApiResponse<{ message: string }>> {
    return http.put('/api/settings/ai', config)
  },

  // 测试连接
  testConnection(request: TestConnectionRequest): Promise<ApiResponse<{ message: string }>> {
    return http.post('/api/ai/test', request)
  },

  // 获取可用的嵌入模型列表
  getEmbeddingModels(): Promise<ApiResponse<string[]>> {
    return http.get('/api/ai/embedding-models')
  },

  // ================== 队列管理 ==================
  // 获取队列详情（含失败图片列表）
  getQueueDetail(queueId: number, page = 1, pageSize = 20): Promise<ApiResponse<AIQueueDetail>> {
    return http.get(`/api/ai/queues/${queueId}`, {
      params: {page, page_size: pageSize}
    })
  },

  // 重试队列所有失败图片
  retryQueueFailedImages(queueId: number): Promise<ApiResponse<null>> {
    return http.post(`/api/ai/queues/${queueId}/retry`)
  },

  // ================== 单张图片操作 ==================

  // 重试单张图片
  retryTaskImage(taskImageId: number): Promise<ApiResponse<null>> {
    return http.post(`/api/ai/task-images/${taskImageId}/retry`)
  },

  // 忽略单张图片
  ignoreTaskImage(taskImageId: number): Promise<ApiResponse<null>> {
    return http.post(`/api/ai/task-images/${taskImageId}/ignore`)
  },
}
