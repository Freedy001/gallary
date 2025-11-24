import type {Image} from './image'

export interface Share {
  id: number
  share_code: string
  title?: string
  description?: string
  password?: string
  expire_at?: string
  view_count: number
  download_count: number
  is_active: boolean
  created_at: string
  updated_at: string
  images?: Image[]
}

export interface CreateShareRequest {
  image_ids: number[]
  title?: string
  description?: string
  password?: string
  expire_days: number
}

export interface SharePublicInfo {
  title?: string
  description?: string
  has_password: boolean
  expire_at?: string
  created_at: string
  share_code: string
}