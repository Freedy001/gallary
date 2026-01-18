<template>
  <div class="rounded-2xl bg-white/5 ring-1 ring-white/10 overflow-hidden">

    <div class="border-b border-white/5 p-5 bg-white/2 flex items-center justify-between">
      <div>
        <h2 class="text-lg font-medium text-white">存储配置</h2>
        <p class="mt-1 text-sm text-gray-500">选择并配置图片存储方式</p>
      </div>
      <button
          class="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-primary-600 hover:bg-primary-700 rounded-xl transition-colors"
          @click="showMigrationDialog = true"
      >
        <ArrowsRightLeftIcon class="h-4 w-4" />
        存储迁移
      </button>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="p-6 flex justify-center">
      <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
    </div>

    <div v-else class="p-6 space-y-6">
      <!-- 默认存储方式选择 -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <label class="block text-sm font-medium text-gray-300 mb-2">原图默认存储</label>
          <div class="w-full">
            <BaseSelect
                v-model="form.storageId"
                :options="storageOptions"
                placeholder="选择默认存储"
                @update:modelValue="handleDefaultStorageChange"
            />
          </div>
          <p class="mt-1.5 text-xs text-gray-500">新上传的原图将使用此存储方式</p>
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-300 mb-2">缩略图默认存储</label>
          <div class="w-full">
            <BaseSelect
                v-model="form.thumbnailStorageId"
                :options="storageOptions"
                placeholder="选择缩略图存储"
                @update:modelValue="handleThumbnailStorageChange"
            />
          </div>
          <p class="mt-1.5 text-xs text-gray-500">新上传的缩略图将使用此存储方式</p>
        </div>
      </div>

      <!-- 存储配置选择 -->
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-3">存储配置</label>
        <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
          <button
              v-for="type in storageTypes"
              :key="type.value"
              @click="editingType = type.value"
              :class="[
              'p-4 rounded-xl border text-center transition-all duration-300 relative',
              editingType === type.value
                ? 'border-primary-500 bg-primary-500/10 text-primary-400'
                : 'border-white/10 bg-white/2 text-gray-400 hover:border-white/20 hover:bg-white/5'
            ]"
          >
            <!-- 默认标识 -->
            <span
                v-if="isDefaultStorage(type.value)"
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
          v-if="editingType === 'local' && form.localConfig"
          v-model:base-path="form.localConfig.base_path"
      />

      <!-- 阿里云盘配置 (多账号) -->
      <AliyunPanConfig
          v-if="editingType === 'aliyunpan'"
          :accounts="form.aliyunpanConfig || []"
          :global-config="form.aliyunpanGlobal"
          :user-infos="form.aliyunpan_user || []"
          :default-storage-id="form.storageId"
          @update:accounts="form.aliyunpanConfig = $event"
          @update:global-config="form.aliyunpanGlobal = $event"
          @account-added="handleAccountAdded"
          @account-removed="handleAccountRemoved"
          @set-default="handleSetDefault"
      />

      <!-- OSS 配置 (占位) -->
      <div v-if="editingType === 'oss'" class="p-4 text-center text-gray-500">
        阿里云 OSS 配置暂未实现
      </div>

      <!-- S3 配置 (多账号) -->
      <S3Config
          v-if="editingType === 's3'"
          :accounts="form.s3Config || []"
          :default-storage-id="form.storageId"
          @update:accounts="form.s3Config = $event"
          @account-added="handleS3AccountAdded"
          @account-updated="handleS3AccountUpdated"
          @account-removed="handleS3AccountRemoved"
          @set-default="handleSetDefault"
      />

      <!-- MinIO 配置 (占位) -->
      <div v-if="editingType === 'minio'" class="p-4 text-center text-gray-500">
        MinIO 配置暂未实现
      </div>
    </div>

    <!-- 存储迁移对话框 -->
    <StorageMigrationDialog
        :storage-options="storageOptions"
        :visible="showMigrationDialog"
        @close="showMigrationDialog = false"
    />
  </div>
</template>

<script setup lang="ts">
import {computed, onMounted, reactive, ref, watch} from 'vue'
import {type AliyunPanStorageConfig, type S3StorageConfig, settingsApi, type StorageConfigPO} from '@/api/settings'
import type {StorageId} from '@/api/storage'
import {parseStorageId} from '@/api/storage'
import {useDialogStore} from '@/stores/dialog'
import {ArrowsRightLeftIcon} from '@heroicons/vue/24/outline'
import LocalStorageConfig from './storage/LocalStorageConfig.vue'
import AliyunPanConfig from './storage/AliyunPanConfig.vue'
import S3Config from './storage/S3Config.vue'
import BaseSelect from '@/components/common/BaseSelect.vue'
import StorageMigrationDialog from './storage/StorageMigrationDialog.vue'

const dialogStore = useDialogStore()

// 定义 emits
const emit = defineEmits<{
  change: [hasChanges: boolean]
  saving: [isSaving: boolean]
}>()

const storageTypes = [
  { value: 'local', label: '本地存储', desc: '文件系统' },
  { value: 'aliyunpan', label: '阿里云盘', desc: '网盘存储' },
  { value: 's3', label: 'S3 存储', desc: 'AWS/MinIO/OSS' },
]

const loading = ref(true)
const saving = ref(false)
const showMigrationDialog = ref(false)


// 当前正在编辑的存储类型
const editingType = ref<string>('local')

const form = reactive<StorageConfigPO>({
  storageId: 'local',
  thumbnailStorageId: 'local',
  localConfig: {
    id: 'local',
    base_path: '',
  },
  aliyunpanConfig: [],
  aliyunpanGlobal: {
    download_chunk_size: 512,
    download_concurrency: 8,
  },
  aliyunpan_user: [],
  s3Config: [],
})

// 原始配置，用于对比是否有变化
const originalForm = reactive<StorageConfigPO>({
  storageId: 'local',
  thumbnailStorageId: 'local',
  localConfig: {
    id: 'local',
    base_path: '',
  },
  aliyunpanConfig: [],
  aliyunpanGlobal: {
    download_chunk_size: 512,
    download_concurrency: 8,
  },
  aliyunpan_user: [],
  s3Config: [],
})

// 监听表单变化
watch(
  () => form,
  () => {
    const hasChanges = JSON.stringify(form) !== JSON.stringify(originalForm)
    emit('change', hasChanges)
  },
  { deep: true }
)

// 判断是否是默认存储
function isDefaultStorage(type: string): boolean {
  if (type === 'local') {
    return form.storageId === 'local'
  }
  if (type === 'aliyunpan') {
    return form.storageId.startsWith('aliyunpan:')
  }
  if (type === 's3') {
    return form.storageId.startsWith('s3:')
  }
  return false
}

// 获取阿里云盘账号名称
function getAliyunPanAccountName(id: StorageId): string {
  const { accountId } = parseStorageId(id)
  const userInfo = form.aliyunpan_user?.find(u => u.user_id === accountId)
  return userInfo?.nick_name || accountId || '未知账号'
}

const storageOptions = computed(() => {
  const options = [
    { label: '本地存储', value: 'local' }
  ]

  if (form.aliyunpanConfig && form.aliyunpanConfig.length > 0) {
    form.aliyunpanConfig.forEach(account => {
      options.push({
        label: `阿里云盘 - ${getAliyunPanAccountName(account.id)}`,
        value: account.id
      })
    })
  }

  if (form.s3Config && form.s3Config.length > 0) {
    form.s3Config.forEach(account => {
      options.push({
        label: `S3 - ${account.name}`,
        value: account.id
      })
    })
  }

  return options
})

async function loadSettings() {
  loading.value = true
  try {
    const resp = await settingsApi.getByCategory('storage')
    const data = resp.data as StorageConfigPO

    Object.assign(form, {
      storageId: data.storageId || 'local',
      thumbnailStorageId: data.thumbnailStorageId || 'local',
      localConfig: data.localConfig || { id: 'local', base_path: '', url_prefix: '' },
      aliyunpanConfig: data.aliyunpanConfig || [],
      aliyunpanGlobal: data.aliyunpanGlobal || { download_chunk_size: 512, download_concurrency: 8 },
      aliyunpan_user: data.aliyunpan_user || [],
      s3Config: data.s3Config || [],
    })

    // 保存原始数据
    Object.assign(originalForm, JSON.parse(JSON.stringify(form)))
    // 加载完成后，数据已同步，通知父组件无未保存更改
    emit('change', false)

    // 初始化时，编辑类型默认为当前默认存储类型
    const { driver } = parseStorageId(form.storageId)
    editingType.value = driver
  } catch (error) {
    console.error('Failed to load storage settings:', error)
  } finally {
    loading.value = false
  }
}


async function handleDefaultStorageChange() {
  try {
    await settingsApi.setDefaultStorage(form.storageId)
    // 同步 originalForm，因为后端已经保存
    originalForm.storageId = form.storageId
    // 重新计算是否有未保存的更改
    emit('change', JSON.stringify(form) !== JSON.stringify(originalForm))
    dialogStore.alert({
      title: '成功',
      message: '默认存储已更新',
      type: 'success'
    })
  } catch (error: any) {
    dialogStore.alert({
      title: '错误',
      message: error.message || '设置默认存储失败',
      type: 'error'
    })
    // 重新加载以恢复正确状态
    await loadSettings()
  }
}

async function handleThumbnailStorageChange() {
  try {
    await settingsApi.setThumbnailDefaultStorage(form.thumbnailStorageId!)
    // 同步 originalForm，因为后端已经保存
    originalForm.thumbnailStorageId = form.thumbnailStorageId
    // 重新计算是否有未保存的更改
    emit('change', JSON.stringify(form) !== JSON.stringify(originalForm))
    dialogStore.alert({
      title: '成功',
      message: '缩略图默认存储已更新',
      type: 'success'
    })
  } catch (error: any) {
    dialogStore.alert({
      title: '错误',
      message: error.message || '设置缩略图默认存储失败',
      type: 'error'
    })
    // 重新加载以恢复正确状态
    await loadSettings()
  }
}

async function handleAccountAdded(account: AliyunPanStorageConfig) {
  try {
    await settingsApi.addStorage({
      type: 'aliyunpan',
      config: account,
    })
    await loadSettings()
    dialogStore.alert({
      title: '成功',
      message: '阿里云盘账号添加成功',
      type: 'success'
    })
  } catch (error: any) {
    dialogStore.alert({
      title: '错误',
      message: error.message || '添加账号失败',
      type: 'error'
    })
  }
}

async function handleAccountRemoved(id: StorageId) {
  try {
    await settingsApi.deleteStorage(id)
    await loadSettings()
    dialogStore.alert({
      title: '成功',
      message: '阿里云盘账号已删除',
      type: 'success'
    })
  } catch (error: any) {
    dialogStore.alert({
      title: '错误',
      message: error.message || '删除账号失败',
      type: 'error'
    })
  }
}

// S3 账号处理
async function handleS3AccountAdded(account: S3StorageConfig) {
  try {
    await settingsApi.addStorage({
      type: 's3',
      config: account,
    })
    await loadSettings()
    dialogStore.alert({
      title: '成功',
      message: 'S3 存储账号添加成功',
      type: 'success'
    })
  } catch (error: any) {
    dialogStore.alert({
      title: '错误',
      message: error.message || '添加 S3 账号失败',
      type: 'error'
    })
  }
}

async function handleS3AccountUpdated(account: S3StorageConfig) {
  try {
    await settingsApi.updateStorage(account.id, account)
    await loadSettings()
    dialogStore.alert({
      title: '成功',
      message: 'S3 存储账号更新成功',
      type: 'success'
    })
  } catch (error: any) {
    dialogStore.alert({
      title: '错误',
      message: error.message || '更新 S3 账号失败',
      type: 'error'
    })
  }
}

async function handleS3AccountRemoved(id: StorageId) {
  try {
    await settingsApi.deleteStorage(id)
    await loadSettings()
    dialogStore.alert({
      title: '成功',
      message: 'S3 存储账号已删除',
      type: 'success'
    })
  } catch (error: any) {
    dialogStore.alert({
      title: '错误',
      message: error.message || '删除 S3 账号失败',
      type: 'error'
    })
  }
}

async function handleSetDefault(id: StorageId) {
  form.storageId = id
  await handleDefaultStorageChange()
}

async function handleSave() {
  saving.value = true
  emit('saving', true)
  try {
    // 根据当前编辑的类型保存配置
    if (editingType.value === 'local' && form.localConfig) {
      const resp = await settingsApi.updateStorage('local', form.localConfig)
      const result = resp.data

      dialogStore.alert({
        title: '成功',
        message: result.message || '存储配置更新成功',
        type: 'success'
      })
    } else if (editingType.value === 'aliyunpan' && form.aliyunpanGlobal) {
      // 保存全局配置
      await settingsApi.updateGlobalConfig(form.aliyunpanGlobal)
      dialogStore.alert({
        title: '成功',
        message: '阿里云盘全局配置更新成功',
        type: 'success'
      })
    }
    
    // 更新原始数据
    Object.assign(originalForm, JSON.parse(JSON.stringify(form)))
    emit('change', false)
  } catch (error: any) {
    if (error.response?.status === 423) {
      dialogStore.alert({
        title: '配置被锁定',
        message: '迁移正在进行中，请等待完成后再修改配置',
        type: 'warning'
      })
    } else {
      dialogStore.alert({
        title: '错误',
        message: error.message || '更新存储配置失败',
        type: 'error'
      })
    }
  } finally {
    saving.value = false
    emit('saving', false)
  }
}

// 暴露 save 方法
function save() {
  return handleSave()
}

// 还原配置方法
function restore() {
  Object.assign(form, JSON.parse(JSON.stringify(originalForm)))
  emit('change', false)
}

defineExpose({
  save,
  restore
})

onMounted(() => {
  loadSettings()
})
</script>
