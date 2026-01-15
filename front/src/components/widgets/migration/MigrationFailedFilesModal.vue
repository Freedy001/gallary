<template>
  <Modal
      ref="modal"
      :closable="true"
      :model-value="visible"
      size="xl"
      @close="$emit('close')"
  >
    <template #header>
      <div class="flex items-center gap-3">
        <div class="w-2.5 h-2.5 rounded-full shadow-[0_0_8px_currentColor] bg-red-500"></div>
        <h3 class="text-xl font-semibold text-white tracking-wide">
          失败文件详情
        </h3>
      </div>
    </template>

    <div class="space-y-6">
      <!-- 统计信息 -->
      <div class="grid grid-cols-1 gap-4">
        <div class="glass-stat-card bg-red-500/10 border-red-500/20">
          <div class="text-3xl font-bold text-red-400 mb-1">
            {{ totalCount }}
          </div>
          <div class="text-xs text-red-200/70 font-medium tracking-wider uppercase">失败文件</div>
        </div>
      </div>

      <!-- 操作栏 -->
      <div class="flex items-center justify-between pb-2 border-b border-white/10">
        <span class="text-sm font-medium text-white/80 flex items-center gap-2">
          <ExclamationCircleIcon class="w-4 h-4 text-red-400"/>
          失败文件列表
        </span>
        <button
            v-if="hasFailedFiles"
            :disabled="migrationStore.loading"
            class="flex items-center gap-1.5 px-5 py-3 text-sm rounded-xl border border-white/10 bg-white/5 font-medium text-white hover:bg-white/10 transition-colors disabled:opacity-50"
            @click="retryAll"
        >
          <ArrowPathIcon :class="{ 'animate-spin': migrationStore.loading }" class="h-4 w-4"/>
          全部重试
        </button>
      </div>

      <!-- 失败文件列表 -->
      <div ref="scrollContainer" class="min-h-[300px] max-h-[50vh] overflow-y-auto pr-2 custom-scrollbar"
           @scroll="handleScroll">
        <div v-if="initialLoading" class="flex flex-col items-center justify-center py-12 space-y-3">
          <div class="animate-spin rounded-full h-8 w-8 border-2 border-blue-500 border-t-transparent"></div>
          <span class="text-sm text-white/50">加载中...</span>
        </div>

        <div v-else-if="!hasFailedFiles"
             class="flex flex-col items-center justify-center py-12 text-white/30 space-y-3">
          <CheckCircleIcon class="h-16 w-16 opacity-50"/>
          <span class="text-sm">暂无失败记录</span>
        </div>

        <div v-else class="space-y-3">
          <div
              v-for="record in failedItems"
              :key="record.id"
              class="glass-list-item group"
          >
            <!-- 缩略图 -->
            <div
                class="w-16 h-16 shrink-0 rounded-lg overflow-hidden bg-black/40 border border-white/10 relative flex items-center justify-center">
              <img
                  v-if="record.thumb_url"
                  :alt="record.image_name"
                  :src="record.thumb_url"
                  class="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110"
                  @error="(e: Event) => (e.target as HTMLImageElement).style.display = 'none'"
              />
              <div v-else class="w-full h-full flex items-center justify-center text-white/20">
                <PhotoIcon class="h-8 w-8"/>
              </div>
            </div>

            <!-- 信息 -->
            <div class="flex-1 min-w-0 px-4">
              <div :title="record.image_name || `图片 #${record.image_id}`" class="text-sm font-medium text-white/90 truncate mb-1">
                {{ record.image_name || `图片 #${record.image_id}` }}
              </div>
              <div v-if="record.error_msg"
                   class="text-xs text-red-300/90 bg-red-500/10 rounded px-2 py-1 border border-red-500/10 whitespace-pre-wrap break-words">
                {{ record.error_msg }}
              </div>
              <div class="text-[10px] text-white/40 mt-1.5 font-mono">
                {{ formatTime(record.created_at) }}
              </div>
            </div>
          </div>

          <!-- 加载更多指示器 -->
          <div v-if="loadingMore" class="flex justify-center py-4">
            <div class="animate-spin rounded-full h-6 w-6 border-2 border-blue-500 border-t-transparent"></div>
          </div>

          <!-- 已加载全部 -->
          <div v-else-if="!hasMore && failedItems.length > 0" class="text-center py-4 text-xs text-white/40">
            已加载全部
          </div>
        </div>
      </div>
    </div>
  </Modal>
</template>

<script lang="ts" setup>
import {computed, nextTick, ref, watch} from 'vue'
import {ArrowPathIcon, CheckCircleIcon, ExclamationCircleIcon, PhotoIcon} from '@heroicons/vue/24/outline'
import {useMigrationStore} from '@/stores/migration.ts'
import {storageMigrationApi} from '@/api/storageMigration'
import Modal from '@/components/common/Modal.vue'
import {useDialogStore} from "@/stores/dialog.ts"
import type {MigrationFileRecord} from "@/types/migration.ts";

const props = defineProps<{
  visible: boolean
  taskId: number | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const migrationStore = useMigrationStore()
const dialogStore = useDialogStore()

const modal = ref<InstanceType<typeof Modal> | null>(null)
const scrollContainer = ref<HTMLElement | null>(null)
const initialLoading = ref(false)
const loadingMore = ref(false)
const currentPage = ref(1)
const pageSize = 20
const failedItems = ref<MigrationFileRecord[]>([])
const hasMore = ref(true)
const totalCount = ref(0) // 总数

const hasFailedFiles = computed(() => failedItems.value.length > 0)

function formatTime(timeStr: string): string {
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// 初始加载数据
async function initLoadData() {
  if (!props.taskId) return

  initialLoading.value = true
  currentPage.value = 1
  failedItems.value = []
  hasMore.value = true

  try {
    const res = await storageMigrationApi.getFailedFileRecords(props.taskId, 1, pageSize)
    failedItems.value = res.data.items || []
    totalCount.value = res.data.total || 0

    // 检查是否还有更多数据
    hasMore.value = failedItems.value.length < totalCount.value
  } finally {
    initialLoading.value = false
  }
}

// 加载更多数据
async function loadMore() {
  if (loadingMore.value || !hasMore.value || !props.taskId) return

  loadingMore.value = true
  currentPage.value++

  try {
    const res = await storageMigrationApi.getFailedFileRecords(props.taskId, currentPage.value, pageSize)
    const newItems = res.data.items || []
    failedItems.value.push(...newItems)
    totalCount.value = res.data.total || 0

    // 检查是否还有更多数据
    hasMore.value = failedItems.value.length < totalCount.value
  } finally {
    loadingMore.value = false
  }
}

// 滚动事件处理
function handleScroll(event: Event) {
  const target = event.target as HTMLElement
  const scrollTop = target.scrollTop
  const scrollHeight = target.scrollHeight
  const clientHeight = target.clientHeight

  // 距离底部 100px 时触发加载
  if (scrollHeight - scrollTop - clientHeight < 100) {
    loadMore()
  }
}

// 全部重试
async function retryAll() {
  if (!props.taskId) return

  try {
    emit('close')
    await storageMigrationApi.retryFailedFiles(props.taskId)
    dialogStore.notify({
      title: '成功',
      message: '重试请求发送成功!',
      type: 'success'
    })
  } catch (error) {
    dialogStore.notify({
      title: '错误',
      message: (error as Error)?.message || '重试失败',
      type: 'error'
    })
  }
}

// 监听可见性变化
watch(() => props.visible, (newVal) => {
  if (newVal && props.taskId) {
    initLoadData()
    // 滚动到顶部
    nextTick(() => {
      if (scrollContainer.value) {
        scrollContainer.value.scrollTop = 0
      }
    })
  }
}, { immediate: true })

// 监听任务ID变化
watch(() => props.taskId, (newVal) => {
  if (props.visible && newVal) {
    initLoadData()
  }
})
</script>

<style scoped>
.glass-stat-card {
  padding: 1.25rem;
  border-radius: 1rem;
  text-align: center;
  border-width: 1px;
  background-image: linear-gradient(145deg, rgba(255, 255, 255, 0.05) 0%, rgba(255, 255, 255, 0.01) 100%);
}

.glass-list-item {
  display: flex;
  align-items: flex-start;
  padding: 0.75rem;
  border-radius: 0.75rem;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.05);
  transition: all 0.2s ease;
}

.glass-list-item:hover {
  background: rgba(255, 255, 255, 0.06);
  border-color: rgba(255, 255, 255, 0.1);
}

.custom-scrollbar::-webkit-scrollbar {
  width: 4px;
}

.custom-scrollbar::-webkit-scrollbar-track {
  background: rgba(255, 255, 255, 0.02);
  border-radius: 2px;
}

.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 2px;
}

.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.2);
}
</style>
