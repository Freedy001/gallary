<template>
  <div class="rounded-2xl bg-white/5 ring-1 ring-white/10 overflow-hidden">
    <div class="border-b border-white/5 p-5 bg-white/[0.02]">
      <h2 class="text-lg font-medium text-white">存储配置</h2>
      <p class="mt-1 text-sm text-gray-500">选择并配置图片存储方式</p>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="p-6 flex justify-center">
      <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
    </div>

    <div v-else class="p-6 space-y-6">
      <!-- 存储类型选择 -->
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-3">存储类型</label>
        <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
          <button
              v-for="type in storageTypes"
              :key="type.value"
              @click="form.storage_default_type = type.value as StorageConfig['storage_default_type']"
              :class="[
              'p-4 rounded-xl border text-center transition-all duration-300',
              form.storage_default_type === type.value
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
      <LocalStorageConfig
          v-if="form.storage_default_type === 'local'"
          v-model:base-path="form.local_base_path"
          v-model:url-prefix="form.local_url_prefix"
          :original-base-path="originalLocalBasePath"
          :original-url-prefix="originalLocalUrlPrefix"
          @migration-started="handleMigrationStarted"
      />

      <!-- 阿里云盘配置 -->
      <AliyunPanConfig
          v-if="form.storage_default_type === 'aliyunpan'"
          v-model:refresh-token="form.aliyunpan_refresh_token"
          v-model:base-path="form.aliyunpan_base_path"
          v-model:drive-type="form.aliyunpan_drive_type"
      />

      <!-- OSS 配置 -->
      <OSSConfig
          v-if="form.storage_default_type === 'oss'"
          v-model:endpoint="form.oss_endpoint"
          v-model:bucket="form.oss_bucket"
          v-model:access-key-id="form.oss_access_key_id"
          v-model:access-key-secret="form.oss_access_key_secret"
          v-model:url-prefix="form.oss_url_prefix"
      />

      <!-- S3 配置 -->
      <S3Config
          v-if="form.storage_default_type === 's3'"
          v-model:region="form.s3_region"
          v-model:bucket="form.s3_bucket"
          v-model:access-key-id="form.s3_access_key_id"
          v-model:secret-access-key="form.s3_secret_access_key"
          v-model:url-prefix="form.s3_url_prefix"
      />

      <!-- MinIO 配置 -->
      <MinIOConfig
          v-if="form.storage_default_type === 'minio'"
          v-model:endpoint="form.minio_endpoint"
          v-model:bucket="form.minio_bucket"
          v-model:access-key-id="form.minio_access_key_id"
          v-model:secret-access-key="form.minio_secret_access_key"
          :use-ssl="form.minio_use_ssl ?? false"
          @update:use-ssl="form.minio_use_ssl = $event"
          v-model:url-prefix="form.minio_url_prefix"
      />

      <div class="pt-4">
        <button
            @click="handleSave"
            :disabled="saving"
            class="px-6 py-2.5 rounded-lg bg-primary-500/20 text-primary-400 hover:bg-primary-500/30 ring-1 ring-primary-500/30 transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {{ saving ? '保存中...' : '保存存储配置' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {ref, reactive, onMounted} from 'vue'
import {settingsApi, type StorageConfig} from '@/api/settings'
import {useDialogStore} from '@/stores/dialog'
import LocalStorageConfig from './storage/LocalStorageConfig.vue'
import AliyunPanConfig from './storage/AliyunPanConfig.vue'
import OSSConfig from './storage/OSSConfig.vue'
import S3Config from './storage/S3Config.vue'
import MinIOConfig from './storage/MinIOConfig.vue'

const dialogStore = useDialogStore()

const storageTypes = [
  {value: 'local', label: '本地存储', desc: '文件系统'},
  {value: 'aliyunpan', label: '阿里云盘', desc: '网盘存储'},
  {value: 'oss', label: '阿里云 OSS', desc: '对象存储'},
  {value: 's3', label: 'AWS S3', desc: '云存储'},
  {value: 'minio', label: 'MinIO', desc: '自托管'},
]

const loading = ref(true)
const saving = ref(false)

const form = reactive<StorageConfig>({
  storage_default_type: 'local',
  local_base_path: '',
  local_url_prefix: '',
  aliyunpan_refresh_token: '',
  aliyunpan_base_path: '/gallery/images',
  aliyunpan_drive_type: 'file',
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

// 原始存储路径（用于迁移功能）
const originalLocalBasePath = ref('')
const originalLocalUrlPrefix = ref('')

async function loadSettings() {
  loading.value = true
  try {
    const resp = await settingsApi.getByCategory('storage')
    const data = resp.data

    if (data.local_base_path) originalLocalBasePath.value = data.local_base_path
    if (data.local_url_prefix) originalLocalUrlPrefix.value = data.local_url_prefix

    Object.assign(form, data)
  } catch (error) {
    console.error('Failed to load storage settings:', error)
  } finally {
    loading.value = false
  }
}

function handleMigrationStarted() {
  originalLocalBasePath.value = form.local_base_path || ''
  originalLocalUrlPrefix.value = form.local_url_prefix || ''
}

async function handleSave() {
  saving.value = true
  try {
    await settingsApi.updateStorage(form)
    await dialogStore.alert({title: '成功', message: '存储配置更新成功', type: 'success'})
    originalLocalBasePath.value = form.local_base_path || ''
    originalLocalUrlPrefix.value = form.local_url_prefix || ''
  } catch (error: any) {
    await dialogStore.alert({title: '错误', message: error.message || '更新存储配置失败', type: 'error'})
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadSettings()
})
</script>
