import { defineStore } from 'pinia'
import { ref } from 'vue'
import { authApi } from '@/api/auth'

export const useAuthStore = defineStore('auth', () => {
  // State
  const token = ref<string | null>(localStorage.getItem('auth_token'))
  const isAuthenticated = ref(false)
  const requiresAuth = ref(true) // 是否需要认证
  const hasCheckedAuth = ref(false) // 是否已检查过认证状态

  // Actions
  async function checkAuthStatus() {
    try {
      const response = await authApi.checkAuth()
      requiresAuth.value = response.data.requires_auth

      if (!requiresAuth.value) {
        // 不需要认证，直接设置为已认证
        isAuthenticated.value = true
      } else if (token.value) {
        // 需要认证且有token，验证token是否有效
        isAuthenticated.value = true
      } else {
        isAuthenticated.value = false
      }

      hasCheckedAuth.value = true
    } catch (error) {
      console.error('Failed to check auth status:', error)
      // 发生错误时假设需要认证
      requiresAuth.value = true
      isAuthenticated.value = false
      hasCheckedAuth.value = true
    }
  }

  async function login(password: string) {
    try {
      const response = await authApi.login({ password })
      token.value = response.data.token
      isAuthenticated.value = true
      localStorage.setItem('auth_token', response.data.token)
      return true
    } catch (error) {
      console.error('Login failed:', error)
      return false
    }
  }

  function logout() {
    token.value = null
    isAuthenticated.value = false
    localStorage.removeItem('auth_token')
  }

  return {
    // State
    token,
    isAuthenticated,
    requiresAuth,
    hasCheckedAuth,

    // Actions
    checkAuthStatus,
    login,
    logout,
  }
})
