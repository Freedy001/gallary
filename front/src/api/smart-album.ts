import http from './http'
import type {ApiResponse} from '@/types'
import type {GenerateSmartAlbumsRequest, GenerateSmartAlbumsResponse, SmartAlbumTaskVO} from '@/types/smart-album'

export const smartAlbumApi = {
  // 生成智能相册（同步接口，保持向后兼容）
  generate(request: GenerateSmartAlbumsRequest): Promise<ApiResponse<GenerateSmartAlbumsResponse>> {
    return http.post('/api/albums/smart-generate', request)
  },

  // 提交智能相册任务（异步接口，进度通过 WebSocket 推送）
  submitTask(request: GenerateSmartAlbumsRequest): Promise<ApiResponse<SmartAlbumTaskVO>> {
    return http.post('/api/albums/smart-tasks', request)
  }
}
