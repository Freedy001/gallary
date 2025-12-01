<template>
  <div class="rounded-2xl bg-white/5 ring-1 ring-white/10 overflow-hidden">
    <!-- 迁移进度（在卡片内顶部） -->
    <MigrationProgress
        ref="migrationProgressRef"
        @migration-completed="handleMigrationCompleted"
    />

    <div class="border-b border-white/5 p-5 bg-white/[0.02]">
      <h2 class="text-lg font-medium text-white">存储配置</h2>
      <p class="mt-1 text-sm text-gray-500">选择并配置图片存储方式</p>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="p-6 flex justify-center">
      <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
    </div>

    <!-- 迁移中时显示锁定提示 -->
    <div v-else-if="isMigrating" class="p-6 text-center space-y-4">
      <div class="text-gray-400">
        <div class="h-12 w-12 mx-auto mb-4 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
        <p class="text-lg font-medium text-white">迁移进行中</p>
        <p class="text-sm mt-2">迁移期间配置无法修改，请等待迁移完成...</p>
      </div>
    </div>

    <div v-else class="p-6 space-y-6">
      <!-- 默认存储方式选择 -->
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-2">默认存储方式</label>
        <select
            v-model="form.storage_default_type"
            class="w-full md:w-64 px-4 py-2.5 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-primary-500 focus:ring-1 focus:ring-primary-500 transition-colors"
        >
          <option v-for="type in storageTypes" :key="type.value" :value="type.value">
            {{ type.label }}
          </option>
        </select>
        <p class="mt-1.5 text-xs text-gray-500">新上传的图片将使用此存储方式</p>
      </div>

      <!-- 存储配置选择 -->
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-3">存储配置</label>
        <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
          <button
              v-for="type in storageTypes"
              :key="type.value"
              @click="editingType = type.value as StorageConfig['storage_default_type']"
              :class="[
              'p-4 rounded-xl border text-center transition-all duration-300 relative',
              editingType === type.value
                ? 'border-primary-500 bg-primary-500/10 text-primary-400'
                : 'border-white/10 bg-white/[0.02] text-gray-400 hover:border-white/20 hover:bg-white/5'
            ]"
          >
            <!-- 默认标识 -->
            <span
                v-if="form.storage_default_type === type.value"
                class="absolute -top-2 -right-2 px-2 py-0.5 text-[10px] font-medium rounded-full bg-green-500/20 text-green-400 border border-green-500/30"
            >
              默认
            </span>
            <div class="text-sm font-medium">{{ type.label }}</div>
            <div class="text-xs mt-1 opacity-60">{{ type.desc }}</div>
          </button>
        </div>
      </div>

      <!-- 本地存储配置 -->
      <LocalStorageConfig
          v-if="editingType === 'local'"
          v-model:base-path="form.local_base_path"
          v-model:url-prefix="form.local_url_prefix"
      />

      <!-- 阿里云盘配置 -->
      <AliyunPanConfig
          v-if="editingType === 'aliyunpan'"
          v-model:refresh-token="form.aliyunpan_refresh_token"
          v-model:base-path="form.aliyunpan_base_path"
          v-model:drive-type="form.aliyunpan_drive_type"
          v-model:download-chunk-size="form.aliyunpan_download_chunk_size"
          v-model:download-concurrency="form.aliyunpan_download_concurrency"
          :user-info="form.aliyunpan_user"
          @logout="handleAliyunPanLogout"
      />

      <!-- OSS 配置 -->
      <OSSConfig
          v-if="editingType === 'oss'"
          v-model:endpoint="form.oss_endpoint"
          v-model:bucket="form.oss_bucket"
          v-model:access-key-id="form.oss_access_key_id"
          v-model:access-key-secret="form.oss_access_key_secret"
          v-model:url-prefix="form.oss_url_prefix"
      />

      <!-- S3 配置 -->
      <S3Config
          v-if="editingType === 's3'"
          v-model:region="form.s3_region"
          v-model:bucket="form.s3_bucket"
          v-model:access-key-id="form.s3_access_key_id"
          v-model:secret-access-key="form.s3_secret_access_key"
          v-model:url-prefix="form.s3_url_prefix"
      />

      <!-- MinIO 配置 -->
      <MinIOConfig
          v-if="editingType === 'minio'"
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
import {ref, reactive, onMounted, computed} from 'vue'
import {settingsApi, type StorageConfig} from '@/api/settings'
import {useDialogStore} from '@/stores/dialog'
import LocalStorageConfig from './storage/LocalStorageConfig.vue'
import AliyunPanConfig from './storage/AliyunPanConfig.vue'
import OSSConfig from './storage/OSSConfig.vue'
import S3Config from './storage/S3Config.vue'
import MinIOConfig from './storage/MinIOConfig.vue'
import MigrationProgress from './MigrationProgress.vue'

const dialogStore = useDialogStore()

const storageTypes = [
  {value: 'local', label: '本地存储', desc: '文件系统'},
  {value: 'aliyunpan', label: '阿里云盘', desc: '网盘存储'},
  // {value: 'oss', label: '阿里云 OSS', desc: '对象存储'},
  // {value: 's3', label: 'AWS S3', desc: '云存储'},
  // {value: 'minio', label: 'MinIO', desc: '自托管'},
]

const loading = ref(true)
const saving = ref(false)

// 迁移进度组件引用
const migrationProgressRef = ref<InstanceType<typeof MigrationProgress> | null>(null)

// 是否正在迁移
const isMigrating = computed(() => {
  return migrationProgressRef.value?.isMigrating ?? false
})

// 当前正在编辑的存储类型（与默认类型分开）
const editingType = ref<StorageConfig['storage_default_type']>('local')

const form = reactive<StorageConfig>({
  storage_default_type: 'local',
  local_base_path: '',
  local_url_prefix: '',
  aliyunpan_refresh_token: '',
  aliyunpan_base_path: '/gallery/images',
  aliyunpan_drive_type: 'file',
  aliyunpan_download_chunk_size: 512,
  aliyunpan_download_concurrency: 8,
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

async function loadSettings() {
  loading.value = true
  try {
    const resp = await settingsApi.getByCategory('storage')
    const data = resp.data

    Object.assign(form, data)
    // 初始化时，编辑类型默认为当前默认存储类型
    editingType.value = form.storage_default_type
  } catch (error) {
    console.error('Failed to load storage settings:', error)
  } finally {
    loading.value = false
  }
}

function handleMigrationCompleted() {
  // 迁移完成后重新加载设置
  loadSettings()
}

async function handleAliyunPanLogout() {
  // 清除用户信息并保存配置
  form.aliyunpan_user = undefined
  await handleSave()
  // 重新加载设置以获取最新状态
  await loadSettings()
}

async function handleSave() {
  saving.value = true
  try {
    const resp = await settingsApi.updateStorage(form)
    const result = resp.data

    if (result.needs_migration) {
      // 触发了迁移，开始轮询进度
      if (migrationProgressRef.value) {
        migrationProgressRef.value.refresh()
      }
      await dialogStore.alert({
        title: '迁移已启动',
        message: '存储路径已变更，正在迁移文件...',
        type: 'info'
      })
    } else {
      await dialogStore.alert({
        title: '成功',
        message: result.message || '存储配置更新成功',
        type: 'success'
      })
    }
  } catch (error: any) {
    // 检查是否是迁移锁定错误
    if (error.response?.status === 423) {
      await dialogStore.alert({
        title: '配置被锁定',
        message: '迁移正在进行中，请等待完成后再修改配置',
        type: 'warning'
      })
    } else {
      await dialogStore.alert({
        title: '错误',
        message: error.message || '更新存储配置失败',
        type: 'error'
      })
    }
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadSettings()
})
</script>
