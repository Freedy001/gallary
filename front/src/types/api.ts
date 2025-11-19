export interface ApiResponse<T = any> {
  code: ErrorCode
  message: string
  data?: T
}

export interface LoginRequest {
  password: string
}

export interface LoginResponse {
  token: string
  expires_in: number
}

export interface CheckAuthResponse {
  authenticated: boolean
}

export enum ErrorCode {
  SUCCESS = 0,
  BAD_REQUEST = 400,
  UNAUTHORIZED = 401,
  FORBIDDEN = 403,
  NOT_FOUND = 404,
  SERVER_ERROR = 500
}
