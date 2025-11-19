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
  tags: Tag[]
  metadata: Metadata[]
  created_at: string
  updated_at: string
}

export interface SearchParams {
  keyword?: string
  start_date?: string
  end_date?: string
  location?: string
  camera_model?: string
  page?: number
  page_size?: number
}
