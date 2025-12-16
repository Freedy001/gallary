<template>
  <div class="space-y-3">
    <!-- 展开状态 -->
    <template v-if="!collapsed">
      <div class="text-xs text-gray-500 font-mono tracking-wider mb-2 flex items-center justify-between">
        <span class="flex items-center gap-1.5">
          <SparklesIcon class="h-3.5 w-3.5" />
          AI 处理
        </span>
      </div>

      <!-- 有队列数据时 -->
      <template v-if="hasAnyActivity">
        <div class="space-y-2">
          <!-- 队列列表 -->
          <div
              v-for="queue in displayQueues"
              :key="queue.id"
              class="space-y-1.5 cursor-pointer hover:bg-white/5 -mx-2 px-2 py-1 rounded transition-colors"
              @click="openQueueDetail(queue)"
          >
            <div class="flex items-center justify-between text-xs">
              <span class="flex items-center gap-1.5">
                <span
                    class="w-1.5 h-1.5 rounded-full"
                    :class="getQueueStatusDotClass(queue)"
                ></span>
                <span class="text-gray-300 truncate max-w-[100px]">
                  {{ getQueueLabel(queue) }}
                </span>
              </span>
              <span class="text-gray-400 tabular-nums flex items-center gap-1">
                <span v-if="queue.pending_count > 0 || queue.processing_count > 0">
                  {{ queue.processing_count + queue.pending_count }}
                </span>
                <span v-if="queue.failed_count > 0" class="text-red-400">
                  {{ queue.failed_count }} 失败
                </span>
              </span>
            </div>
            <!-- 进度条 -->
            <div class="h-1 bg-white/5 rounded-full overflow-hidden">
              <div
                  class="h-full rounded-full transition-all duration-500 ease-out"
                  :class="getQueueProgressBarClass(queue)"
                  :style="{ width: `${getQueueProgress(queue)}%` }"
              ></div>
            </div>
          </div>

          <!-- 统计摘要 -->
          <div class="flex items-center justify-between text-[10px] text-gray-500 pt-1">
            <span>待处理: {{ totalPending }}</span>
            <span v-if="totalFailed > 0" class="text-red-400">
              失败: {{ totalFailed }}
            </span>
          </div>
        </div>
      </template>

      <!-- 无任务时 -->
      <div v-else class="text-xs text-gray-600 flex items-center gap-1.5">
        <CheckCircleIcon class="h-3.5 w-3.5 text-green-500/50" />
        暂无处理任务
      </div>
    </template>

    <!-- 收起状态 -->
    <template v-else>
      <div class="flex flex-col items-center gap-1">
        <!-- AI 图标 + 状态指示 -->
        <div class="relative">
          <SparklesIcon class="h-4 w-4 text-gray-500" />
          <!-- 活动指示器 -->
          <span
              v-if="aiStore.hasActiveTasks"
              class="absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full bg-primary-500 animate-pulse"
          ></span>
          <!-- 失败指示器 -->
          <span
              v-else-if="aiStore.hasFailedTasks"
              class="absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full bg-red-500"
          ></span>
        </div>
        <!-- 进度条 (收起状态) -->
        <div v-if="currentQueue" class="w-8 h-1 bg-white/5 rounded-full overflow-hidden">
          <div
              class="h-full rounded-full transition-all duration-500 ease-out"
              :class="getQueueProgressBarClass(currentQueue)"
              :style="{ width: `${getQueueProgress(currentQueue)}%` }"
          ></div>
        </div>
        <span v-if="aiStore.hasActiveTasks" class="text-[10px] text-gray-500">
          {{ totalProcessing }}
        </span>
      </div>
    </template>

    <!-- 队列详情弹窗 -->
    <AIQueueDetailModal
        v-if="selectedQueue"
        :queue="selectedQueue"
        :visible="detailModalVisible"
        @close="closeQueueDetail"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { SparklesIcon, CheckCircleIcon } from '@heroicons/vue/24/outline'
import { useAIStore } from '@/stores/ai'
import type { AIQueueInfo } from '@/types/ai'
import { getQueueDisplayName } from '@/types/ai'
import AIQueueDetailModal from './AIQueueDetailModal.vue'

defineProps<{
  collapsed: boolean
}>()

const aiStore = useAIStore()

// 详情弹窗状态
const detailModalVisible = ref(false)
const selectedQueue = ref<AIQueueInfo | null>(null)

// 计算属性
const hasAnyActivity = computed(() => {
  return aiStore.queues.length > 0
})

const totalPending = computed(() => {
  return aiStore.queueStatus?.total_pending || 0
})

const totalProcessing = computed(() => {
  return aiStore.queueStatus?.total_processing || 0
})

const totalFailed = computed(() => {
  return aiStore.queueStatus?.total_failed || 0
})

// 显示的队列列表（最多显示 3 个）
const displayQueues = computed(() => {
  const queues = [...aiStore.queues]

  // 按优先级排序：处理中 > 待处理 > 有失败
  queues.sort((a, b) => {
    // 处理中的优先
    if (a.status === 'processing' && b.status !== 'processing') return -1
    if (b.status === 'processing' && a.status !== 'processing') return 1
    // 有待处理的优先
    if (a.pending_count > 0 && b.pending_count === 0) return -1
    if (b.pending_count > 0 && a.pending_count === 0) return 1
    // 有失败的优先
    if (a.failed_count > 0 && b.failed_count === 0) return -1
    if (b.failed_count > 0 && a.failed_count === 0) return 1
    return 0
  })

  return queues.slice(0, 3)
})

// 当前队列（用于收起状态）
const currentQueue = computed(() => {
  return displayQueues.value[0] || null
})

// 获取队列标签
function getQueueLabel(queue: AIQueueInfo): string {
  return getQueueDisplayName(queue)
}

// 队列状态点颜色
function getQueueStatusDotClass(queue: AIQueueInfo): string {
  if (queue.status === 'processing' || queue.processing_count > 0) {
    return 'bg-primary-500 animate-pulse'
  }
  if (queue.pending_count > 0) {
    return 'bg-yellow-500'
  }
  if (queue.failed_count > 0) {
    return 'bg-red-500'
  }
  return 'bg-green-500'
}

// 队列进度条样式
function getQueueProgressBarClass(queue: AIQueueInfo): string {
  if (queue.status === 'processing' || queue.processing_count > 0) {
    return 'bg-gradient-to-r from-primary-500 to-primary-600 shadow-[0_0_10px_rgba(139,92,246,0.3)]'
  }
  if (queue.pending_count > 0) {
    return 'bg-gradient-to-r from-yellow-500 to-orange-500 shadow-[0_0_10px_rgba(234,179,8,0.3)]'
  }
  if (queue.failed_count > 0) {
    return 'bg-gradient-to-r from-red-500 to-red-600 shadow-[0_0_10px_rgba(239,68,68,0.4)]'
  }
  return 'bg-gradient-to-r from-green-500 to-green-600'
}

// 计算队列进度
function getQueueProgress(queue: AIQueueInfo): number {
  const total = queue.pending_count + queue.processing_count + queue.failed_count
  if (total === 0) return 100
  // 如果有待处理或处理中的，显示处理中的进度
  if (queue.pending_count > 0 || queue.processing_count > 0) {
    return Math.round((queue.processing_count / (queue.pending_count + queue.processing_count)) * 50)
  }
  // 只有失败的，显示100%红色进度条
  return 100
}

// 打开队列详情
function openQueueDetail(queue: AIQueueInfo) {
  selectedQueue.value = queue
  detailModalVisible.value = true
}

// 关闭队列详情
function closeQueueDetail() {
  detailModalVisible.value = false
  selectedQueue.value = null
}

onMounted(() => {
  // 开始智能轮询
  aiStore.smartPolling(3000)
})

onUnmounted(() => {
  aiStore.stopPolling()
})
</script>
