<template>
  <div class="space-y-4 pt-4 border-t border-white/5">
    <!-- 账号列表 -->
    <div class="space-y-3">
      <div class="flex items-center justify-between">
        <h3 class="text-sm font-medium text-gray-300">S3 存储账号</h3>
        <button
            class="px-3 py-1.5 rounded-lg bg-primary-500/20 text-primary-400 hover:bg-primary-500/30 ring-1 ring-primary-500/30 transition-all duration-300 text-xs"
            @click="openAddModal"
        >
          + 添加账号
        </button>
      </div>

      <!-- 已配置的账号列表 -->
      <div v-if="accounts.length > 0" class="space-y-2">
        <div
            v-for="account in accounts"
            :key="account.id"
            class="rounded-xl bg-white/[0.03] border border-white/10 p-4"
        >
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-3">
              <!-- S3 图标 -->
              <div class="w-10 h-10 rounded-full bg-orange-500/20 flex items-center justify-center">
                <CloudIcon class="w-5 h-5 text-orange-400" />
              </div>
              <div>
                <h4 class="text-sm font-medium text-white flex items-center gap-2">
                  {{ account.name || 'S3 存储' }}
                  <span
                      class="px-2 py-0.5 text-[10px] font-medium rounded-full bg-gray-500/20 text-gray-400 border border-gray-500/30"
                  >
                    {{ getProviderName(account.provider) }}
                  </span>
                  <span
                      v-if="defaultStorageId === account.id"
                      class="px-2 py-0.5 text-[10px] font-medium rounded-full bg-green-500/20 text-green-400 border border-green-500/30"
                  >
                    默认
                  </span>
                </h4>
                <p class="text-xs text-gray-500 mt-0.5">{{ account.bucket }} · {{ account.region }}</p>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <button
                  v-if="defaultStorageId !== account.id"
                  class="px-3 py-1.5 rounded-lg bg-white/5 text-gray-400 hover:bg-white/10 hover:text-white ring-1 ring-white/10 transition-all duration-300 text-xs"
                  @click="emit('setDefault', account.id)"
              >
                设为默认
              </button>
              <button
                  class="px-3 py-1.5 rounded-lg bg-white/5 text-gray-400 hover:bg-white/10 hover:text-white ring-1 ring-white/10 transition-all duration-300 text-xs"
                  @click="openEditModal(account)"
              >
                编辑
              </button>
              <button
                  :disabled="defaultStorageId === account.id"
                  class="px-3 py-1.5 rounded-lg bg-red-500/10 text-red-400 hover:bg-red-500/20 ring-1 ring-red-500/20 transition-all duration-300 text-xs disabled:opacity-50 disabled:cursor-not-allowed"
                  @click="handleDeleteAccount(account.id)"
              >
                删除
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 无账号提示 -->
      <div v-else class="text-center py-8 text-gray-500">
        <p class="text-sm">暂无配置的 S3 存储账号</p>
        <p class="text-xs mt-1">点击上方「添加账号」按钮配置 S3 兼容存储</p>
      </div>
    </div>

    <!-- 添加/编辑账号对话框 -->
    <div v-if="showModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div class="bg-gray-900 rounded-2xl p-6 w-full max-w-2xl mx-4 ring-1 ring-white/10 max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-lg font-medium text-white">{{ editingAccount ? '编辑 S3 账号' : '添加 S3 账号' }}</h3>
          <button class="text-gray-400 hover:text-white" @click="closeModal">
            <XMarkIcon class="w-5 h-5" />
          </button>
        </div>

        <form class="space-y-4" @submit.prevent="handleSubmit">
          <!-- 基础配置 -->
          <div class="space-y-4">
            <h4 class="text-sm font-medium text-gray-400">基础配置</h4>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label class="block text-sm font-medium text-gray-300 mb-2">账号名称 <span class="text-red-400">*</span></label>
                <input
                    v-model="formData.name"
                    class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                    placeholder="例如: 主存储"
                    required
                    type="text"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-300 mb-2">服务商 <span class="text-red-400">*</span></label>
                <BaseSelect
                    v-model="formData.provider"
                    :options="providerOptions"
                    placeholder="选择服务商"
                    @update:modelValue="handleProviderChange"
                />
              </div>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label class="block text-sm font-medium text-gray-300 mb-2">Endpoint <span class="text-red-400">*</span></label>
                <input
                    v-model="formData.endpoint"
                    class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                    placeholder="s3.amazonaws.com"
                    required
                    type="text"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-300 mb-2">Region <span class="text-red-400">*</span></label>
                <input
                    v-model="formData.region"
                    class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                    placeholder="us-east-1"
                    required
                    type="text"
                />
              </div>
            </div>

            <div>
              <label class="block text-sm font-medium text-gray-300 mb-2">Bucket <span class="text-red-400">*</span></label>
              <input
                  v-model="formData.bucket"
                  class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                  placeholder="your-bucket-name"
                  required
                  type="text"
              />
            </div>
          </div>

          <!-- 认证配置 -->
          <div class="space-y-4 pt-4 border-t border-white/5">
            <h4 class="text-sm font-medium text-gray-400">认证配置</h4>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label class="block text-sm font-medium text-gray-300 mb-2">Access Key ID <span class="text-red-400">*</span></label>
                <div class="relative">
                  <input
                      v-model="formData.access_key_id"
                      :type="showAccessKeyId ? 'text' : 'password'"
                      class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 pr-12 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                      placeholder="Access Key ID"
                      required
                  />
                  <button
                      class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white transition-colors"
                      type="button"
                      @click="showAccessKeyId = !showAccessKeyId"
                  >
                    <EyeIcon v-if="!showAccessKeyId" class="h-5 w-5" />
                    <EyeSlashIcon v-else class="h-5 w-5" />
                  </button>
                </div>
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-300 mb-2">Secret Access Key <span class="text-red-400">*</span></label>
                <div class="relative">
                  <input
                      v-model="formData.secret_access_key"
                      :type="showSecretKey ? 'text' : 'password'"
                      class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 pr-12 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                      placeholder="Secret Access Key"
                      required
                  />
                  <button
                      class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white transition-colors"
                      type="button"
                      @click="showSecretKey = !showSecretKey"
                  >
                    <EyeIcon v-if="!showSecretKey" class="h-5 w-5" />
                    <EyeSlashIcon v-else class="h-5 w-5" />
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- 高级配置 -->
          <div class="space-y-4 pt-4 border-t border-white/5">
            <button
                class="flex items-center gap-2 text-sm font-medium text-gray-400 hover:text-white transition-colors"
                type="button"
                @click="showAdvanced = !showAdvanced"
            >
              <ChevronDownIcon
                  :class="{ 'rotate-180': showAdvanced }"
                  class="w-4 h-4 transition-transform"
              />
              高级配置
            </button>

            <div v-if="showAdvanced" class="space-y-4">
              <div>
                <label class="block text-sm font-medium text-gray-300 mb-2">存储路径前缀</label>
                <input
                    v-model="formData.base_path"
                    class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                    placeholder="例如: gallery/images"
                    type="text"
                />
                <p class="text-xs text-gray-500 mt-1">文件将存储在此路径下，留空则存储在 Bucket 根目录</p>
              </div>

              <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div class="flex items-center justify-between p-3 rounded-lg bg-white/[0.02] border border-white/5">
                  <div>
                    <p class="text-sm text-gray-300">使用 HTTPS</p>
                    <p class="text-xs text-gray-500">启用 SSL/TLS 加密连接</p>
                  </div>
                  <label class="relative inline-flex cursor-pointer">
                    <input v-model="formData.use_ssl" class="sr-only peer" type="checkbox" />
                    <div class="w-11 h-6 bg-white/10 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-500"></div>
                  </label>
                </div>

                <div class="flex items-center justify-between p-3 rounded-lg bg-white/[0.02] border border-white/5">
                  <div>
                    <p class="text-sm text-gray-300">路径风格 URL</p>
                    <p class="text-xs text-gray-500">MinIO 等服务需要开启</p>
                  </div>
                  <label class="relative inline-flex cursor-pointer">
                    <input v-model="formData.force_path_style" class="sr-only peer" type="checkbox" />
                    <div class="w-11 h-6 bg-white/10 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-500"></div>
                  </label>
                </div>
              </div>

              <div>
                <label class="block text-sm font-medium text-gray-300 mb-2">自定义访问 URL 前缀</label>
                <input
                    v-model="formData.url_prefix"
                    class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                    placeholder="https://cdn.example.com"
                    type="text"
                />
                <p class="text-xs text-gray-500 mt-1">配置 CDN 加速域名，留空则使用默认预签名 URL</p>
              </div>

              <div>
                <label class="block text-sm font-medium text-gray-300 mb-2">HTTP 代理</label>
                <input
                    v-model="formData.proxy_url"
                    class="w-full rounded-lg bg-white/5 border border-white/10 px-4 py-3 text-white placeholder-gray-500 focus:border-primary-500 focus:ring-1 focus:ring-primary-500 focus:outline-none transition-colors"
                    placeholder="http://127.0.0.1:8080"
                    type="text"
                />
                <p class="text-xs text-gray-500 mt-1">如需通过代理访问 S3 存储，请填写代理地址</p>
              </div>
            </div>
          </div>

          <!-- 操作按钮 -->
          <div class="flex justify-between pt-4 border-t border-white/5">
            <button
                :disabled="testing || !canTest"
                class="px-5 py-2.5 rounded-lg bg-green-500/20 text-green-400 hover:bg-green-500/30 ring-1 ring-green-500/30 transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed"
                type="button"
                @click="handleTestConnection"
            >
              {{ testing ? '测试中...' : '测试连接' }}
            </button>
            <div class="flex gap-3">
              <button
                  class="px-5 py-2.5 rounded-lg bg-white/5 text-gray-400 hover:bg-white/10 ring-1 ring-white/10 transition-all duration-300"
                  type="button"
                  @click="closeModal"
              >
                取消
              </button>
              <button
                  :disabled="submitting"
                  class="px-5 py-2.5 rounded-lg bg-primary-500/20 text-primary-400 hover:bg-primary-500/30 ring-1 ring-primary-500/30 transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed"
                  type="submit"
              >
                {{ submitting ? '保存中...' : (editingAccount ? '保存' : '添加') }}
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {computed, reactive, ref} from 'vue'
import type {StorageId} from '@/api/storage'
import type {S3Provider, S3StorageConfig} from '@/api/settings'
import {S3_PROVIDER_PRESETS, settingsApi} from '@/api/settings'
import {useDialogStore} from '@/stores/dialog'
import {ChevronDownIcon, CloudIcon, EyeIcon, EyeSlashIcon, XMarkIcon} from '@heroicons/vue/24/outline'
import BaseSelect from '@/components/common/BaseSelect.vue'

defineProps<{
  accounts: S3StorageConfig[]
  defaultStorageId: StorageId
}>()

const emit = defineEmits<{
  (e: 'update:accounts', accounts: S3StorageConfig[]): void
  (e: 'accountAdded', account: S3StorageConfig): void
  (e: 'accountUpdated', account: S3StorageConfig): void
  (e: 'accountRemoved', id: StorageId): void
  (e: 'setDefault', id: StorageId): void
}>()

const dialogStore = useDialogStore()

// 对话框状态
const showModal = ref(false)
const showAdvanced = ref(false)
const showAccessKeyId = ref(false)
const showSecretKey = ref(false)
const submitting = ref(false)
const testing = ref(false)
const editingAccount = ref<S3StorageConfig | null>(null)

// 表单数据
const defaultFormData = {
  name: '',
  provider: 'aws' as S3Provider,
  endpoint: 's3.us-east-1.amazonaws.com',
  region: 'us-east-1',
  bucket: '',
  access_key_id: '',
  secret_access_key: '',
  base_path: '',
  use_ssl: true,
  force_path_style: false,
  url_prefix: '',
  proxy_url: '',
}

const formData = reactive({ ...defaultFormData })

// 服务商选项
const providerOptions = [
  { label: 'AWS S3', value: 'aws' },
  { label: 'MinIO', value: 'minio' },
  { label: '阿里云 OSS', value: 'aliyun-oss' },
  { label: '七牛云', value: 'qiniu' },
  { label: '腾讯云 COS', value: 'tencent-cos' },
  { label: '其他', value: 'other' },
]

// 获取服务商名称
function getProviderName(provider: string): string {
  const option = providerOptions.find(o => o.value === provider)
  return option?.label || provider
}

// 处理服务商变更 - 自动填充预设配置
function handleProviderChange(provider: string | number) {
  const providerKey = provider as S3Provider
  const preset = S3_PROVIDER_PRESETS[providerKey]
  if (preset) {
    formData.provider = providerKey
    formData.endpoint = preset.endpoint.replace('{region}', formData.region || preset.region)
    formData.region = formData.region || preset.region
    formData.force_path_style = preset.forcePathStyle
  }
}

// 打开添加对话框
function openAddModal() {
  editingAccount.value = null
  Object.assign(formData, defaultFormData)
  showAdvanced.value = false
  showModal.value = true
}

// 打开编辑对话框
function openEditModal(account: S3StorageConfig) {
  editingAccount.value = account
  Object.assign(formData, {
    name: account.name,
    provider: account.provider,
    endpoint: account.endpoint,
    region: account.region,
    bucket: account.bucket,
    access_key_id: account.access_key_id,
    secret_access_key: account.secret_access_key,
    base_path: account.base_path || '',
    use_ssl: account.use_ssl ?? true,
    force_path_style: account.force_path_style ?? false,
    url_prefix: account.url_prefix || '',
    proxy_url: account.proxy_url || '',
  })
  showAdvanced.value = !!(account.base_path || account.url_prefix || account.proxy_url || !account.use_ssl || account.force_path_style)
  showModal.value = true
}

// 关闭对话框
function closeModal() {
  showModal.value = false
  editingAccount.value = null
  showAccessKeyId.value = false
  showSecretKey.value = false
}

// 处理表单提交
async function handleSubmit() {
  if (!formData.name || !formData.endpoint || !formData.region || !formData.bucket || !formData.access_key_id || !formData.secret_access_key) {
    dialogStore.alert({
      title: '错误',
      message: '请填写所有必填字段',
      type: 'error'
    })
    return
  }

  submitting.value = true

  try {
    const accountData: S3StorageConfig = {
      id: editingAccount.value?.id || `s3:${formData.name.toLowerCase().replace(/\s+/g, '-')}`,
      name: formData.name,
      provider: formData.provider,
      endpoint: formData.endpoint,
      region: formData.region,
      bucket: formData.bucket,
      access_key_id: formData.access_key_id,
      secret_access_key: formData.secret_access_key,
      base_path: formData.base_path || undefined,
      use_ssl: formData.use_ssl,
      force_path_style: formData.force_path_style,
      url_prefix: formData.url_prefix || undefined,
      proxy_url: formData.proxy_url || undefined,
    }

    if (editingAccount.value) {
      emit('accountUpdated', accountData)
    } else {
      emit('accountAdded', accountData)
    }

    closeModal()
  } finally {
    submitting.value = false
  }
}

// 处理删除账号
async function handleDeleteAccount(id: StorageId) {
  const confirmed = await dialogStore.confirm({
    title: '删除账号',
    message: '确定要删除此 S3 存储账号吗？删除后相关配置将被清除。',
    type: 'warning'
  })

  if (confirmed) {
    emit('accountRemoved', id)
  }
}

// 测试连接必填字段检查
const canTest = computed(() => {
  return !!(
    formData.endpoint &&
    formData.region &&
    formData.bucket &&
    formData.access_key_id &&
    formData.secret_access_key
  )
})

// 测试连接
async function handleTestConnection() {
  if (!canTest.value) {
    dialogStore.alert({
      title: '提示',
      message: '请先填写 Endpoint、Region、Bucket、Access Key ID 和 Secret Access Key',
      type: 'warning'
    })
    return
  }

  testing.value = true
  try {
    const resp = await settingsApi.testS3Connection({
      name: formData.name || 'test',
      provider: formData.provider,
      endpoint: formData.endpoint,
      region: formData.region,
      bucket: formData.bucket,
      access_key_id: formData.access_key_id,
      secret_access_key: formData.secret_access_key,
      base_path: formData.base_path || undefined,
      use_ssl: formData.use_ssl,
      force_path_style: formData.force_path_style,
      url_prefix: formData.url_prefix || undefined,
      proxy_url: formData.proxy_url || undefined,
    })
    dialogStore.alert({
      title: '连接成功',
      message: `成功连接到 Bucket: ${resp.data.bucket} (${resp.data.region})`,
      type: 'success'
    })
  } catch (error: any) {
    dialogStore.alert({
      title: '连接失败',
      message: error.response?.data?.message || error.message || '无法连接到 S3 存储',
      type: 'error'
    })
  } finally {
    testing.value = false
  }
}
</script>
