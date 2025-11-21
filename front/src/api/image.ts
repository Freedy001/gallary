import http from './http'
import type {
  ApiResponse,
  Image,
  Pageable,
  SearchParams,
  UpdateMetadataRequest,
} from '@/types'

export const imageApi = {
  // 上传图片
  upload(file: File, onProgress?: (progress: number) => void): Promise<ApiResponse<Image>> {
    const formData = new FormData()
    formData.append('file', file)
    return http.upload('/api/images/upload', formData, onProgress)
  },

  // 获取图片列表
  getList(page = 1, pageSize = 20): Promise<ApiResponse<Pageable<Image>>> {
    return http.get(`/api/images`, {
      params: {page, page_size: pageSize}
    })
  },

  // 获取图片详情
  getDetail(id: number): Promise<ApiResponse<Image>> {
    return http.get(`/api/images/${id}`)
  },

  // 删除图片
  delete(id: number): Promise<ApiResponse<null>> {
    return http.delete(`/api/images/${id}`)
  },

  // 批量删除图片
  deleteBatch(ids: number[]): Promise<ApiResponse<null>> {
    return http.post('/api/images/batch-delete', {ids})
  },

  // 下载图片
  download(id: number, filename: string): Promise<void> {
    return http.download(`/api/images/${id}/download`, filename)
  },

  // 搜索图片
  search(params: SearchParams): Promise<ApiResponse<Pageable<Image>>> {
    return http.get('/api/search', {params})
  },

  // 更新单个图片元数据
  updateMetadata(data: UpdateMetadataRequest): Promise<ApiResponse<Image>> {
    return http.put(`/api/images/metadata`, data)
  },

  // 获取图片 URL
  getImageUrl(storagePath: string): string {
    return `/static/images/${storagePath}`
  },
}
