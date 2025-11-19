import http from './http'
import type {ApiResponse, Image, Pageable, SearchParams} from '@/types'

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

  // 下载图片
  download(id: number, filename: string): Promise<void> {
    return http.download(`/api/images/${id}/download`, filename)
  },

  // 搜索图片
  search(params: SearchParams): Promise<ApiResponse<Pageable<Image>>> {
    return http.get('/api/search', {params})
  },

  // 获取图片 URL
  getImageUrl(storagePath: string): string {
    const baseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:9099'
    return `${baseUrl}/static/images/${storagePath}`
  },
}
