import http from './http'

// 存储统计信息
export interface StorageStats {
  used_bytes: number
  total_bytes: number
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
}