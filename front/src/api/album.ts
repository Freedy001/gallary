import http from './http'
import type { Album, CreateAlbumRequest, UpdateAlbumRequest, Image, Pageable } from '@/types'

export const albumApi = {
  // 获取相册列表
  getList(page: number = 1, pageSize: number = 20) {
    return http.get<Pageable<Album>>('/api/albums', {
      params: { page, page_size: pageSize }
    })
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
  delete(id: number) {
    return http.delete(`/api/albums/${id}`)
  },

  // 获取相册内图片
  getImages(id: number, page: number = 1, pageSize: number = 20) {
    return http.get<Pageable<Image>>(`/api/albums/${id}/images`, {
      params: { page, page_size: pageSize }
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
}
