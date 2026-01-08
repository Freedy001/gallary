import http from './http'
import type {ApiResponse} from '@/types'
import type {AIConfig, AIQueueDetail, EmbeddingModelInfo, TestConnectionRequest,} from '@/types/ai'
import type {GenerateSmartAlbumsRequest, SmartAlbumTaskVO} from "@/types/smart-album.ts";

// 优化提示词请求
export interface OptimizePromptRequest {
  query: string
  system_prompt?: string
  model_id?: string
}

// 优化提示词响应
export interface OptimizePromptResponse {
  original_query: string
  optimized_prompt: string
}

export const aiApi = {
  // 获取 AI 配置（通过 settings API）
  getSettings(): Promise<ApiResponse<AIConfig>> {
    return http.get('/api/settings/ai')
  },

  // 更新 AI 配置（通过 settings API）
  updateSettings(config: AIConfig): Promise<ApiResponse<{ message: string }>> {
    return http.put('/api/settings/ai', config)
  },

  // 测试连接（传入完整配置，后端临时创建客户端测试）
  testConnection(request: TestConnectionRequest): Promise<ApiResponse<{ message: string }>> {
    return http.post('/api/ai/test-connection', request)
  },

  // 获取可用的嵌入模型列表（包含模型名称和供应商ID）
  getEmbeddingModels(): Promise<ApiResponse<EmbeddingModelInfo[]>> {
    return http.get('/api/ai/embedding-models')
  },

  // 检测是否配置了指定的默认模型
  configedDefaultModel(type: "DefaultPromptOptimizeModelId" | "DefaultNamingModelId"): Promise<ApiResponse<boolean>> {
    return http.get('/api/settings/configed-default-model/' + type)
  },

  // 优化提示词
  optimizePrompt(request: OptimizePromptRequest): Promise<ApiResponse<OptimizePromptResponse>> {
    return http.post('/api/ai/optimize-prompt', request)
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
    return http.post(`/api/ai/task-items/${taskImageId}/retry`)
  },

  // 忽略单张图片
  ignoreTaskImage(taskImageId: number): Promise<ApiResponse<null>> {
    return http.post(`/api/ai/task-items/${taskImageId}/ignore`)
  },

  // 提交智能相册任务（异步接口，进度通过 WebSocket 推送）
  generateSmartAlbum(request: GenerateSmartAlbumsRequest): Promise<ApiResponse<SmartAlbumTaskVO>> {
    return http.post('/api/ai/smart-albums-generate', request)
  }
}
