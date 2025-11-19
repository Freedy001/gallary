import http from './http'
import type { ApiResponse, LoginRequest, LoginResponse, CheckAuthResponse } from '@/types'

export const authApi = {
  // 登录
  login(data: LoginRequest): Promise<ApiResponse<LoginResponse>> {
    return http.post('/api/auth/login', data)
  },

  // 检查认证状态
  checkAuth(): Promise<ApiResponse<CheckAuthResponse>> {
    return http.get('/api/auth/check')
  },

  // 健康检查
  healthCheck(): Promise<any> {
    return http.get('/health')
  },
}
