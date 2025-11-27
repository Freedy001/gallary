<template>
  <div class="relative min-h-screen overflow-hidden bg-black">
    <!-- 动态背景 -->
    <div class="absolute inset-0 bg-gradient-to-br from-black via-violet-950/20 to-black">
      <!-- 网格背景 -->
      <div class="absolute inset-0 bg-[linear-gradient(to_right,#80808012_1px,transparent_1px),linear-gradient(to_bottom,#80808012_1px,transparent_1px)] bg-[size:24px_24px]"></div>

      <!-- 极光效果 -->
      <div class="absolute top-0 left-1/4 w-96 h-96 bg-violet-600/10 rounded-full blur-[120px] animate-pulse"></div>
      <div class="absolute bottom-0 right-1/4 w-96 h-96 bg-cyan-600/10 rounded-full blur-[120px] animate-pulse" style="animation-delay: 2s"></div>

      <!-- 流星效果 -->
      <div class="absolute top-20 left-10 w-1 h-1 bg-violet-400 rounded-full animate-ping"></div>
      <div class="absolute top-40 right-20 w-1 h-1 bg-cyan-400 rounded-full animate-ping" style="animation-delay: 1s"></div>
      <div class="absolute bottom-32 left-1/3 w-1 h-1 bg-violet-400 rounded-full animate-ping" style="animation-delay: 3s"></div>
    </div>

    <!-- 主要内容 -->
    <div class="relative z-10 flex min-h-screen items-center justify-center px-4 py-12 sm:px-6 lg:px-8">
      <div class="w-full max-w-md">
        <!-- Logo 和标题 -->
        <div class="text-center mb-8">
          <!-- 装饰性图标 -->
          <div class="mx-auto mb-6 flex h-20 w-20 items-center justify-center rounded-2xl bg-violet-500/10 backdrop-blur-md border border-violet-500/20 shadow-[0_0_30px_rgba(139,92,246,0.3)]">
            <svg class="h-10 w-10 text-violet-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
          </div>

          <h1 class="text-4xl font-bold tracking-tight text-transparent bg-clip-text bg-gradient-to-r from-violet-400 to-cyan-400 mb-2">
            影像库
          </h1>
          <p class="text-violet-300/60 text-sm font-light">
            登录以管理您的图片
          </p>
        </div>

        <!-- 登录卡片 -->
        <div class="relative">
          <!-- 卡片光效 -->
          <div class="absolute inset-0 bg-gradient-to-r from-violet-500/10 to-cyan-500/10 rounded-3xl blur-xl"></div>

          <div class="relative rounded-3xl border border-white/10 bg-white/5 backdrop-blur-xl p-8 shadow-[0_8px_32px_rgba(0,0,0,0.3)]">
            <form @submit.prevent="handleSubmit" class="space-y-8">
              <!-- 密码输入 -->
              <div>
                <label class="block text-sm font-medium text-violet-300/80 mb-3">
                  密码
                </label>
                <div class="relative">
                  <input
                    v-model="password"
                    :type="showPassword ? 'text' : 'password'"
                    placeholder="请输入密码"
                    :class="[
                      'w-full rounded-2xl border bg-white/5 px-6 py-4 text-white placeholder-violet-300/30 outline-none transition-all duration-300 backdrop-blur-sm',
                      error
                        ? 'border-red-500/30 bg-red-500/5 focus:border-red-400/50 focus:bg-red-500/10'
                        : 'border-white/10 bg-white/[0.02] focus:border-violet-400/50 focus:bg-white/10'
                    ]"
                    @focus="handleFocus"
                    @blur="handleBlur"
                  />
                  <button
                    type="button"
                    @click="showPassword = !showPassword"
                    class="absolute right-4 top-1/2 -translate-y-1/2 text-violet-300/50 hover:text-violet-300/80 transition-colors"
                  >
                    <EyeIcon v-if="!showPassword" class="h-5 w-5" />
                    <EyeSlashIcon v-else class="h-5 w-5" />
                  </button>
                </div>
                <p v-if="error" class="mt-3 text-sm text-red-400/80 flex items-center gap-2">
                  <ExclamationTriangleIcon class="h-4 w-4" />
                  {{ error }}
                </p>
              </div>

              <!-- 提交按钮 -->
              <button
                type="submit"
                :disabled="loading || !password"
                class="relative w-full overflow-hidden rounded-2xl bg-gradient-to-r from-violet-600 to-cyan-600 px-6 py-4 text-center font-medium text-white shadow-[0_4px_20px_rgba(139,92,246,0.4)] transition-all duration-300 hover:scale-[1.02] hover:shadow-[0_6px_30px_rgba(139,92,246,0.6)] disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100 disabled:hover:shadow-none"
              >
                <span v-if="loading" class="flex items-center justify-center gap-2">
                  <svg class="animate-spin h-5 w-5" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none"/>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
                  </svg>
                  登录中...
                </span>
                <span v-else>登录</span>

                <!-- 按钮光效 -->
                <div class="absolute inset-0 bg-gradient-to-r from-violet-400 to-cyan-400 opacity-0 transition-opacity duration-300 hover:opacity-20"></div>
              </button>
            </form>
          </div>
        </div>

        <!-- 底部装饰 -->
        <div class="mt-12 text-center">
          <p class="text-xs text-violet-300/30 font-light">
            安全 · 高效 · 智能
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useDialogStore } from '@/stores/dialog'
import {
  EyeIcon,
  EyeSlashIcon,
  ExclamationTriangleIcon
} from '@heroicons/vue/24/outline'

const router = useRouter()
const authStore = useAuthStore()
const dialogStore = useDialogStore()

const password = ref('')
const loading = ref(false)
const error = ref('')
const showPassword = ref(false)

function handleFocus() {
  error.value = ''
}

function handleBlur() {
  // 可以在这里添加失去焦点时的逻辑
}

async function handleSubmit() {
  error.value = ''

  if (!password.value) {
    error.value = '请输入密码'
    return
  }

  loading.value = true

  try {
    const success = await authStore.login(password.value)
    if (success) {
      await router.push('/gallery')
    } else {
      error.value = '密码错误，请重试'
      // 震动效果
      const inputElement = document.querySelector('input[type="password"]') as HTMLElement
      if (inputElement) {
        inputElement.classList.add('animate-pulse')
        setTimeout(() => {
          inputElement.classList.remove('animate-pulse')
        }, 1000)
      }
    }
  } catch (err) {
    const errorMessage = err instanceof Error ? err.message : '登录失败'
    error.value = errorMessage

    // 显示错误对话框
    await dialogStore.alert({
      title: '登录失败',
      message: errorMessage,
      type: 'error'
    })
  } finally {
    loading.value = false
  }
}
</script>
