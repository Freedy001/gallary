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
        <div class="w-full md:w-64">
          <BaseSelect
              v-model="form.storageId"
              :options="storageOptions"
              @update:modelValue="handleDefaultStorageChange"
              placeholder="选择默认存储"
          />
        </div>
        <p class="mt-1.5 text-xs text-gray-500">新上传的图片将使用此存储方式</p>
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
                : 'border-white/10 bg-white/[0.02] text-gray-400 hover:border-white/20 hover:bg-white/5'
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
          v-model:url-prefix="form.localConfig.url_prefix"
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

      <!-- S3 配置 (占位) -->
      <div v-if="editingType === 's3'" class="p-4 text-center text-gray-500">
        AWS S3 配置暂未实现
      </div>

      <!-- MinIO 配置 (占位) -->
      <div v-if="editingType === 'minio'" class="p-4 text-center text-gray-500">
        MinIO 配置暂未实现
      </div>

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
import { ref, reactive, onMounted, computed } from 'vue'
import { settingsApi, type StorageConfigPO, type AliyunPanStorageConfig, type AliyunPanGlobalConfig } from '@/api/settings'
import type { StorageId } from '@/api/storage'
import { parseStorageId } from '@/api/storage'
import { useDialogStore } from '@/stores/dialog'
import LocalStorageConfig from './storage/LocalStorageConfig.vue'
import AliyunPanConfig from './storage/AliyunPanConfig.vue'
import MigrationProgress from './MigrationProgress.vue'
import BaseSelect from '@/components/common/BaseSelect.vue'

const dialogStore = useDialogStore()

const storageTypes = [
  { value: 'local', label: '本地存储', desc: '文件系统' },
  { value: 'aliyunpan', label: '阿里云盘', desc: '网盘存储' },
  // { value: 'oss', label: '阿里云 OSS', desc: '对象存储' },
  // { value: 's3', label: 'AWS S3', desc: '云存储' },
  // { value: 'minio', label: 'MinIO', desc: '自托管' },
]

const loading = ref(true)
const saving = ref(false)

// 迁移进度组件引用
const migrationProgressRef = ref<InstanceType<typeof MigrationProgress> | null>(null)

// 是否正在迁移
const isMigrating = computed(() => {
  return migrationProgressRef.value?.isMigrating ?? false
})

// 当前正在编辑的存储类型
const editingType = ref<string>('local')

const form = reactive<StorageConfigPO>({
  storageId: 'local',
  localConfig: {
    id: 'local',
    base_path: '',
    url_prefix: '',
  },
  aliyunpanConfig: [],
  aliyunpanGlobal: {
    download_chunk_size: 512,
    download_concurrency: 8,
  },
  aliyunpan_user: [],
})

// 判断是否是默认存储
function isDefaultStorage(type: string): boolean {
  if (type === 'local') {
    return form.storageId === 'local'
  }
  if (type === 'aliyunpan') {
    return form.storageId.startsWith('aliyunpan:')
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

  return options
})

async function loadSettings() {
  loading.value = true
  try {
    const resp = await settingsApi.getByCategory('storage')
    const data = resp.data as StorageConfigPO

    Object.assign(form, {
      storageId: data.storageId || 'local',
      localConfig: data.localConfig || { id: 'local', base_path: '', url_prefix: '' },
      aliyunpanConfig: data.aliyunpanConfig || [],
      aliyunpanGlobal: data.aliyunpanGlobal || { download_chunk_size: 512, download_concurrency: 8 },
      aliyunpan_user: data.aliyunpan_user || [],
    })

    // 初始化时，编辑类型默认为当前默认存储类型
    const { driver } = parseStorageId(form.storageId)
    editingType.value = driver
  } catch (error) {
    console.error('Failed to load storage settings:', error)
  } finally {
    loading.value = false
  }
}

function handleMigrationCompleted() {
  loadSettings()
}

async function handleDefaultStorageChange() {
  try {
    await settingsApi.setDefaultStorage(form.storageId)
    await dialogStore.alert({
      title: '成功',
      message: '默认存储已更新',
      type: 'success'
    })
  } catch (error: any) {
    await dialogStore.alert({
      title: '错误',
      message: error.message || '设置默认存储失败',
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
    await dialogStore.alert({
      title: '成功',
      message: '阿里云盘账号添加成功',
      type: 'success'
    })
  } catch (error: any) {
    await dialogStore.alert({
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
    await dialogStore.alert({
      title: '成功',
      message: '阿里云盘账号已删除',
      type: 'success'
    })
  } catch (error: any) {
    await dialogStore.alert({
      title: '错误',
      message: error.message || '删除账号失败',
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
  try {
    // 根据当前编辑的类型保存配置
    if (editingType.value === 'local' && form.localConfig) {
      const resp = await settingsApi.updateStorage('local', form.localConfig)
      const result = resp.data

      if (result.needs_migration) {
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
    } else if (editingType.value === 'aliyunpan' && form.aliyunpanGlobal) {
      // 保存全局配置
      await settingsApi.updateGlobalConfig(form.aliyunpanGlobal)
      await dialogStore.alert({
        title: '成功',
        message: '阿里云盘全局配置更新成功',
        type: 'success'
      })
    }
  } catch (error: any) {
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
