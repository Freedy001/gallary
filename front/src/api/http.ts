import axios, {type AxiosInstance, type AxiosRequestConfig, type AxiosResponse} from 'axios'
import type {ApiResponse} from '@/types'

class HttpClient {
  private instance: AxiosInstance

  constructor() {
    this.instance = axios.create({
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    })

    this.setupInterceptors()
  }

  private setupInterceptors() {
    // 请求拦截器
    this.instance.interceptors.request.use(
      (config) => {
        // 从 localStorage 获取 token
        const token = localStorage.getItem('auth_token')
        if (token) {
          config.headers.Authorization = `Bearer ${token}`
        }
        return config
      },
      (error) => {
        return Promise.reject(error)
      }
    )

    // 响应拦截器
    this.instance.interceptors.response.use(
      (response: AxiosResponse<ApiResponse>) => {
        const {code, message} = response.data

        // 成功响应
        if (!code) {
          return response
        }

        // 业务错误
        return Promise.reject(new Error(message || '请求失败'))
      },
      (error) => {
        // 网络错误或 HTTP 错误
        if (error.response) {
          const {status, data} = error.response

          // 401 未授权，清除 token 并跳转登录
          if (status === 401) {
            localStorage.removeItem('auth_token')
            window.location.href = '/login'
            return Promise.reject(new Error('未授权，请重新登录'))
          }

          // 其他 HTTP 错误
          return Promise.reject(new Error(data?.message || `请求失败: ${status}`))
        }

        // 请求超时或网络错误
        if (error.code === 'ECONNABORTED') {
          return Promise.reject(new Error('请求超时，请检查网络'))
        }

        return Promise.reject(new Error(error.message || '网络错误，请稍后重试'))
      }
    )
  }

  async get<T = any>(url: string, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return (await this.instance.get(url, config)).data
  }

  async post<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return (await this.instance.post(url, data, config)).data
  }

  async put<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return (await this.instance.put(url, data, config)).data
  }

  async delete<T = any>(url: string, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return (await this.instance.delete(url, config)).data
  }

  // 上传文件专用方法
  async upload<T = any>(url: string, formData: FormData, onProgress?: (progress: number) => void): Promise<ApiResponse<T>> {
    return (await this.instance.post(url, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      onUploadProgress: (progressEvent) => {
        if (progressEvent.total && onProgress) {
          const percentage = Math.round((progressEvent.loaded * 100) / progressEvent.total)
          onProgress(percentage)
        }
      },
    })).data
  }

  // 下载文件专用方法
  async download(url: string, filename?: string): Promise<void> {
    const response = await this.instance.get(url, {responseType: 'blob',});
    const blob = new Blob([response.data])
    const downloadUrl = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = downloadUrl
    link.download = filename || 'download'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(downloadUrl)
  }

  // 二进制上传方法（统一处理 presigned 和 backend 两种模式）
  async uploadBinary(
    url: string,
    data: Blob | File,
    contentType: string,
    options?: {
      skipAuth?: boolean  // true: 预签名上传（不带认证头），false: 后端代理（带认证头）
      onProgress?: (progress: number) => void
    }
  ): Promise<void> {
    const {skipAuth = false, onProgress} = options || {}

    if (skipAuth) {
      // 预签名上传：使用原生 XHR，不带认证头
      return new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest()
        xhr.open('PUT', url, true)
        xhr.setRequestHeader('Content-Type', contentType)

        xhr.upload.onprogress = (event) => {
          if (event.lengthComputable && onProgress) {
            onProgress(Math.round((event.loaded * 100) / event.total))
          }
        }

        xhr.onload = () => {
          if (xhr.status >= 200 && xhr.status < 300) {
            resolve()
          } else {
            reject(new Error(`上传失败: ${xhr.status} ${xhr.statusText}`))
          }
        }

        xhr.onerror = () => reject(new Error('网络错误'))
        xhr.send(data)
      })
    } else {
      // 后端代理：使用 axios，自动带认证头
      await this.instance.put(url, data, {
        headers: {'Content-Type': contentType},
        onUploadProgress: (e) => {
          if (e.total && onProgress) {
            onProgress(Math.round((e.loaded * 100) / e.total))
          }
        }
      })
    }
  }
}

export const http = new HttpClient()
export default http
