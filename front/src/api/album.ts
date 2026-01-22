import http from './http'
import type {Album, CreateAlbumRequest, Image, Pageable, UpdateAlbumRequest} from '@/types'

export interface AlbumListParams {
  page?: number
  pageSize?: number
  isSmart?: boolean  // true-只返回智能相册，false-只返回普通相册，不传-返回全部
}

export const albumApi = {
  // 获取相册列表
  getList(params: AlbumListParams = {}) {
    const { page = 1, pageSize = 20, isSmart } = params
    return http.get<Pageable<Album>>('/api/albums', {
      params: {
        page,
        page_size: pageSize,
        ...(isSmart !== undefined && { is_smart: isSmart })
      }
    })
  },

  // 根据 ID 批量获取相册
  getByIds(ids: number[]) {
    return http.post<Album[]>('/api/albums/batch-get', { ids })
  },


  // 创建相册
  create(data: CreateAlbumRequest) {
    return http.post<Album>('/api/albums', data)
  },

  // 更新相册
  update(id: number, data: UpdateAlbumRequest) {
    return http.put<Album>(`/api/albums/${id}`, data)
  },

  // 删除相册
  delete(ids: number[]) {
    return http.post('/api/albums/batch-delete', { ids })
  },

  // 复制相册
  copy(ids: number[]) {
    return http.post<Album[]>('/api/albums/batch-copy', { ids })
  },

  // 获取相册内图片
  getImages(id: number, page: number = 1, pageSize: number = 20, sortBy = 'taken_at') {
    return http.get<Pageable<Image>>(`/api/albums/${id}/images`, {
      params: { page, page_size: pageSize, sort_by: sortBy }
    })
  },

  // 添加图片到相册
  addImages(id: number, imageIds: number[]) {
    return http.post(`/api/albums/${id}/images`, { image_ids: imageIds })
  },

  // 从相册移除图片
  removeImages(id: number, imageIds: number[]) {
    return http.delete(`/api/albums/${id}/images`, {
      data: { image_ids: imageIds }
    })
  },

  // 设置相册封面
  setCover(id: number, imageId: number) {
    return http.put(`/api/albums/${id}/cover`, { image_id: imageId })
  },

  // 移除相册封面
  removeCover(id: number) {
    return http.delete(`/api/albums/${id}/cover`)
  },

  // 设置平均向量封面
  setAverageCover(id: number, modelName: string) {
    return http.put(`/api/albums/${id}/cover/average`, { model_name: modelName })
  },

  // AI 命名相册（添加到队列）
  aiNaming(ids: number[]) {
    return http.post<{ added: number }>('/api/albums/ai-naming', { ids })
  },

  // 合并相册
  merge(sourceIds: number[], targetId: number, keepSource: boolean = false) {
    return http.post('/api/albums/batch-merge', { source_ids: sourceIds, target_id: targetId, keep_source: keepSource })
  },
}
