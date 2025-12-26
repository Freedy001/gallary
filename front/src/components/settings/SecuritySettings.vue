<template>
  <div class="rounded-2xl bg-white/5 ring-1 ring-white/10 overflow-hidden">
    <div class="border-b border-white/5 p-5 bg-white/[0.02]">
      <h2 class="text-lg font-medium text-white">密码设置</h2>
      <p class="mt-1 text-sm text-gray-500">设置或修改管理员密码，启用认证保护</p>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="p-6 flex justify-center">
      <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
    </div>

    <div v-else class="p-6 space-y-5">
      <div v-if="passwordSet">
        <label class="block text-sm font-medium text-gray-300 mb-2">当前密码</label>
        <div class="relative">
          <input
              v-model="form.oldPassword"
              :type="showOldPassword ? 'text' : 'password'"
              class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 pr-12 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
              placeholder="请输入当前密码"
          />
          <button
              type="button"
              @click="showOldPassword = !showOldPassword"
              class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white transition-colors"
          >
            <EyeIcon v-if="!showOldPassword" class="h-5 w-5" />
            <EyeSlashIcon v-else class="h-5 w-5" />
          </button>
        </div>
      </div>
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-2">{{
            passwordSet ? '新密码' : '设置密码'
          }}</label>
        <div class="relative">
          <input
              v-model="form.newPassword"
              :type="showNewPassword ? 'text' : 'password'"
              class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 pr-12 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
              placeholder="请输入密码 (至少6位)"
          />
          <button
              type="button"
              @click="showNewPassword = !showNewPassword"
              class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white transition-colors"
          >
            <EyeIcon v-if="!showNewPassword" class="h-5 w-5" />
            <EyeSlashIcon v-else class="h-5 w-5" />
          </button>
        </div>
      </div>
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-2">确认密码</label>
        <div class="relative">
          <input
              v-model="form.confirmPassword"
              :type="showConfirmPassword ? 'text' : 'password'"
              class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 pr-12 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
              placeholder="请再次输入密码"
          />
          <button
              type="button"
              @click="showConfirmPassword = !showConfirmPassword"
              class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white transition-colors"
          >
            <EyeIcon v-if="!showConfirmPassword" class="h-5 w-5" />
            <EyeSlashIcon v-else class="h-5 w-5" />
          </button>
        </div>
      </div>
      <div class="pt-2 flex items-center gap-4">
        <button
            @click="handleUpdatePassword"
            :disabled="saving"
            class="px-6 py-2.5 rounded-lg bg-primary-500/20 text-primary-400 hover:bg-primary-500/30 ring-1 ring-primary-500/30 transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {{ saving ? '保存中...' : '保存密码' }}
        </button>

        <button
            v-if="isLoggedIn"
            @click="handleLogout"
            :disabled="loggingOut"
            class="px-6 py-2.5 rounded-lg bg-red-500/20 text-red-400 hover:bg-red-500/30 ring-1 ring-red-500/30 transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {{ loggingOut ? '退出中...' : '退出登录' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { settingsApi } from '@/api/settings'
import { useDialogStore } from '@/stores/dialog'
import { useAuthStore } from '@/stores/auth'
import { EyeIcon, EyeSlashIcon } from '@heroicons/vue/24/outline'

const router = useRouter()
const dialogStore = useDialogStore()
const authStore = useAuthStore()

const loading = ref(true)
const passwordSet = ref(false)
const saving = ref(false)
const loggingOut = ref(false)

// 密码可见性状态
const showOldPassword = ref(false)
const showNewPassword = ref(false)
const showConfirmPassword = ref(false)

const form = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
})

const isLoggedIn = computed(() => !!authStore.token)

async function loadSettings() {
  loading.value = true
  try {
    const resp = await settingsApi.getPasswordStatus()
    passwordSet.value = resp.data.is_set
  } catch (error) {
    console.error('Failed to load security settings:', error)
  } finally {
    loading.value = false
  }
}

async function handleUpdatePassword() {
  if (form.newPassword.length < 6) {
    dialogStore.alert({ title: '错误', message: '密码长度至少为6位', type: 'error' })
    return
  }
  if (form.newPassword !== form.confirmPassword) {
    dialogStore.alert({ title: '错误', message: '两次输入的密码不一致', type: 'error' })
    return
  }

  saving.value = true
  try {
    await settingsApi.updatePassword({
      old_password: form.oldPassword,
      new_password: form.newPassword,
    })
    dialogStore.alert({ title: '成功', message: '密码更新成功', type: 'success' })
    form.oldPassword = ''
    form.newPassword = ''
    form.confirmPassword = ''
    passwordSet.value = true
    await router.push('/login')
  } catch (error: any) {
    dialogStore.alert({ title: '错误', message: error.message || '更新密码失败', type: 'error' })
  } finally {
    saving.value = false
  }
}

async function handleLogout() {
  loggingOut.value = true
  try {
    const result = await dialogStore.confirm({
      title: '退出登录',
      message: '确定要退出登录吗？退出后需要重新输入密码才能访问系统。',
      type: 'warning'
    })

    if (!result) {
      return
    }

    authStore.logout()
    await router.push('/login')
  } catch (error: any) {
    dialogStore.alert({
      title: '错误',
      message: error.message || '退出登录失败',
      type: 'error'
    })
  } finally {
    loggingOut.value = false
  }
}

onMounted(() => {
  loadSettings()
})
</script>
