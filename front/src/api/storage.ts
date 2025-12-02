import http from './http'

// 存储ID类型 - "local" | "aliyunpan:{userId}"
export type StorageId = string

// 单个存储提供者的统计信息
export interface ProviderStats {
  id: StorageId           // 存储ID，如 "local", "aliyunpan:123456"
  name: string
  used_bytes: number
  total_bytes: number
  is_active: boolean
}

// 多存储提供者统计信息
export interface StorageStats {
  providers: ProviderStats[]
}

// 阿里云盘二维码响应
export interface AliyunPanQRCodeResponse {
  qr_code_url: string
  status: string
  message: string
}

// 阿里云盘登录状态响应
export interface AliyunPanLoginResponse {
  status: 'NEW' | 'SCANED' | 'CONFIRMED' | 'EXPIRED'
  message: string
  refresh_token?: string
  user_id?: string        // 用户ID（用于构建 StorageId）
  user_name?: string
  nick_name?: string
  avatar?: string
}

// 解析存储ID
export function parseStorageId(id: StorageId): { driver: string; accountId?: string } {
  const parts = id.split(':')
  return {
    driver: parts[0],
    accountId: parts[1],
  }
}

// 获取存储驱动名称
export function getStorageDriverName(id: StorageId): string {
  const { driver } = parseStorageId(id)
  switch (driver) {
    case 'local':
      return '本地存储'
    case 'aliyunpan':
      return '阿里云盘'
    case 'oss':
      return '阿里云OSS'
    case 's3':
      return 'Amazon S3'
    case 'minio':
      return 'MinIO'
    default:
      return driver
  }
}

export const storageApi = {
  // 获取存储统计信息
  getStorageStats: () => http.get<StorageStats>('/api/storage/stats'),

  // 生成阿里云盘登录二维码
  generateAliyunPanQRCode: () =>
    http.post<AliyunPanQRCodeResponse>('/api/storage/aliyunpan/qrcode'),

  // 检查阿里云盘二维码扫描状态
  checkAliyunPanQRCodeStatus: () =>
    http.get<AliyunPanLoginResponse>('/api/storage/aliyunpan/qrcode/status'),

  // 退出阿里云盘登录
  logoutAliyunPan: () =>
    http.post<{ message: string }>('/api/storage/aliyunpan/logout'),
}
