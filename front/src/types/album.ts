import type { Image } from './image'

// 相册类型
export interface Album {
  id: number
  name: string
  description?: string
  cover_image?: Image
  image_count: number
  sort_order: number
  created_at: string
  updated_at: string
}

// 创建相册请求
export interface CreateAlbumRequest {
  name: string
  description?: string
}

// 更新相册请求
export interface UpdateAlbumRequest {
  name?: string
  description?: string
  sort_order?: number
}
