import http from './http'
import type { ApiResponse, Image } from '@/types'
import type {
  AIConfig,
  AIQueueStatus,
  AIQueueDetail,
  SemanticSearchRequest,
  TestConnectionRequest,
} from '@/types/ai'

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

  // ================== 队列管理 ==================

  // 获取所有队列状态
  getQueueStatus(): Promise<ApiResponse<AIQueueStatus>> {
    return http.get('/api/ai/queues')
  },

  // 获取队列详情（含失败图片列表）
  getQueueDetail(queueId: number, page = 1, pageSize = 20): Promise<ApiResponse<AIQueueDetail>> {
    return http.get(`/api/ai/queues/${queueId}`, {
      params: { page, page_size: pageSize }
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

  // ================== 搜索 ==================

  // 语义搜索
  semanticSearch(request: SemanticSearchRequest): Promise<ApiResponse<Image[]>> {
    return http.post('/api/ai/search', request)
  },
}
