export interface Tag {
  id: number
  name: string
  color: string
}

export interface Metadata {
  id: number
  meta_key: string
  meta_value: string
  value_type: string
}

export interface Image {
  id: number
  uuid: string
  original_name: string
  storage_path: string
  storage_type: string
  file_size: number
  file_hash: string
  thumbnail_path: string
  thumbnail_width: number
  thumbnail_height: number
  mime_type: string
  width: number
  height: number
  // URL 字段（由后端生成）
  url: string                    // 原图访问URL
  thumbnail_url?: string         // 缩略图访问URL
  taken_at: string | null
  latitude: number | null
  longitude: number | null
  location_name: string | null
  camera_model: string | null
  camera_make: string | null
  aperture: string | null
  shutter_speed: string | null
  iso: number | null
  focal_length: string | null
  ai_score: number | null
  tags: Tag[]
  metadata: Metadata[]
  created_at: string
  updated_at: string
  deleted_at?: string | null
}

export interface SearchParams {
  keyword?: string
  start_date?: string
  end_date?: string
  location?: string
  tags?: number[]  // 标签ID数组
  model_id?: string
  page?: number
  page_size?: number
  // 经纬度搜索
  latitude?: number
  longitude?: number
  radius?: number // 搜索半径（公里），默认 10km
  // 以图搜图
  file?: File
}

export interface MetadataUpdate {
  key: string
  value?: string
  value_type?: string
}

export interface UpdateMetadataRequest {
  image_ids: number[]
  original_name?: string
  taken_at?: string
  location_name?: string
  latitude?: number
  longitude?: number
  metadata?: MetadataUpdate[]
  tags?: string[]
}

export interface ClusterResult {
  min_lat: number
  max_lat: number
  min_lng: number
  max_lng: number
  latitude: number
  longitude: number
  count: number
  cover_image: Image
}

export interface GeoBounds {
  min_lat: number
  max_lat: number
  min_lng: number
  max_lng: number
  count: number
}
