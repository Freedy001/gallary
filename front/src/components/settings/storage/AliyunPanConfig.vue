<template>
  <div class="space-y-4 pt-4 border-t border-white/5">
    <!-- 账号状态区域 -->
    <div class="rounded-xl bg-white/[0.03] border border-white/10 p-5">
      <!-- 已登录状态 -->
      <div v-if="userInfo?.is_logged_in" class="flex items-center justify-between">
        <div class="flex items-center gap-4">
          <img
              v-if="userInfo.avatar"
              :src="userInfo.avatar"
              class="w-12 h-12 rounded-full ring-2 ring-primary-500/30"
          />
          <div v-else class="w-12 h-12 rounded-full bg-primary-500/20 flex items-center justify-center">
            <UserIcon class="w-6 h-6 text-primary-400" />
          </div>
          <div>
            <h3 class="text-sm font-medium text-white">{{ userInfo.nick_name || '阿里云盘用户' }}</h3>
            <p class="text-xs text-green-400 mt-1 flex items-center gap-1">
              <CheckCircleIcon class="w-3.5 h-3.5" />
              已绑定
            </p>
          </div>
        </div>
        <button
            @click="handleLogout"
            :disabled="logoutLoading"
            class="px-4 py-2 rounded-lg bg-red-500/10 text-red-400 hover:bg-red-500/20 ring-1 ring-red-500/20 transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed text-sm"
        >
          {{ logoutLoading ? '退出中...' : '退出登录' }}
        </button>
      </div>

      <!-- 未登录状态 -->
      <template v-else>
        <div class="flex items-center justify-between mb-4">
          <div>
            <h3 class="text-sm font-medium text-white">账号绑定</h3>
            <p class="text-xs text-gray-500 mt-1">扫描二维码登录阿里云盘账号</p>
          </div>
          <div v-if="tempUser.nickName" class="flex items-center gap-2">
            <img v-if="tempUser.avatar" :src="tempUser.avatar" class="w-8 h-8 rounded-full"/>
            <span class="text-sm text-primary-400">{{ tempUser.nickName }}</span>
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
      </template>
    </div>

    <!-- 基本配置项 -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
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

    <!-- 下载配置 -->
    <div class="rounded-xl bg-white/[0.03] border border-white/10 p-5">
      <h3 class="text-sm font-medium text-white mb-4">下载配置</h3>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <!-- 分片大小 -->
        <div>
          <div class="flex items-center justify-between mb-2">
            <label class="text-sm font-medium text-gray-300">分片大小</label>
            <span class="text-sm text-primary-400">{{ downloadChunkSize }} KB</span>
          </div>
          <input
              type="range"
              :value="downloadChunkSize"
              @input="$emit('update:downloadChunkSize', Number(($event.target as HTMLInputElement).value))"
              min="128"
              max="4096"
              step="128"
              class="w-full cursor-pointer"
          />
          <div class="flex justify-between text-xs text-gray-500 mt-1">
            <span>128 KB</span>
            <span>4096 KB</span>
          </div>
        </div>
        <!-- 并发数 -->
        <div>
          <div class="flex items-center justify-between mb-2">
            <label class="text-sm font-medium text-gray-300">下载并发数</label>
            <span class="text-sm text-primary-400">{{ downloadConcurrency }}</span>
          </div>
          <input
              type="range"
              :value="downloadConcurrency"
              @input="$emit('update:downloadConcurrency', Number(($event.target as HTMLInputElement).value))"
              min="1"
              max="16"
              step="1"
              class="w-full cursor-pointer"
          />
          <div class="flex justify-between text-xs text-gray-500 mt-1">
            <span>1</span>
            <span>16</span>
          </div>
        </div>
      </div>
      <p class="text-xs text-gray-500 mt-3">
        较大的分片和更多并发可提高下载速度，但会占用更多内存
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onUnmounted } from 'vue'
import { storageApi } from '@/api/storage'
import { useDialogStore } from '@/stores/dialog'
import { UserIcon, CheckCircleIcon } from '@heroicons/vue/24/outline'
import type { AliyunPanUserInfo } from '@/api/settings'

const props = defineProps<{
  refreshToken?: string
  basePath?: string
  driveType?: string
  downloadChunkSize?: number
  downloadConcurrency?: number
  userInfo?: AliyunPanUserInfo
}>()

const emit = defineEmits<{
  (e: 'update:refreshToken', value: string): void
  (e: 'update:basePath', value: string): void
  (e: 'update:driveType', value: string): void
  (e: 'update:downloadChunkSize', value: number): void
  (e: 'update:downloadConcurrency', value: number): void
  (e: 'logout'): void
}>()

const dialogStore = useDialogStore()

const loading = ref(false)
const logoutLoading = ref(false)

const qrCode = reactive({
  url: '',
  status: '' as '' | 'NEW' | 'SCANED' | 'CONFIRMED' | 'EXPIRED',
  message: '',
})

// 扫码过程中临时显示的用户信息
const tempUser = reactive({
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
        tempUser.nickName = resp.data.nick_name || ''
        tempUser.userName = resp.data.user_name || ''
        tempUser.avatar = resp.data.avatar || ''

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

async function handleLogout() {
  const confirmed = await dialogStore.confirm({
    title: '退出登录',
    message: '确定要退出阿里云盘账号吗？退出后需要重新扫码登录。',
    type: 'warning'
  })

  if (!confirmed) return

  logoutLoading.value = true
  try {
    await storageApi.logoutAliyunPan()
    emit('update:refreshToken', '')
    emit('logout')
    await dialogStore.alert({
      title: '成功',
      message: '已退出阿里云盘账号',
      type: 'success'
    })
  } catch (error: any) {
    await dialogStore.alert({
      title: '错误',
      message: error.message || '退出登录失败',
      type: 'error'
    })
  } finally {
    logoutLoading.value = false
  }
}

onUnmounted(() => {
  stopPolling()
})
</script>

<style scoped>
/* 滑块轨道样式 */
input[type="range"] {
  -webkit-appearance: none;
  appearance: none;
  width: 100%;
  height: 8px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 4px;
  outline: none;
}

/* WebKit 浏览器的滑块样式 */
input[type="range"]::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: #8b5cf6;
  cursor: pointer;
  border: 2px solid #a78bfa;
  box-shadow: 0 0 10px rgba(139, 92, 246, 0.5);
  transition: all 0.15s ease;
}

input[type="range"]::-webkit-slider-thumb:hover {
  transform: scale(1.1);
  box-shadow: 0 0 15px rgba(139, 92, 246, 0.7);
}

/* Firefox 浏览器的滑块样式 */
input[type="range"]::-moz-range-thumb {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: #8b5cf6;
  cursor: pointer;
  border: 2px solid #a78bfa;
  box-shadow: 0 0 10px rgba(139, 92, 246, 0.5);
  transition: all 0.15s ease;
}

input[type="range"]::-moz-range-thumb:hover {
  transform: scale(1.1);
  box-shadow: 0 0 15px rgba(139, 92, 246, 0.7);
}

/* Firefox 轨道样式 */
input[type="range"]::-moz-range-track {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 4px;
  height: 8px;
}
</style>
