<template>
  <div class="space-y-4 pt-4 border-t border-white/5">
    <!-- 账号列表 -->
    <div class="space-y-3">
      <div class="flex items-center justify-between">
        <h3 class="text-sm font-medium text-gray-300">阿里云盘账号</h3>
        <button
            @click="showAddAccount = true"
            class="px-3 py-1.5 rounded-lg bg-blue-500/20 text-blue-400 hover:bg-blue-500/30 ring-1 ring-blue-500/30 transition-all duration-300 text-xs"
        >
          + 添加账号
        </button>
      </div>

      <!-- 已绑定的账号列表 -->
      <div v-if="accounts.length > 0" class="space-y-2">
        <div
            v-for="(account, index) in accounts"
            :key="account.id"
            class="rounded-xl bg-white/[0.03] border border-white/10 p-4"
        >
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-3">
              <!-- 用户头像 -->
              <div class="w-10 h-10 rounded-full bg-primary-500/20 flex items-center justify-center">
                <UserIcon class="w-5 h-5 text-primary-400" />
              </div>
              <div>
                <h4 class="text-sm font-medium text-white flex items-center gap-2">
                  {{ getUserInfo(account.id)?.nick_name || '阿里云盘用户' }}
                  <span
                      v-if="defaultStorageId === account.id"
                      class="px-2 py-0.5 text-[10px] font-medium rounded-full bg-green-500/20 text-green-400 border border-green-500/30"
                  >
                    默认
                  </span>
                </h4>
                <p class="text-xs text-gray-500 mt-0.5">{{ account.id }}</p>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <button
                  v-if="defaultStorageId !== account.id"
                  @click="emit('setDefault', account.id)"
                  class="px-3 py-1.5 rounded-lg bg-white/5 text-gray-400 hover:bg-white/10 hover:text-white ring-1 ring-white/10 transition-all duration-300 text-xs"
              >
                设为默认
              </button>
              <button
                  @click="handleEditAccount(index)"
                  class="px-3 py-1.5 rounded-lg bg-white/5 text-gray-400 hover:bg-white/10 hover:text-white ring-1 ring-white/10 transition-all duration-300 text-xs"
              >
                编辑
              </button>
              <button
                  @click="handleDeleteAccount(account.id)"
                  :disabled="defaultStorageId === account.id"
                  class="px-3 py-1.5 rounded-lg bg-red-500/10 text-red-400 hover:bg-red-500/20 ring-1 ring-red-500/20 transition-all duration-300 text-xs disabled:opacity-50 disabled:cursor-not-allowed"
              >
                删除
              </button>
            </div>
          </div>

          <!-- 展开的账号配置 -->
          <div v-if="editingIndex === index" class="mt-4 pt-4 border-t border-white/5 space-y-4">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label class="block text-sm font-medium text-gray-300 mb-2">网盘类型</label>
                <BaseSelect
                    v-model="account.drive_type"
                    :options="driveTypeOptions"
                    @update:modelValue="handleAccountChange(index)"
                    placeholder="选择网盘类型"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-300 mb-2">存储路径</label>
                <input
                    v-model="account.base_path"
                    @change="handleAccountChange(index)"
                    type="text"
                    class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                    placeholder="例如: /gallery/images"
                />
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 无账号提示 -->
      <div v-else class="text-center py-8 text-gray-500">
        <p class="text-sm">暂无绑定的阿里云盘账号</p>
        <p class="text-xs mt-1">点击上方「添加账号」按钮扫码登录</p>
      </div>
    </div>

    <!-- 全局下载配置 -->
    <div v-if="globalConfig" class="rounded-xl bg-white/[0.03] border border-white/10 p-5">
      <h3 class="text-sm font-medium text-white mb-4">下载配置（全局）</h3>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <!-- 分片大小 -->
        <div>
          <div class="flex items-center justify-between mb-2">
            <label class="text-sm font-medium text-gray-300">分片大小</label>
            <span class="text-sm text-primary-400">{{ globalConfig.download_chunk_size }} KB</span>
          </div>
          <input
              type="range"
              v-model.number="globalConfig.download_chunk_size"
              @change="handleGlobalConfigChange"
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
            <span class="text-sm text-primary-400">{{ globalConfig.download_concurrency }}</span>
          </div>
          <input
              type="range"
              v-model.number="globalConfig.download_concurrency"
              @change="handleGlobalConfigChange"
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
        此配置对所有阿里云盘账号生效。较大的分片和更多并发可提高下载速度，但会占用更多内存
      </p>
    </div>

    <!-- 添加账号对话框 -->
    <div v-if="showAddAccount" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div class="bg-gray-900 rounded-2xl p-6 w-full max-w-md mx-4 ring-1 ring-white/10">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-lg font-medium text-white">添加阿里云盘账号</h3>
          <button @click="cancelAddAccount" class="text-gray-400 hover:text-white">
            <XMarkIcon class="w-5 h-5" />
          </button>
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
        <div class="flex justify-center gap-3 mt-4">
          <button
              v-if="!qrCode.url || qrCode.status === 'EXPIRED'"
              @click="generateQRCode"
              :disabled="qrLoading"
              class="px-5 py-2.5 rounded-lg bg-blue-500/20 text-blue-400 hover:bg-blue-500/30 ring-1 ring-blue-500/30 transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {{ qrLoading ? '生成中...' : (qrCode.status === 'EXPIRED' ? '刷新二维码' : '生成登录二维码') }}
          </button>
          <button
              @click="cancelAddAccount"
              class="px-5 py-2.5 rounded-lg bg-white/5 text-gray-400 hover:bg-white/10 ring-1 ring-white/10 transition-all duration-300"
          >
            取消
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onUnmounted } from 'vue'
import { storageApi } from '@/api/storage'
import type { StorageId } from '@/api/storage'
import type { AliyunPanStorageConfig, AliyunPanGlobalConfig, AliyunPanUserInfo } from '@/api/settings'
import { useDialogStore } from '@/stores/dialog'
import { UserIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import BaseSelect from '@/components/common/BaseSelect.vue'

const props = defineProps<{
  accounts: AliyunPanStorageConfig[]
  globalConfig?: AliyunPanGlobalConfig
  userInfos: AliyunPanUserInfo[]
  defaultStorageId: StorageId
}>()

const emit = defineEmits<{
  (e: 'update:accounts', accounts: AliyunPanStorageConfig[]): void
  (e: 'update:globalConfig', config: AliyunPanGlobalConfig): void
  (e: 'accountAdded', account: AliyunPanStorageConfig): void
  (e: 'accountRemoved', id: StorageId): void
  (e: 'setDefault', id: StorageId): void
}>()

const dialogStore = useDialogStore()

// 编辑状态
const editingIndex = ref<number | null>(null)

const driveTypeOptions = [
  { label: '备份盘', value: 'file' },
  { label: '相册', value: 'album' },
  { label: '资源库', value: 'resource' },
]

// 添加账号状态
const showAddAccount = ref(false)
const qrLoading = ref(false)
const qrCode = ref({
  url: '',
  status: '' as '' | 'NEW' | 'SCANED' | 'CONFIRMED' | 'EXPIRED',
  message: '',
})

let pollingTimer: ReturnType<typeof setInterval> | null = null

// 获取用户信息
function getUserInfo(id: StorageId): AliyunPanUserInfo | undefined {
  const parts = id.split(':')
  const userId = parts[1]
  return props.userInfos.find(u => u.user_id === userId)
}

// 生成二维码URL
function generateQRCodeDataUrl(text: string): string {
  return `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(text)}`
}

// 处理账号编辑
function handleEditAccount(index: number) {
  editingIndex.value = editingIndex.value === index ? null : index
}

// 处理账号配置变更
function handleAccountChange(_index: number) {
  emit('update:accounts', [...props.accounts])
}

// 处理全局配置变更
function handleGlobalConfigChange() {
  if (props.globalConfig) {
    emit('update:globalConfig', { ...props.globalConfig })
  }
}

// 处理删除账号
async function handleDeleteAccount(id: StorageId) {
  const confirmed = await dialogStore.confirm({
    title: '删除账号',
    message: '确定要删除此阿里云盘账号吗？删除后相关配置将被清除。',
    type: 'warning'
  })

  if (confirmed) {
    emit('accountRemoved', id)
  }
}

// 生成二维码
async function generateQRCode() {
  qrLoading.value = true
  try {
    const resp = await storageApi.generateAliyunPanQRCode()
    qrCode.value.url = resp.data.qr_code_url
    qrCode.value.status = resp.data.status as any
    qrCode.value.message = resp.data.message
    startPolling()
  } catch (error: any) {
    await dialogStore.alert({
      title: '错误',
      message: error.message || '生成二维码失败',
      type: 'error'
    })
  } finally {
    qrLoading.value = false
  }
}

// 开始轮询
function startPolling() {
  stopPolling()
  pollingTimer = setInterval(async () => {
    try {
      const resp = await storageApi.checkAliyunPanQRCodeStatus()
      qrCode.value.status = resp.data.status
      qrCode.value.message = resp.data.message

      if (resp.data.status === 'CONFIRMED') {
        stopPolling()

        if (resp.data.refresh_token && resp.data.user_id) {
          const newAccount: AliyunPanStorageConfig = {
            id: `aliyunpan:${resp.data.user_id}`,
            refresh_token: resp.data.refresh_token,
            base_path: '/gallery/images',
            drive_type: 'file',
          }
          emit('accountAdded', newAccount)
        }

        showAddAccount.value = false
        qrCode.value.url = ''
      } else if (resp.data.status === 'EXPIRED') {
        stopPolling()
      }
    } catch (error) {
      console.error('检查二维码状态失败:', error)
    }
  }, 2000)
}

// 停止轮询
function stopPolling() {
  if (pollingTimer) {
    clearInterval(pollingTimer)
    pollingTimer = null
  }
}

// 取消添加账号
function cancelAddAccount() {
  stopPolling()
  showAddAccount.value = false
  qrCode.value = { url: '', status: '', message: '' }
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
