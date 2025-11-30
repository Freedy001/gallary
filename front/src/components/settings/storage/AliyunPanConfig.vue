<template>
  <div class="space-y-4 pt-4 border-t border-white/5">
    <!-- 扫码登录区域 -->
    <div class="rounded-xl bg-white/[0.03] border border-white/10 p-5">
      <div class="flex items-center justify-between mb-4">
        <div>
          <h3 class="text-sm font-medium text-white">账号绑定</h3>
          <p class="text-xs text-gray-500 mt-1">扫描二维码登录阿里云盘账号</p>
        </div>
        <div v-if="user.nickName" class="flex items-center gap-2">
          <img v-if="user.avatar" :src="user.avatar" class="w-8 h-8 rounded-full"/>
          <span class="text-sm text-primary-400">{{ user.nickName }}</span>
        </div>
      </div>

      <!-- 二维码显示区域 -->
      <div v-if="qrCode.url" class="flex flex-col items-center py-4">
        <div class="p-4 bg-white rounded-xl mb-4">
          <img :src="generateQRCodeDataUrl(qrCode.url)" alt="QRCode" class="w-48 h-48"/>
        </div>
        <p class="text-sm text-gray-400 mb-2">{{ qrCode.message }}</p>
        <div v-if="qrCode.status === 'SCANED'" class="flex items-center gap-2 text-yellow-400">
          <svg class="w-4 h-4 animate-pulse" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/>
          </svg>
          <span class="text-sm">请在手机上确认登录</span>
        </div>
      </div>

      <!-- 操作按钮 -->
      <div class="flex justify-center gap-3">
        <button
            v-if="!qrCode.url || qrCode.status === 'EXPIRED'"
            @click="generateQRCode"
            :disabled="loading"
            class="px-5 py-2.5 rounded-lg bg-blue-500/20 text-blue-400 hover:bg-blue-500/30 ring-1 ring-blue-500/30 transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {{ loading ? '生成中...' : (qrCode.status === 'EXPIRED' ? '刷新二维码' : '生成登录二维码') }}
        </button>
        <button
            v-if="qrCode.url && qrCode.status !== 'EXPIRED' && qrCode.status !== 'CONFIRMED'"
            @click="cancelLogin"
            class="px-5 py-2.5 rounded-lg bg-white/5 text-gray-400 hover:bg-white/10 ring-1 ring-white/10 transition-all duration-300"
        >
          取消
        </button>
      </div>
    </div>

    <!-- 配置项 -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-2">Refresh Token</label>
        <div class="relative">
          <input
              :value="refreshToken"
              @input="$emit('update:refreshToken', ($event.target as HTMLInputElement).value)"
              :type="showRefreshToken ? 'text' : 'password'"
              class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 pr-12 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
              placeholder="扫码登录后自动填入"
          />
          <button
              type="button"
              @click="showRefreshToken = !showRefreshToken"
              class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white transition-colors"
          >
            <EyeIcon v-if="!showRefreshToken" class="h-5 w-5" />
            <EyeSlashIcon v-else class="h-5 w-5" />
          </button>
        </div>
      </div>
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-2">网盘类型</label>
        <select
            :value="driveType"
            @change="$emit('update:driveType', ($event.target as HTMLSelectElement).value)"
            class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
        >
          <option value="file">备份盘</option>
          <option value="album">相册</option>
          <option value="resource">资源库</option>
        </select>
      </div>
    </div>
    <div>
      <label class="block text-sm font-medium text-gray-300 mb-2">存储路径</label>
      <input
          :value="basePath"
          @input="$emit('update:basePath', ($event.target as HTMLInputElement).value)"
          type="text"
          class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
          placeholder="例如: /gallery/images"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onUnmounted } from 'vue'
import { storageApi } from '@/api/storage'
import { useDialogStore } from '@/stores/dialog'
import { EyeIcon, EyeSlashIcon } from '@heroicons/vue/24/outline'

defineProps<{
  refreshToken?: string
  basePath?: string
  driveType?: string
}>()

const emit = defineEmits<{
  (e: 'update:refreshToken', value: string): void
  (e: 'update:basePath', value: string): void
  (e: 'update:driveType', value: string): void
}>()

const dialogStore = useDialogStore()

const loading = ref(false)
const showRefreshToken = ref(false)

const qrCode = reactive({
  url: '',
  status: '' as '' | 'NEW' | 'SCANED' | 'CONFIRMED' | 'EXPIRED',
  message: '',
})
const user = reactive({
  nickName: '',
  userName: '',
  avatar: '',
})
let pollingTimer: ReturnType<typeof setInterval> | null = null

function generateQRCodeDataUrl(text: string): string {
  return `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(text)}`
}

async function generateQRCode() {
  loading.value = true
  try {
    const resp = await storageApi.generateAliyunPanQRCode()
    qrCode.url = resp.data.qr_code_url
    qrCode.status = resp.data.status as any
    qrCode.message = resp.data.message
    startPolling()
  } catch (error: any) {
    await dialogStore.alert({
      title: '错误',
      message: error.message || '生成二维码失败',
      type: 'error'
    })
  } finally {
    loading.value = false
  }
}

function startPolling() {
  stopPolling()
  pollingTimer = setInterval(async () => {
    try {
      const resp = await storageApi.checkAliyunPanQRCodeStatus()
      qrCode.status = resp.data.status
      qrCode.message = resp.data.message

      if (resp.data.status === 'CONFIRMED') {
        stopPolling()
        user.nickName = resp.data.nick_name || ''
        user.userName = resp.data.user_name || ''
        user.avatar = resp.data.avatar || ''

        if (resp.data.refresh_token) {
          emit('update:refreshToken', resp.data.refresh_token)
        }

        await dialogStore.alert({
          title: '成功',
          message: `阿里云盘账号 ${resp.data.nick_name} 登录成功！`,
          type: 'success'
        })

        qrCode.url = ''
      } else if (resp.data.status === 'EXPIRED') {
        stopPolling()
      }
    } catch (error) {
      console.error('检查二维码状态失败:', error)
    }
  }, 2000)
}

function stopPolling() {
  if (pollingTimer) {
    clearInterval(pollingTimer)
    pollingTimer = null
  }
}

function cancelLogin() {
  stopPolling()
  qrCode.url = ''
  qrCode.status = ''
  qrCode.message = ''
}

onUnmounted(() => {
  stopPolling()
})
</script>
