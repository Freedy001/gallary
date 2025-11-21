<template>
  <div class="flex min-h-screen items-center justify-center bg-gray-50 px-4 py-12 sm:px-6 lg:px-8">
    <div class="w-full max-w-md space-y-8">
      <!-- Logo 和标题 -->
      <div class="text-center">
        <h2 class="text-3xl font-bold tracking-tight text-gray-900">影像库</h2>
        <p class="mt-2 text-sm text-gray-600">登录以管理您的图片</p>
      </div>

      <!-- 登录表单 -->
      <div class="mt-8 rounded-lg bg-white p-8 shadow-md">
        <form @submit.prevent="handleSubmit" class="space-y-6">
          <!-- 密码输入 -->
          <div>
            <Input
              v-model="password"
              type="password"
              label="密码"
              placeholder="请输入密码"
              required
              :error="error"
            />
          </div>

          <!-- 提交按钮 -->
          <Button
            type="submit"
            variant="primary"
            size="lg"
            :loading="loading"
            full-width
          >
            登录
          </Button>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import Button from '@/components/common/Button.vue'
import Input from '@/components/common/Input.vue'

const router = useRouter()
const authStore = useAuthStore()

const password = ref('')
const loading = ref(false)
const error = ref('')

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
      error.value = '密码错误,请重试'
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : '登录失败'
  } finally {
    loading.value = false
  }
}
</script>
