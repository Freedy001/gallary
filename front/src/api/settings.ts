import http from './http'
import type { StorageId } from './storage'

// 阿里云盘用户信息
export interface AliyunPanUserInfo {
  is_logged_in: boolean
  nick_name?: string
  avatar?: string
  user_id?: string  // 用户ID（用于构建 StorageId）
}

// 本地存储配置
export interface LocalStorageConfig {
  id: StorageId
  base_path: string
}

// 阿里云盘存储配置（单个账号）
export interface AliyunPanStorageConfig {
  id: StorageId
  refresh_token?: string
  base_path?: string
  drive_type?: 'file' | 'album' | 'resource'
}

// 阿里云盘全局配置（所有账号共享）
export interface AliyunPanGlobalConfig {
  download_chunk_size: number    // 下载分片大小 (KB)
  download_concurrency: number   // 下载并发数
}

// 完整存储配置（后端返回格式）
export interface StorageConfigPO {
  storageId: StorageId                        // 默认存储ID
  localConfig?: LocalStorageConfig            // 本地存储配置
  aliyunpanConfig?: AliyunPanStorageConfig[]  // 阿里云盘账号配置数组
  aliyunpanGlobal?: AliyunPanGlobalConfig     // 阿里云盘全局配置
  aliyunpan_user?: AliyunPanUserInfo[]        // 用户信息（只读）
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

// 添加存储请求
export interface AddStorageRequest {
  type: 'aliyunpan'  // 目前只支持添加阿里云盘
  config: Omit<AliyunPanStorageConfig, 'id'> & { id: StorageId }
}

// 设置默认存储请求
export interface SetDefaultStorageRequest {
  storageId: StorageId
}

// 旧的扁平化存储配置类型（保留用于兼容，已废弃）
/** @deprecated 请使用 StorageConfigPO */
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

  // OSS (未实现)
  oss_endpoint?: string
  oss_access_key_id?: string
  oss_access_key_secret?: string
  oss_bucket?: string
  oss_url_prefix?: string

  // S3 (未实现)
  s3_region?: string
  s3_access_key_id?: string
  s3_secret_access_key?: string
  s3_bucket?: string
  s3_url_prefix?: string

  // MinIO (未实现)
  minio_endpoint?: string
  minio_access_key_id?: string
  minio_secret_access_key?: string
  minio_bucket?: string
  minio_use_ssl?: boolean
  minio_url_prefix?: string
}

// 设置 API
export const settingsApi = {
  // 获取所有设置
  getAll: () => http.get<Record<string, any>>('/api/settings'),

  // 按分类获取设置
  getByCategory: (category: string) =>
    http.get<Record<string, any>>(`/api/settings/${category}`),

  // 获取存储配置
  getStorageConfig: () =>
    http.get<StorageConfigPO>('/api/settings/storage'),

  // 获取密码设置状态
  getPasswordStatus: () =>
    http.get<{ is_set: boolean }>('/api/settings/password/status'),

  // 更新密码
  updatePassword: (data: PasswordUpdateDTO) =>
    http.put<{ message: string }>('/api/settings/password', data),

  // 添加新存储配置 (POST)
  addStorage: (config: AddStorageRequest) =>
    http.post<StorageUpdateResult>('/api/settings/storage', config),

  // 修改存储配置 (PUT)
  updateStorage: (storageId: StorageId, config: LocalStorageConfig | AliyunPanStorageConfig) =>
    http.put<StorageUpdateResult>(`/api/settings/storage/${encodeURIComponent(storageId)}`, config),

  // 删除存储配置 (DELETE)
  deleteStorage: (storageId: StorageId) =>
    http.delete<{ message: string }>(`/api/settings/storage/${encodeURIComponent(storageId)}`),

  // 设置默认存储
  setDefaultStorage: (storageId: StorageId) =>
    http.put<{ message: string }>('/api/settings/storage/default', { storageId }),

  // 更新阿里云盘全局配置
  updateGlobalConfig: (config: AliyunPanGlobalConfig) =>
    http.put<{ message: string }>('/api/settings/storage/global', config),

  // 更新清理配置
  updateCleanup: (config: CleanupConfig) =>
    http.put<{ message: string }>('/api/settings/cleanup', config),
}
