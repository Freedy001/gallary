import http from './http'

// 阿里云盘用户信息
export interface AliyunPanUserInfo {
  is_logged_in: boolean
  nick_name?: string
  avatar?: string
}

// 存储配置类型
export interface StorageConfig {
  storage_default_type: 'local' | 'aliyunpan' | 'oss' | 's3' | 'minio'

  // 本地存储
  local_base_path?: string
  local_url_prefix?: string

  // 阿里云盘
  aliyunpan_refresh_token?: string
  aliyunpan_base_path?: string
  aliyunpan_drive_type?: 'file' | 'album' | 'resource'
  aliyunpan_download_chunk_size?: number   // 下载分片大小(KB)
  aliyunpan_download_concurrency?: number  // 下载并发数

  // 阿里云盘用户信息（只读，由后端返回）
  aliyunpan_user?: AliyunPanUserInfo

  // OSS
  oss_endpoint?: string
  oss_access_key_id?: string
  oss_access_key_secret?: string
  oss_bucket?: string
  oss_url_prefix?: string

  // S3
  s3_region?: string
  s3_access_key_id?: string
  s3_secret_access_key?: string
  s3_bucket?: string
  s3_url_prefix?: string

  // MinIO
  minio_endpoint?: string
  minio_access_key_id?: string
  minio_secret_access_key?: string
  minio_bucket?: string
  minio_use_ssl?: boolean
  minio_url_prefix?: string
}

// 清理配置类型
export interface CleanupConfig {
  trash_auto_delete_days: number
}

// 密码更新类型
export interface PasswordUpdateDTO {
  old_password?: string
  new_password: string
}

// 存储配置更新结果
export interface StorageUpdateResult {
  needs_migration: boolean
  task_id?: number
  message: string
}

// 设置 API
export const settingsApi = {
  // 获取所有设置
  getAll: () => http.get<Record<string, any>>('/api/settings'),

  // 按分类获取设置
  getByCategory: (category: string) =>
    http.get<Record<string, any>>(`/api/settings/${category}`),

  // 获取密码设置状态
  getPasswordStatus: () =>
    http.get<{ is_set: boolean }>('/api/settings/password/status'),

  // 更新密码
  updatePassword: (data: PasswordUpdateDTO) =>
    http.put<{ message: string }>('/api/settings/password', data),

  // 更新存储配置
  updateStorage: (config: StorageConfig) =>
    http.put<StorageUpdateResult>('/api/settings/storage', config),

  // 更新清理配置
  updateCleanup: (config: CleanupConfig) =>
    http.put<{ message: string }>('/api/settings/cleanup', config),
}