import http from './http'
import type {StorageId} from './storage'

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

// S3 兼容存储服务商
export type S3Provider = 'aws' | 'minio' | 'aliyun-oss' | 'qiniu' | 'tencent-cos' | 'other'

// S3 兼容存储配置（单个账号）
export interface S3StorageConfig {
  id: StorageId
  name: string                // 账号显示名称
  provider: S3Provider        // 服务商类型
  endpoint: string            // S3 端点 (如 s3.amazonaws.com)
  region: string              // 区域 (如 us-east-1)
  bucket: string              // 桶名称
  access_key_id: string       // Access Key ID
  secret_access_key: string   // Secret Access Key
  base_path?: string          // 存储基础路径前缀
  use_ssl?: boolean           // 是否使用 HTTPS (默认 true)
  force_path_style?: boolean  // 使用路径风格 URL
  url_prefix?: string         // 自定义访问 URL 前缀 (CDN)
  proxy_url?: string          // HTTP 代理地址 (如 http://127.0.0.1:8080)
}

// 服务商预设配置
export const S3_PROVIDER_PRESETS: Record<S3Provider, { name: string; endpoint: string; region: string; forcePathStyle: boolean }> = {
  'aws': { name: 'AWS S3', endpoint: 's3.{region}.amazonaws.com', region: 'us-east-1', forcePathStyle: false },
  'minio': { name: 'MinIO', endpoint: 'localhost:9000', region: 'us-east-1', forcePathStyle: true },
  'aliyun-oss': { name: '阿里云 OSS', endpoint: 'oss-{region}.aliyuncs.com', region: 'cn-hangzhou', forcePathStyle: false },
  'qiniu': { name: '七牛云', endpoint: 's3-{region}.qiniucs.com', region: 'cn-east-1', forcePathStyle: false },
  'tencent-cos': { name: '腾讯云 COS', endpoint: 'cos.{region}.myqcloud.com', region: 'ap-guangzhou', forcePathStyle: false },
  'other': { name: '其他', endpoint: '', region: '', forcePathStyle: false },
}

// 完整存储配置（后端返回格式）
export interface StorageConfigPO {
  storageId: StorageId                        // 默认存储ID
  thumbnailStorageId?: StorageId              // 缩略图默认存储ID
  localConfig?: LocalStorageConfig            // 本地存储配置
  aliyunpanConfig?: AliyunPanStorageConfig[]  // 阿里云盘账号配置数组
  aliyunpanGlobal?: AliyunPanGlobalConfig     // 阿里云盘全局配置
  aliyunpan_user?: AliyunPanUserInfo[]        // 用户信息（只读）
  s3Config?: S3StorageConfig[]                // S3 兼容存储配置数组
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
  type: 'aliyunpan' | 's3'
  config: (Omit<AliyunPanStorageConfig, 'id'> | Omit<S3StorageConfig, 'id'>) & { id: StorageId }
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
  // 按分类获取设置
  getByCategory: (category: string) =>
    http.get<Record<string, any>>(`/api/settings/${category}`),

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

  // 设置缩略图默认存储
  setThumbnailDefaultStorage: (storageId: StorageId) =>
    http.put<{ message: string }>('/api/settings/storage/thumbnail/default', { storageId }),

  // 更新阿里云盘全局配置
  updateGlobalConfig: (config: AliyunPanGlobalConfig) =>
    http.put<{ message: string }>('/api/settings/storage/alyunpan/global', config),

  // 测试 S3 连接
  testS3Connection: (config: Omit<S3StorageConfig, 'id'> & { id?: StorageId }) =>
    http.post<{ message: string; bucket: string; region: string }>('/api/settings/storage/s3/test', config),

  // 更新清理配置
  updateCleanup: (config: CleanupConfig) =>
    http.put<{ message: string }>('/api/settings/cleanup', config),
}
