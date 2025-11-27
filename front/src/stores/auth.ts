import {defineStore} from 'pinia'
import {ref} from 'vue'
import {authApi} from '@/api/auth'

export const useAuthStore = defineStore('auth', () => {
  // State
  const token = ref<string | null>(localStorage.getItem('auth_token'))
  let checkAuthed = false;
  let authenticated = false; // 默认未认证

  // Actions
  async function requiresAuth(): Promise<boolean> {
    try {
      if (checkAuthed) {
        return !authenticated
      }

      const response = await authApi.checkAuth()
      checkAuthed = true
      authenticated = response.data.authenticated
      console.log('Auth check result:', authenticated)
    } catch (error) {
      console.error('Failed to check auth status:', error)
      // 发生错误时假设需要认证
      authenticated = false
    }

    return !authenticated
  }

  async function login(password: string) {
    try {
      const response = await authApi.login({password})
      token.value = response.data.token
      localStorage.setItem('auth_token', response.data.token)

      // 登录成功后更新认证状态
      checkAuthed = true
      authenticated = true

      console.log('Login successful, token:', response.data.token)
      return !!response.data.token
    } catch (error) {
      console.error('Login failed:', error)
      return false
    }
  }

  function logout() {
    token.value = null
    localStorage.removeItem('auth_token')

    // 登出时重置认证状态
    checkAuthed = false
    authenticated = false
  }

  return {
    // State
    token,
    requiresAuth,

    // Actions
    login,
    logout,
  }
})
