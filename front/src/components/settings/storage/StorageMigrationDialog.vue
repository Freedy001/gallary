<template>
  <Modal
      :model-value="visible"
      size="2xl"
      @close="$emit('close')"
      @update:model-value="(val) => !val && $emit('close')"
  >
    <template #header>
      <div class="flex items-center gap-2">
        <ArrowsRightLeftIcon class="h-5 w-5 text-primary-400"/>
        <h3 class="text-xl font-semibold text-white">创建存储迁移任务</h3>
      </div>
    </template>

    <div class="space-y-5">
      <!-- 迁移类型 -->
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-2">迁移类型</label>
        <div class="grid grid-cols-2 gap-3">
          <button
              v-for="type in migrationTypes"
              :key="type.value"
              :class="[
                'p-3 rounded-xl border text-center transition-all duration-300',
                form.migration_type === type.value
                  ? 'border-primary-500 bg-primary-500/10 text-primary-400'
                  : 'border-white/10 bg-white/2 text-gray-400 hover:border-white/20'
              ]"
              @click="form.migration_type = type.value"
          >
            <div class="text-sm font-medium">{{ type.label }}</div>
            <div class="text-xs mt-0.5 opacity-60">{{ type.desc }}</div>
          </button>
        </div>
      </div>

      <!-- 源存储 -->
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-2">源存储</label>
        <BaseSelect
            v-model="form.source_storage_id"
            :options="storageOptions"
            placeholder="选择源存储"
        />
      </div>

      <!-- 目标存储 -->
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-2">目标存储</label>
        <BaseSelect
            v-model="form.target_storage_id"
            :options="targetStorageOptions"
            placeholder="选择目标存储"
        />
      </div>

      <!-- 过滤条件 -->
      <div class="rounded-xl bg-white/5 border border-white/10 p-4">
        <div class="flex items-center justify-between mb-3">
          <label class="text-sm font-medium text-gray-300">过滤条件（可选）</label>
          <button
              v-if="hasActiveFilters"
              class="text-xs text-gray-500 hover:text-gray-300 transition-colors"
              @click="clearFilters"
          >
            清除过滤
          </button>
        </div>

        <div class="space-y-4">
          <!-- 相册筛选 -->
          <div>
            <label class="block text-xs font-medium text-gray-400 mb-2">选择相册</label>
            <div class="relative">
              <input
                  v-model="albumSearchQuery"
                  class="w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm text-white transition-colors placeholder:text-gray-600 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500/50"
                  placeholder="搜索或选择相册..."
                  type="text"
                  @focus="albumDropdownOpen = true"
              />

              <!-- 已选中的相册标签 -->
              <div v-if="selectedAlbums.length > 0" class="flex flex-wrap gap-2 mt-2">
                <span
                    v-for="album in selectedAlbums"
                    :key="album.id"
                    class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-primary-500/20 text-primary-300 border border-primary-500/30"
                >
                  {{ album.name }}
                  <button
                      class="hover:text-white transition-colors"
                      @click="removeAlbum(album.id)"
                  >
                    <XMarkIcon class="h-3 w-3"/>
                  </button>
                </span>
              </div>

              <!-- 相册下拉列表 -->
              <div
                  v-if="albumDropdownOpen && filteredAlbums.length > 0"
                  class="absolute z-10 mt-2 w-full max-h-48 overflow-y-auto rounded-lg border border-white/10 bg-gray-800 shadow-lg"
              >
                <button
                    v-for="album in filteredAlbums"
                    :key="album.id"
                    :class="form.filter?.album_ids?.includes(album.id) ? 'text-primary-300' : 'text-gray-300'"
                    class="flex items-center justify-between w-full px-3 py-2 text-sm text-left hover:bg-white/5 transition-colors"
                    @click="toggleAlbum(album)"
                >
                  <span>{{ album.name }}</span>
                  <CheckIcon v-if="form.filter?.album_ids?.includes(album.id)" class="h-4 w-4 text-primary-400"/>
                </button>
              </div>

              <!-- 点击外部关闭下拉 -->
              <div
                  v-if="albumDropdownOpen"
                  class="fixed inset-0 z-0"
                  @click="albumDropdownOpen = false"
              />
            </div>
          </div>

          <!-- 日期范围筛选 -->
          <div>
            <label class="block text-xs font-medium text-gray-400 mb-2">拍摄日期范围</label>
            <div class="flex items-center gap-2">
              <input
                  v-model="form.filter.start_date"
                  class="flex-1 rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm text-white transition-colors focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500/50"
                  type="date"
              />
              <span class="text-xs text-gray-500">至</span>
              <input
                  v-model="form.filter.end_date"
                  class="flex-1 rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm text-white transition-colors focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500/50"
                  type="date"
              />
            </div>
          </div>

          <!-- 文件大小筛选 -->
          <div>
            <label class="block text-xs font-medium text-gray-400 mb-2">文件大小范围</label>
            <div class="flex items-center gap-2">
              <div class="relative flex-1">
                <input
                    v-model.number="minFileSizeMB"
                    class="w-full rounded-lg border border-white/10 bg-white/5 pl-3 pr-8 py-2 text-sm text-white transition-colors placeholder:text-gray-600 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500/50 no-spin-button"
                    min="0"
                    placeholder="0"
                    step="0.1"
                    type="number"
                />
                <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-gray-500">MB</span>
              </div>
              <span class="text-gray-500 text-sm">至</span>
              <div class="relative flex-1">
                <input
                    v-model.number="maxFileSizeMB"
                    class="w-full rounded-lg border border-white/10 bg-white/5 pl-3 pr-8 py-2 text-sm text-white transition-colors placeholder:text-gray-600 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500/50 no-spin-button"
                    min="0"
                    placeholder="∞"
                    step="0.1"
                    type="number"
                />
                <span class="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-gray-500">MB</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 选项 -->
      <div class="flex items-center gap-2">
        <input
            id="deleteSource"
            v-model="form.delete_source_after_migration"
            class="h-4 w-4 rounded border-gray-600 bg-gray-700 text-primary-500 focus:ring-primary-500 focus:ring-offset-gray-900"
            type="checkbox"
        />
        <label class="text-sm text-gray-300" for="deleteSource">迁移完成后删除源文件</label>
      </div>

      <!-- 预览结果 -->
      <div v-if="preview" class="p-4 rounded-xl bg-white/5 border border-white/10">
        <div class="flex items-center gap-6 text-sm">
          <div class="text-gray-400">
            待迁移文件: <span class="text-white font-medium">{{ preview.files_count }} 个</span>
          </div>
          <div v-if="preview.total_size && preview.total_size!==0" class="text-gray-400">文件大小:
            <span class="text-white font-medium">{{ formatFileSize(preview.total_size) }}</span>
          </div>
        </div>
      </div>

      <!-- 错误提示 -->
      <div v-if="error" class="p-3 rounded-xl bg-red-500/10 border border-red-500/20 text-sm text-red-400">
        {{ error }}
      </div>
    </div>

    <template #footer>
      <div class="flex gap-3 w-full">
        <button
            class="flex-1 inline-flex justify-center rounded-xl border border-white/10 px-4 py-2 text-sm font-medium text-gray-300 hover:bg-white/5 transition-colors"
            type="button"
            @click="$emit('close')"
        >
          取消
        </button>
        <button
            :disabled="!canSubmit || loading"
            class="flex-1 inline-flex justify-center rounded-xl border border-transparent bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            type="button"
            @click="handleSubmit"
        >
          <span v-if="loading" class="flex items-center gap-2">
            <div class="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"></div>
            处理中...
          </span>
          <span v-else>开始迁移</span>
        </button>
      </div>
    </template>
  </Modal>
</template>

<script lang="ts" setup>
import {computed, reactive, ref, watch} from 'vue'
import {ArrowsRightLeftIcon, CheckIcon, XMarkIcon} from '@heroicons/vue/24/outline'
import {useDialogStore} from '@/stores/dialog'
import {useAlbumStore} from '@/stores/album'
import {storageMigrationApi} from '@/api/storageMigration'
import BaseSelect from '@/components/common/BaseSelect.vue'
import Modal from '@/components/common/Modal.vue'
import type {CreateMigrationRequest, MigrationPreview, MigrationType} from '@/types/migration'
import type {Album} from '@/types'

interface StorageOption {
  label: string
  value: string
}

const props = defineProps<{
  visible: boolean
  storageOptions: StorageOption[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const dialogStore = useDialogStore()
const albumStore = useAlbumStore()

const loading = ref(false)
const error = ref('')
const preview = ref<MigrationPreview | null>(null)

// 相册筛选相关
const albumSearchQuery = ref('')
const albumDropdownOpen = ref(false)
const selectedAlbumsMap = ref<Map<number, Album>>(new Map())

// 文件大小筛选（MB单位）
const minFileSizeMB = ref<number | undefined>()
const maxFileSizeMB = ref<number | undefined>()

const migrationTypes: { value: MigrationType; label: string; desc: string }[] = [
  {value: 'original', label: '原图', desc: '迁移原始图片文件'},
  {value: 'thumbnail', label: '缩略图', desc: '迁移缩略图文件'},
]

const form = reactive<CreateMigrationRequest>({
  migration_type: 'original',
  source_storage_id: '',
  target_storage_id: '',
  delete_source_after_migration: true,
  filter: {
    album_ids: [],
    start_date: undefined,
    end_date: undefined,
    min_file_size: undefined,
    max_file_size: undefined,
  },
})

// 目标存储选项（排除源存储）
const targetStorageOptions = computed(() => {
  return props.storageOptions.filter(opt => opt.value !== form.source_storage_id)
})

// 已选中的相册列表
const selectedAlbums = computed(() => {
  return Array.from(selectedAlbumsMap.value.values())
})

// 过滤后的相册列表
const filteredAlbums = computed(() => {
  const query = albumSearchQuery.value.toLowerCase().trim()
  if (!query) return albumStore.albums
  return albumStore.albums.filter(album =>
      album.name.toLowerCase().includes(query)
  )
})

// 是否有活动的过滤条件
const hasActiveFilters = computed(() => {
  return (form.filter?.album_ids && form.filter.album_ids.length > 0) ||
      !!form.filter?.start_date ||
      !!form.filter?.end_date ||
      minFileSizeMB.value !== undefined ||
      maxFileSizeMB.value !== undefined
})

// 能否提交
const canSubmit = computed(() => {
  return form.source_storage_id &&
      form.target_storage_id &&
      form.source_storage_id !== form.target_storage_id &&
      preview.value &&
      preview.value.files_count > 0
})

// 监听文件大小输入，转换为字节
watch([minFileSizeMB, maxFileSizeMB], ([min, max]) => {
  if (!form.filter) {
    form.filter = {}
  }
  form.filter.min_file_size = min !== undefined ? Math.round(min * 1024 * 1024) : undefined
  form.filter.max_file_size = max !== undefined ? Math.round(max * 1024 * 1024) : undefined
})

// 监听表单变化，自动预览
watch(
    () => [
      form.migration_type,
      form.source_storage_id,
      form.target_storage_id,
      form.filter?.album_ids?.length,
      form.filter?.start_date,
      form.filter?.end_date,
      form.filter?.min_file_size,
      form.filter?.max_file_size,
    ],
    async () => {
      if (form.source_storage_id && form.target_storage_id && form.source_storage_id !== form.target_storage_id) {
        await loadPreview()
      } else {
        preview.value = null
      }
    },
    {deep: true}
)

// 重置表单
watch(() => props.visible, async (visible) => {
  if (visible) {
    form.migration_type = 'original'
    form.source_storage_id = ''
    form.target_storage_id = ''
    form.delete_source_after_migration = true
    form.filter = {
      album_ids: [],
      start_date: undefined,
      end_date: undefined,
      min_file_size: undefined,
      max_file_size: undefined,
    }
    selectedAlbumsMap.value.clear()
    albumSearchQuery.value = ''
    minFileSizeMB.value = undefined
    maxFileSizeMB.value = undefined
    preview.value = null
    error.value = ''

    // 加载相册列表
    if (albumStore.albums.length === 0) {
      await albumStore.refreshAlbums()
    }
  }
})

async function loadPreview() {
  try {
    error.value = ''
    loading.value = true
    // 清理空的过滤条件
    const requestData: Partial<CreateMigrationRequest> = {...form}
    if (requestData.filter) {
      const filter = requestData.filter
      if (!filter.album_ids?.length && !filter.start_date && !filter.end_date &&
          !filter.min_file_size && !filter.max_file_size) {
        delete requestData.filter
      } else {
        // 清理未设置的字段
        if (!filter.album_ids?.length) delete filter.album_ids
        if (!filter.start_date) delete filter.start_date
        if (!filter.end_date) delete filter.end_date
        if (!filter.min_file_size) delete filter.min_file_size
        if (!filter.max_file_size) delete filter.max_file_size
      }
    }
    const response = await storageMigrationApi.previewMigration(requestData as CreateMigrationRequest)
    preview.value = response.data
  } catch (err: any) {
    error.value = err.message || '预览失败'
    preview.value = null
  } finally {
    loading.value = false
  }
}

function toggleAlbum(album: Album) {
  if (!form.filter) {
    form.filter = {album_ids: []}
  }
  if (!form.filter.album_ids) {
    form.filter.album_ids = []
  }

  const index = form.filter.album_ids.indexOf(album.id)
  if (index > -1) {
    form.filter.album_ids.splice(index, 1)
    selectedAlbumsMap.value.delete(album.id)
  } else {
    form.filter.album_ids.push(album.id)
    selectedAlbumsMap.value.set(album.id, album)
  }
}

function removeAlbum(albumId: number) {
  if (form.filter?.album_ids) {
    const index = form.filter.album_ids.indexOf(albumId)
    if (index > -1) {
      form.filter.album_ids.splice(index, 1)
    }
  }
  selectedAlbumsMap.value.delete(albumId)
}

function clearFilters() {
  if (form.filter) {
    form.filter.album_ids = []
    form.filter.start_date = undefined
    form.filter.end_date = undefined
    form.filter.min_file_size = undefined
    form.filter.max_file_size = undefined
  }
  selectedAlbumsMap.value.clear()
  minFileSizeMB.value = undefined
  maxFileSizeMB.value = undefined
}

async function handleSubmit() {
  if (!canSubmit.value) return

  loading.value = true
  error.value = ''

  try {
    // 清理空的过滤条件
    const requestData: Partial<CreateMigrationRequest> = {...form}
    if (requestData.filter) {
      const filter = requestData.filter
      if (!filter.album_ids?.length && !filter.start_date && !filter.end_date &&
          !filter.min_file_size && !filter.max_file_size) {
        delete requestData.filter
      } else {
        // 清理未设置的字段
        if (!filter.album_ids?.length) delete filter.album_ids
        if (!filter.start_date) delete filter.start_date
        if (!filter.end_date) delete filter.end_date
        if (!filter.min_file_size) delete filter.min_file_size
        if (!filter.max_file_size) delete filter.max_file_size
      }
    }

    await storageMigrationApi.createMigration(requestData as CreateMigrationRequest)
    dialogStore.alert({
      title: '成功',
      message: '迁移任务已创建',
      type: 'success'
    })
    emit('close')
  } catch (err: any) {
    error.value = err.message || '创建迁移任务失败'
  } finally {
    loading.value = false
  }
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const k = 1024
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(2)} ${units[i]}`
}
</script>

<style scoped>
/* Remove spin buttons for Chrome, Safari, Edge, Opera */
.no-spin-button::-webkit-outer-spin-button,
.no-spin-button::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}

/* Remove spin buttons for Firefox */
.no-spin-button {
  -moz-appearance: textfield;
}
</style>
