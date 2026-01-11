import http from './http'
import type {
  ApiResponse,
  ClusterResult,
  GeoBounds,
  Image,
  Pageable,
  SearchParams,
  Tag,
  UpdateMetadataRequest,
} from '@/types'

export const imageApi = {
  // 上传图片
  upload(file: File, albumId?: number, onProgress?: (progress: number) => void): Promise<ApiResponse<Image>> {
    const formData = new FormData()
    formData.append('file', file)
    if (albumId) {
      formData.append('album_id', albumId.toString())
    }
    return http.upload('/api/images/upload', formData, onProgress)
  },

  // 获取图片列表
  getList(page = 1, pageSize = 20, sortBy = 'taken_at'): Promise<ApiResponse<Pageable<Image>>> {
    return http.get(`/api/images`, {
      params: {page, page_size: pageSize, sort_by: sortBy}
    })
  },

  // 批量获取图片（根据ID列表）
  getByIds(ids: number[]): Promise<ApiResponse<Image[]>> {
    return http.post('/api/images/batch', {ids})
  },

  // 获取标签列表（支持搜索）
  getTags(keyword?: string, limit?: number): Promise<ApiResponse<Tag[]>> {
    const params: Record<string, string | number> = {}
    if (keyword) params.keyword = keyword
    if (limit) params.limit = limit
    return http.get('/api/tags', {params})
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

  downloadBatch(images: Image[]): Promise<void> {
    return Promise.all(images.filter(i => i && i.id).map(ims => http.download(`/api/images/${ims.id}/download`, ims.original_name))).then()
  },

  // 批量下载图片（打包为 ZIP，流式下载）
  downloadZipped(ids: number[]): void {
    // 创建隐藏的表单提交，让浏览器直接处理下载
    const form = document.createElement('form')
    form.method = 'POST'
    form.action = '/api/images/batch-download'
    form.style.display = 'none'

    // 添加 ids 参数
    const input = document.createElement('input')
    input.type = 'hidden'
    input.name = 'ids'
    input.value = JSON.stringify(ids)
    form.appendChild(input)

    // 添加 token
    const token = localStorage.getItem('auth_token')
    if (token) {
      const tokenInput = document.createElement('input')
      tokenInput.type = 'hidden'
      tokenInput.name = 'token'
      tokenInput.value = token
      form.appendChild(tokenInput)
    }

    document.body.appendChild(form)
    form.submit()
    document.body.removeChild(form)
  },

  // 搜索图片（支持图片上传进行以图搜图）
  search(params: SearchParams): Promise<ApiResponse<Pageable<Image>>> {
    if (params.file) {
      // 有图片时使用 FormData（multipart/form-data）
      const formData = new FormData()
      formData.append('file', params.file)
      params.file = undefined
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null && value !== '') {
          if (Array.isArray(value)) {
            value.forEach(v => formData.append(key, v.toString()))
          } else {
            formData.append(key, value.toString())
          }
        }
      })
      // 使用 upload 方法确保 Content-Type 为 multipart/form-data
      return http.upload('/api/search', formData)
    }
    // 无图片保持原有 JSON 方式
    return http.post('/api/search', params)
  },

  // 更新单个图片元数据
  updateMetadata(data: UpdateMetadataRequest): Promise<ApiResponse<number[]>> {
    return http.put(`/api/images/metadata`, data)
  },

  // 获取图片聚合数据
  getClusters(minLat: number, maxLat: number, minLng: number, maxLng: number, zoom: number): Promise<ApiResponse<ClusterResult[]>> {
    return http.get('/api/images/clusters', {
      params: {
        min_lat: minLat,
        max_lat: maxLat,
        min_lng: minLng,
        max_lng: maxLng,
        zoom
      }
    })
  },

  // 获取聚合组内的图片
  getClusterImages(minLat: number, maxLat: number, minLng: number, maxLng: number, page = 1, pageSize = 20): Promise<ApiResponse<Pageable<Image>>> {
    return http.get('/api/images/clusters/images', {
      params: {
        min_lat: minLat,
        max_lat: maxLat,
        min_lng: minLng,
        max_lng: maxLng,
        page,
        page_size: pageSize
      }
    })
  },

  // 获取图片地理边界
  getGeoBounds(): Promise<ApiResponse<GeoBounds | null>> {
    return http.get('/api/images/geo-bounds')
  },

  // 回收站相关 API
  // 获取已删除图片列表
  getDeletedList(page = 1, pageSize = 20): Promise<ApiResponse<Pageable<Image>>> {
    return http.get('/api/images/trash', {
      params: {page, page_size: pageSize}
    })
  },

  // 恢复已删除图片
  restoreImages(ids: number[]): Promise<ApiResponse<null>> {
    return http.post('/api/images/trash/restore', {ids})
  },

  // 彻底删除图片
  permanentlyDelete(ids: number[]): Promise<ApiResponse<null>> {
    return http.post('/api/images/trash/delete', {ids})
  },
}
