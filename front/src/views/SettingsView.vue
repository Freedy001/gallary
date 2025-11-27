<template>
  <AppLayout>
    <template #header>
      <header
          class="sticky top-0 z-30 flex h-16 w-full items-center justify-between border-b border-white/5 bg-black/20 backdrop-blur-xl px-6 transition-all duration-300">
        <h1 class="text-xl font-bold tracking-wide text-white/90 font-display">系统设置</h1>
      </header>
    </template>

    <template #default>
      <div class="p-6 min-h-[calc(100vh-4rem)]">
        <!-- 加载状态 -->
        <div v-if="loading" class="flex h-64 items-center justify-center">
          <div
              class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent shadow-[0_0_15px_rgba(139,92,246,0.5)]"></div>
        </div>

        <div v-else class="max-w-4xl mx-auto">
          <!-- Tab 导航 -->
          <div class="flex gap-2 mb-8 p-1 rounded-xl bg-white/5 ring-1 ring-white/10 w-fit">
            <button
                v-for="tab in tabs"
                :key="tab.id"
                @click="activeTab = tab.id"
                :class="[
                'px-5 py-2.5 rounded-lg text-sm font-medium transition-all duration-300',
                activeTab === tab.id
                  ? 'bg-primary-500/20 text-primary-400 ring-1 ring-primary-500/30 shadow-[0_0_15px_rgba(139,92,246,0.2)]'
                  : 'text-gray-400 hover:text-white hover:bg-white/5'
              ]"
            >
              <component :is="tab.icon" class="h-4 w-4 inline-block mr-2"/>
              {{ tab.name }}
            </button>
          </div>

          <!-- 安全设置 Tab -->
          <div v-show="activeTab === 'security'" class="space-y-6">
            <div class="rounded-2xl bg-white/5 ring-1 ring-white/10 overflow-hidden">
              <div class="border-b border-white/5 p-5 bg-white/[0.02]">
                <h2 class="text-lg font-medium text-white">密码设置</h2>
                <p class="mt-1 text-sm text-gray-500">设置或修改管理员密码，启用认证保护</p>
              </div>
              <div class="p-6 space-y-5">
                <div v-if="passwordSet">
                  <label class="block text-sm font-medium text-gray-300 mb-2">当前密码</label>
                  <input
                      v-model="passwordForm.oldPassword"
                      type="password"
                      class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                      placeholder="请输入当前密码"
                  />
                </div>
                <div>
                  <label class="block text-sm font-medium text-gray-300 mb-2">{{
                      passwordSet ? '新密码' : '设置密码'
                    }}</label>
                  <input
                      v-model="passwordForm.newPassword"
                      type="password"
                      class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                      placeholder="请输入密码 (至少6位)"
                  />
                </div>
                <div>
                  <label class="block text-sm font-medium text-gray-300 mb-2">确认密码</label>
                  <input
                      v-model="passwordForm.confirmPassword"
                      type="password"
                      class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                      placeholder="请再次输入密码"
                  />
                </div>
                <div class="pt-2 flex items-center gap-4">
                  <button
                      @click="updatePassword"
                      :disabled="savingPassword"
                      class="px-6 py-2.5 rounded-lg bg-primary-500/20 text-primary-400 hover:bg-primary-500/30 ring-1 ring-primary-500/30 transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {{ savingPassword ? '保存中...' : '保存密码' }}
                  </button>

                  <button
                      v-if="authStore.token"
                      @click="handleLogout"
                      :disabled="loggingOut"
                      class="px-6 py-2.5 rounded-lg bg-red-500/20 text-red-400 hover:bg-red-500/30 ring-1 ring-red-500/30 transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {{ loggingOut ? '退出中...' : '退出登录' }}
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- 存储设置 Tab -->
          <div v-show="activeTab === 'storage'" class="space-y-6">
            <div class="rounded-2xl bg-white/5 ring-1 ring-white/10 overflow-hidden">
              <div class="border-b border-white/5 p-5 bg-white/[0.02]">
                <h2 class="text-lg font-medium text-white">存储配置</h2>
                <p class="mt-1 text-sm text-gray-500">选择并配置图片存储方式</p>
              </div>
              <div class="p-6 space-y-6">
                <!-- 存储类型选择 -->
                <div>
                  <label class="block text-sm font-medium text-gray-300 mb-3">存储类型</label>
                  <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
                    <button
                        v-for="type in storageTypes"
                        :key="type.value"
                        @click="storageForm.default_type = type.value"
                        :class="[
                        'p-4 rounded-xl border text-center transition-all duration-300',
                        storageForm.default_type === type.value
                          ? 'border-primary-500 bg-primary-500/10 text-primary-400'
                          : 'border-white/10 bg-white/[0.02] text-gray-400 hover:border-white/20 hover:bg-white/5'
                      ]"
                    >
                      <div class="text-sm font-medium">{{ type.label }}</div>
                      <div class="text-xs mt-1 opacity-60">{{ type.desc }}</div>
                    </button>
                  </div>
                </div>

                <!-- 本地存储配置 -->
                <div v-if="storageForm.default_type === 'local'" class="space-y-4 pt-4 border-t border-white/5">
                  <div>
                    <label class="block text-sm font-medium text-gray-300 mb-2">存储路径</label>
                    <input
                        v-model="storageForm.local_base_path"
                        type="text"
                        class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                        placeholder="例如: ./storage/images"
                    />
                  </div>
                  <div>
                    <label class="block text-sm font-medium text-gray-300 mb-2">URL 前缀</label>
                    <input
                        v-model="storageForm.local_url_prefix"
                        type="text"
                        class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                        placeholder="例如: /static/images"
                    />
                  </div>
                </div>

                <!-- OSS 配置 -->
                <div v-if="storageForm.default_type === 'oss'" class="space-y-4 pt-4 border-t border-white/5">
                  <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div>
                      <label class="block text-sm font-medium text-gray-300 mb-2">Endpoint</label>
                      <input v-model="storageForm.oss_endpoint" type="text"
                             class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                             placeholder="oss-cn-hangzhou.aliyuncs.com"/>
                    </div>
                    <div>
                      <label class="block text-sm font-medium text-gray-300 mb-2">Bucket</label>
                      <input v-model="storageForm.oss_bucket" type="text"
                             class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                             placeholder="your-bucket-name"/>
                    </div>
                    <div>
                      <label class="block text-sm font-medium text-gray-300 mb-2">Access Key ID</label>
                      <input v-model="storageForm.oss_access_key_id" type="text"
                             class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                             placeholder="Access Key ID"/>
                    </div>
                    <div>
                      <label class="block text-sm font-medium text-gray-300 mb-2">Access Key Secret</label>
                      <input v-model="storageForm.oss_access_key_secret" type="password"
                             class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                             placeholder="Access Key Secret"/>
                    </div>
                  </div>
                  <div>
                    <label class="block text-sm font-medium text-gray-300 mb-2">URL 前缀</label>
                    <input v-model="storageForm.oss_url_prefix" type="text"
                           class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                           placeholder="https://your-bucket.oss-cn-hangzhou.aliyuncs.com"/>
                  </div>
                </div>

                <!-- S3 配置 -->
                <div v-if="storageForm.default_type === 's3'" class="space-y-4 pt-4 border-t border-white/5">
                  <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div>
                      <label class="block text-sm font-medium text-gray-300 mb-2">Region</label>
                      <input v-model="storageForm.s3_region" type="text"
                             class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                             placeholder="us-east-1"/>
                    </div>
                    <div>
                      <label class="block text-sm font-medium text-gray-300 mb-2">Bucket</label>
                      <input v-model="storageForm.s3_bucket" type="text"
                             class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                             placeholder="your-bucket-name"/>
                    </div>
                    <div>
                      <label class="block text-sm font-medium text-gray-300 mb-2">Access Key ID</label>
                      <input v-model="storageForm.s3_access_key_id" type="text"
                             class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                             placeholder="Access Key ID"/>
                    </div>
                    <div>
                      <label class="block text-sm font-medium text-gray-300 mb-2">Secret Access Key</label>
                      <input v-model="storageForm.s3_secret_access_key" type="password"
                             class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                             placeholder="Secret Access Key"/>
                    </div>
                  </div>
                  <div>
                    <label class="block text-sm font-medium text-gray-300 mb-2">URL 前缀</label>
                    <input v-model="storageForm.s3_url_prefix" type="text"
                           class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                           placeholder="https://your-bucket.s3.amazonaws.com"/>
                  </div>
                </div>

                <!-- MinIO 配置 -->
                <div v-if="storageForm.default_type === 'minio'" class="space-y-4 pt-4 border-t border-white/5">
                  <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div>
                      <label class="block text-sm font-medium text-gray-300 mb-2">Endpoint</label>
                      <input v-model="storageForm.minio_endpoint" type="text"
                             class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                             placeholder="localhost:9000"/>
                    </div>
                    <div>
                      <label class="block text-sm font-medium text-gray-300 mb-2">Bucket</label>
                      <input v-model="storageForm.minio_bucket" type="text"
                             class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                             placeholder="your-bucket-name"/>
                    </div>
                    <div>
                      <label class="block text-sm font-medium text-gray-300 mb-2">Access Key ID</label>
                      <input v-model="storageForm.minio_access_key_id" type="text"
                             class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                             placeholder="Access Key ID"/>
                    </div>
                    <div>
                      <label class="block text-sm font-medium text-gray-300 mb-2">Secret Access Key</label>
                      <input v-model="storageForm.minio_secret_access_key" type="password"
                             class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                             placeholder="Secret Access Key"/>
                    </div>
                  </div>
                  <div class="flex items-center gap-3">
                    <label class="relative inline-flex items-center cursor-pointer">
                      <input v-model="storageForm.minio_use_ssl" type="checkbox" class="sr-only peer"/>
                      <div
                          class="w-11 h-6 bg-white/10 rounded-full peer peer-checked:bg-primary-500/50 peer-focus:ring-2 peer-focus:ring-primary-500/30 transition-colors after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full"></div>
                    </label>
                    <span class="text-sm text-gray-300">使用 SSL</span>
                  </div>
                  <div>
                    <label class="block text-sm font-medium text-gray-300 mb-2">URL 前缀</label>
                    <input v-model="storageForm.minio_url_prefix" type="text"
                           class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                           placeholder="http://localhost:9000/bucket"/>
                  </div>
                </div>

                <div class="pt-4">
                  <button
                      @click="updateStorage"
                      :disabled="savingStorage"
                      class="px-6 py-2.5 rounded-lg bg-primary-500/20 text-primary-400 hover:bg-primary-500/30 ring-1 ring-primary-500/30 transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {{ savingStorage ? '保存中...' : '保存存储配置' }}
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- 清理策略 Tab -->
          <div v-show="activeTab === 'cleanup'" class="space-y-6">
            <div class="rounded-2xl bg-white/5 ring-1 ring-white/10 overflow-hidden">
              <div class="border-b border-white/5 p-5 bg-white/[0.02]">
                <h2 class="text-lg font-medium text-white">自动清理策略</h2>
                <p class="mt-1 text-sm text-gray-500">配置回收站图片的自动清理规则</p>
              </div>
              <div class="p-6 space-y-6">
                <div>
                  <label class="block text-sm font-medium text-gray-300 mb-3">回收站自动删除天数</label>
                  <div class="flex items-center gap-4">
                    <input
                        v-model.number="cleanupForm.trash_auto_delete_days"
                        type="range"
                        min="0"
                        max="365"
                        class="flex-1 h-2 bg-white/10 rounded-lg appearance-none cursor-pointer accent-primary-500"
                    />
                    <div class="w-24 text-center">
                      <input
                          v-model.number="cleanupForm.trash_auto_delete_days"
                          type="number"
                          min="0"
                          max="365"
                          class="w-full rounded-lg bg-white/5 border border-white/10 px-3 py-2 text-white text-center focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                      />
                    </div>
                    <span class="text-gray-400 text-sm">天</span>
                  </div>
                  <p class="mt-3 text-xs text-gray-500">
                    {{
                      cleanupForm.trash_auto_delete_days === 0 ? '已禁用自动清理' : `回收站中的图片将在 ${cleanupForm.trash_auto_delete_days} 天后自动永久删除`
                    }}
                  </p>
                </div>

                <div class="pt-4">
                  <button
                      @click="updateCleanup"
                      :disabled="savingCleanup"
                      class="px-6 py-2.5 rounded-lg bg-primary-500/20 text-primary-400 hover:bg-primary-500/30 ring-1 ring-primary-500/30 transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {{ savingCleanup ? '保存中...' : '保存清理策略' }}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>
  </AppLayout>
</template>

<script setup lang="ts">
import {ref, reactive, onMounted} from 'vue'
import {useRouter} from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import {settingsApi, type StorageConfig, type CleanupConfig} from '@/api/settings'
import {useDialogStore} from '@/stores/dialog'
import {useAuthStore} from '@/stores/auth'
import {
  ShieldCheckIcon,
  CloudIcon,
  TrashIcon
} from '@heroicons/vue/24/outline'

const dialogStore = useDialogStore()
const router = useRouter()
const authStore = useAuthStore()

// Tab 配置
const tabs = [
  {id: 'security', name: '安全设置', icon: ShieldCheckIcon},
  {id: 'storage', name: '存储设置', icon: CloudIcon},
  {id: 'cleanup', name: '清理策略', icon: TrashIcon},
]

const activeTab = ref('security')
const loading = ref(true)
const passwordSet = ref(false)

// 存储类型选项
const storageTypes = [
  {value: 'local', label: '本地存储', desc: '文件系统'},
  {value: 'oss', label: '阿里云 OSS', desc: '对象存储'},
  {value: 's3', label: 'AWS S3', desc: '云存储'},
  {value: 'minio', label: 'MinIO', desc: '自托管'},
]

// 表单状态
const passwordForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
})

const storageForm = reactive<StorageConfig>({
  default_type: 'local',
  local_base_path: '',
  local_url_prefix: '',
  oss_endpoint: '',
  oss_access_key_id: '',
  oss_access_key_secret: '',
  oss_bucket: '',
  oss_url_prefix: '',
  s3_region: '',
  s3_access_key_id: '',
  s3_secret_access_key: '',
  s3_bucket: '',
  s3_url_prefix: '',
  minio_endpoint: '',
  minio_access_key_id: '',
  minio_secret_access_key: '',
  minio_bucket: '',
  minio_use_ssl: false,
  minio_url_prefix: '',
})

const cleanupForm = reactive<CleanupConfig>({
  trash_auto_delete_days: 30,
})

// 保存状态
const savingPassword = ref(false)
const savingStorage = ref(false)
const savingCleanup = ref(false)
const loggingOut = ref(false)

// 加载设置
async function loadSettings() {
  loading.value = true
  try {
    // 并行加载所有设置
    const [allSettings, passwordStatus] = await Promise.all([
      settingsApi.getAll(),
      settingsApi.getPasswordStatus(),
    ])

    passwordSet.value = passwordStatus.data.is_set

    // 应用设置到表单
    const data = allSettings.data
    if (data.storage_default_type) storageForm.default_type = data.storage_default_type as StorageConfig['default_type']
    if (data.local_base_path) storageForm.local_base_path = data.local_base_path
    if (data.local_url_prefix) storageForm.local_url_prefix = data.local_url_prefix
    if (data.oss_endpoint) storageForm.oss_endpoint = data.oss_endpoint
    if (data.oss_access_key_id) storageForm.oss_access_key_id = data.oss_access_key_id
    if (data.oss_bucket) storageForm.oss_bucket = data.oss_bucket
    if (data.oss_url_prefix) storageForm.oss_url_prefix = data.oss_url_prefix
    if (data.s3_region) storageForm.s3_region = data.s3_region
    if (data.s3_access_key_id) storageForm.s3_access_key_id = data.s3_access_key_id
    if (data.s3_bucket) storageForm.s3_bucket = data.s3_bucket
    if (data.s3_url_prefix) storageForm.s3_url_prefix = data.s3_url_prefix
    if (data.minio_endpoint) storageForm.minio_endpoint = data.minio_endpoint
    if (data.minio_access_key_id) storageForm.minio_access_key_id = data.minio_access_key_id
    if (data.minio_bucket) storageForm.minio_bucket = data.minio_bucket
    if (data.minio_use_ssl !== undefined) storageForm.minio_use_ssl = data.minio_use_ssl
    if (data.minio_url_prefix) storageForm.minio_url_prefix = data.minio_url_prefix
    if (data.trash_auto_delete_days !== undefined) cleanupForm.trash_auto_delete_days = data.trash_auto_delete_days
  } catch (error) {
    console.error('Failed to load settings:', error)
  } finally {
    loading.value = false
  }
}

// 更新密码
async function updatePassword() {
  if (passwordForm.newPassword.length < 6) {
    await dialogStore.alert({title: '错误', message: '密码长度至少为6位', type: 'error'})
    return
  }
  if (passwordForm.newPassword !== passwordForm.confirmPassword) {
    await dialogStore.alert({title: '错误', message: '两次输入的密码不一致', type: 'error'})
    return
  }

  savingPassword.value = true
  try {
    await settingsApi.updatePassword({
      old_password: passwordForm.oldPassword,
      new_password: passwordForm.newPassword,
    })
    dialogStore.alert({title: '成功', message: '密码更新成功', type: 'success'})
    passwordForm.oldPassword = ''
    passwordForm.newPassword = ''
    passwordForm.confirmPassword = ''
    passwordSet.value = true
  } catch (error: any) {
    dialogStore.alert({title: '错误', message: error.message || '更新密码失败', type: 'error'})
  } finally {
    savingPassword.value = false
  }
}

// 更新存储配置
async function updateStorage() {
  savingStorage.value = true
  try {
    await settingsApi.updateStorage(storageForm)
    dialogStore.alert({title: '成功', message: '存储配置更新成功', type: 'success'})
  } catch (error: any) {
    dialogStore.alert({title: '错误', message: error.message || '更新存储配置失败', type: 'error'})
  } finally {
    savingStorage.value = false
  }
}

// 更新清理配置
async function updateCleanup() {
  savingCleanup.value = true
  try {
    await settingsApi.updateCleanup(cleanupForm)
    dialogStore.alert({title: '成功', message: '清理策略更新成功', type: 'success'})
  } catch (error: any) {
    dialogStore.alert({title: '错误', message: error.message || '更新清理策略失败', type: 'error'})
  } finally {
    savingCleanup.value = false
  }
}

// 退出登录
async function handleLogout() {
  loggingOut.value = true
  try {
    // 显示确认对话框
    const result = await dialogStore.confirm({
      title: '退出登录',
      message: '确定要退出登录吗？退出后需要重新输入密码才能访问系统。',
      type: 'warning'
    })

    if (!result) {
      return
    }

    // 执行退出登录
    authStore.logout()

    await router.push('/login')

  } catch (error: any) {
    await dialogStore.alert({
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
