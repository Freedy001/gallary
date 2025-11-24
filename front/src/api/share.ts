import http from './http'
import type {CreateShareRequest, Image, Pageable, Share, SharePublicInfo} from '@/types'

export const shareApi = {
  // 创建分享
  create(data: CreateShareRequest) {
    return http.post<Share>('/api/shares', data)
  },

  // 获取分享列表
  getList(page: number = 1, size: number = 20) {
    return http.get<Pageable<Share>>('/api/shares', {
      params: { page, page_size: size },
    })
  },

  // 删除分享
  delete(id: number) {
    return http.delete(`/api/shares/${id}`)
  },

  // 获取公开分享信息
  getPublicInfo(code: string) {
    return http.get<SharePublicInfo>(`/api/s/${code}/info`)
  },

  // 验证并获取分享详情（支持分页）
  getImages(code: string, password?: string, page: number = 1, pageSize: number = 20) {
    return http.post<Pageable<Image>>(`/api/s/${code}/images`, { password }, {
      params: { page, page_size: pageSize }
    })
  },
}
