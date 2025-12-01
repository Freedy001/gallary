import http from './http'

// 单个存储提供者的统计信息
export interface ProviderStats {
  type: 'local' | 'aliyunpan' | 'oss' | 's3' | 'minio'
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
  user_name?: string
  nick_name?: string
  avatar?: string
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