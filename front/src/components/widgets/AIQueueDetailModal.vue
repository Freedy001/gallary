<template>
  <Modal
    :model-value="visible"
    size="xl"
    :closable="true"
    @update:model-value="$emit('close')"
  >
    <template #header>
      <div class="flex items-center gap-3">
        <div class="w-2.5 h-2.5 rounded-full shadow-[0_0_8px_currentColor]" :class="getStatusDotClass()"></div>
        <h3 class="text-xl font-semibold text-white tracking-wide">
          {{ getQueueDisplayName(queue) }}
        </h3>
      </div>
    </template>

    <div class="space-y-6">
      <!-- 统计信息 -->
      <div class="grid grid-cols-2 gap-4">
        <div class="glass-stat-card bg-yellow-500/10 border-yellow-500/20">
          <div class="text-3xl font-bold text-yellow-400 mb-1">{{ queue.pending_count }}</div>
          <div class="text-xs text-yellow-200/70 font-medium tracking-wider uppercase">待处理</div>
        </div>
        <div class="glass-stat-card bg-red-500/10 border-red-500/20">
          <div class="text-3xl font-bold text-red-400 mb-1">{{ queue.failed_count }}</div>
          <div class="text-xs text-red-200/70 font-medium tracking-wider uppercase">失败</div>
        </div>
      </div>

      <!-- 操作栏 -->
      <div class="flex items-center justify-between pb-2 border-b border-white/10">
        <span class="text-sm font-medium text-white/80 flex items-center gap-2">
          <ExclamationCircleIcon class="w-4 h-4 text-red-400" />
          失败任务列表
        </span>
        <button
            v-if="queue.failed_count > 0"
            @click="retryAll"
            :disabled="aiStore.loading"
            class="glass-button-primary flex items-center gap-1.5 px-3 py-1.5 text-sm"
        >
          <ArrowPathIcon class="h-4 w-4" :class="{ 'animate-spin': aiStore.loading }" />
          全部重试
        </button>
      </div>

      <!-- 失败图片列表 -->
      <div ref="scrollContainer" class="min-h-[300px] max-h-[50vh] overflow-y-auto pr-2 custom-scrollbar" @scroll="handleScroll">
        <div v-if="initialLoading" class="flex flex-col items-center justify-center py-12 space-y-3">
          <div class="animate-spin rounded-full h-8 w-8 border-2 border-primary-500 border-t-transparent"></div>
          <span class="text-sm text-white/50">加载中...</span>
        </div>

        <div v-else-if="failedItems.length === 0" class="flex flex-col items-center justify-center py-12 text-white/30 space-y-3">
          <CheckCircleIcon class="h-16 w-16 opacity-50" />
          <span class="text-sm">暂无失败任务</span>
        </div>

        <div v-else class="space-y-3">
          <div
              v-for="item in failedItems"
              :key="item.id"
              class="glass-list-item group"
          >
            <!-- 缩略图/图标 -->
            <div class="w-16 h-16 shrink-0 rounded-lg overflow-hidden bg-black/40 border border-white/10 relative flex items-center justify-center">
              <!-- 标签类型：显示标签图标 -->
              <template v-if="item.item_type === 'tag-embedding'">
                <div class="w-full h-full flex items-center justify-center bg-gradient-to-br from-primary-500/20 to-purple-500/20">
                  <TagIcon class="h-8 w-8 text-primary-400" />
                </div>
              </template>
              <!-- 图片类型：显示缩略图 -->
              <template v-else>
                <img
                    v-if="item.item_thumb"
                    :alt="item.item_id+''"
                    :src="item.item_thumb"
                    class="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110"
                />
                <div v-else class="w-full h-full flex items-center justify-center text-white/20">
                  <PhotoIcon class="h-8 w-8" />
                </div>
              </template>
            </div>

            <!-- 信息 -->
            <div class="flex-1 min-w-0 px-4">
              <div class="text-sm font-medium text-white/90 truncate mb-1">
                {{ item.item_name || `#${item.item_id}` }}
              </div>
              <div v-if="item.error" class="text-xs text-red-300/90 bg-red-500/10 rounded px-2 py-1 border border-red-500/10 whitespace-pre-wrap break-words">
                {{ item.error }}
              </div>
              <div class="text-[10px] text-white/40 mt-1.5 font-mono">
                {{ formatTime(item.created_at) }}
              </div>
            </div>

            <!-- 操作按钮 -->
            <div class="shrink-0 flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity duration-200">
              <button
                  @click="retryItem(item.id)"
                  :disabled="aiStore.loading"
                  class="glass-icon-btn text-primary-400 hover:text-primary-300 hover:bg-primary-500/20"
                  title="重试"
              >
                <ArrowPathIcon class="h-4 w-4" />
              </button>
              <button
                  @click="ignoreItem(item.id)"
                  :disabled="aiStore.loading"
                  class="glass-icon-btn text-gray-400 hover:text-white hover:bg-white/10"
                  title="忽略"
              >
                <XMarkIcon class="h-4 w-4" />
              </button>
            </div>
          </div>

          <!-- 加载更多指示器 -->
          <div v-if="loadingMore" class="flex justify-center py-4">
            <div class="animate-spin rounded-full h-6 w-6 border-2 border-primary-500 border-t-transparent"></div>
          </div>

          <!-- 已加载全部 -->
          <div v-else-if="hasMore === false && failedItems.length > 0" class="text-center py-4 text-xs text-white/40">
            已加载全部
          </div>
        </div>
      </div>
    </div>
  </Modal>
</template>

<script setup lang="ts">
import {nextTick, ref, watch} from 'vue'
import {
  ArrowPathIcon,
  CheckCircleIcon,
  ExclamationCircleIcon,
  PhotoIcon,
  TagIcon,
  XMarkIcon
} from '@heroicons/vue/24/outline'
import {useAIStore} from '@/stores/ai.ts'
import type {AIQueueInfo, AITaskItemInfo} from '@/types/ai.ts'
import {getQueueDisplayName} from '@/types/ai.ts'
import Modal from '@/components/common/Modal.vue'

const props = defineProps<{
  queue: AIQueueInfo
  visible: boolean
}>()

const aiStore = useAIStore()

const scrollContainer = ref<HTMLElement | null>(null)
const initialLoading = ref(false)
const loadingMore = ref(false)
const currentPage = ref(1)
const pageSize = 20
const failedItems = ref<AITaskItemInfo[]>([])
const hasMore = ref(true)

// 获取状态点样式
function getStatusDotClass(): string {
  if (props.queue.status === 'processing') {
    return 'bg-primary-500 animate-pulse'
  }
  if (props.queue.pending_count > 0) {
    return 'bg-yellow-500'
  }
  if (props.queue.failed_count > 0) {
    return 'bg-red-500'
  }
  return 'bg-green-500'
}


// 格式化时间
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
  initialLoading.value = true
  currentPage.value = 1
  failedItems.value = []
  hasMore.value = true

  try {
    await aiStore.fetchQueueDetail(props.queue.id, 1, pageSize)
    failedItems.value = aiStore.queueDetail?.failed_items || []

    // 检查是否还有更多数据
    const total = aiStore.queueDetail?.total_failed || 0
    hasMore.value = failedItems.value.length < total
  } finally {
    initialLoading.value = false
  }
}

// 加载更多数据
async function loadMore() {
  if (loadingMore.value || !hasMore.value) return

  loadingMore.value = true
  currentPage.value++

  try {
    await aiStore.fetchQueueDetail(props.queue.id, currentPage.value, pageSize)
    const newItems = aiStore.queueDetail?.failed_items || []
    failedItems.value.push(...newItems)

    // 检查是否还有更多数据
    const total = aiStore.queueDetail?.total_failed || 0
    hasMore.value = failedItems.value.length < total
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

// 重试单个任务项
async function retryItem(taskItemId: number) {
  try {
    await aiStore.retryTaskItem(taskItemId)
    // 从列表中移除该项
    const index = failedItems.value.findIndex(item => item.id === taskItemId)
    if (index !== -1) {
      failedItems.value.splice(index, 1)
    }
  } catch (error) {
    console.error('重试失败:', error)
  }
}

// 忽略单个任务项
async function ignoreItem(taskItemId: number) {
  try {
    await aiStore.ignoreTaskItem(taskItemId)
    // 从列表中移除该项
    const index = failedItems.value.findIndex(item => item.id === taskItemId)
    if (index !== -1) {
      failedItems.value.splice(index, 1)
    }
  } catch (error) {
    console.error('忽略失败:', error)
  }
}

// 全部重试
async function retryAll() {
  try {
    await aiStore.retryQueueFailedImages(props.queue.id)
    // 重新加载数据
    await initLoadData()
  } catch (error) {
    console.error('全部重试失败:', error)
  }
}

// 监听可见性变化
watch(() => props.visible, (newVal) => {
  if (newVal) {
    initLoadData()
    // 滚动到顶部
    nextTick(() => {
      if (scrollContainer.value) {
        scrollContainer.value.scrollTop = 0
      }
    })
  }
}, { immediate: true })

// 监听队列变化
watch(() => props.queue.id, () => {
  if (props.visible) {
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
  background-image: linear-gradient(145deg, rgba(255,255,255,0.05) 0%, rgba(255,255,255,0.01) 100%);
}

.glass-button-primary {
  border-radius: 0.5rem;
  font-weight: 500;
  color: white;
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.8), rgba(99, 102, 241, 0.8));
  border: 1px solid rgba(255, 255, 255, 0.1);
  transition: all 0.2s ease;
  backdrop-filter: blur(4px);
}

.glass-button-primary:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
}

.glass-button-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
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

.glass-icon-btn {
  padding: 0.375rem;
  border-radius: 0.5rem;
  transition: all 0.2s ease;
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
