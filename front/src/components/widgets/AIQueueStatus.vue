<template>
  <div
    ref="containerRef"
    class="group cursor-pointer hover:bg-white/5 -mx-2 px-2 py-2 rounded-lg transition-colors"
    @click="openModal"
  >
    <!-- 展开状态 -->
    <template v-if="!collapsed">
      <div class="flex items-center justify-between text-xs text-gray-400 mb-1.5">
        <div class="flex items-center gap-1.5 font-mono tracking-wider">
          <SparklesIcon class="h-3.5 w-3.5" :class="hasActiveTasks ? 'text-primary-400' : ''" />
          <span>AI 处理</span>
        </div>
        <ArrowRightIcon class="h-3 w-3 opacity-0 group-hover:opacity-100 transition-opacity" />
      </div>

      <!-- 状态摘要 -->
      <div class="space-y-1.5">
        <div class="flex items-center justify-between text-xs">
          <span class="text-gray-300">
            <template v-if="hasActiveTasks">
              正在处理
            </template>
            <template v-else-if="hasFailedTasks">
              需关注
            </template>
            <template v-else>
              空闲
            </template>
          </span>

          <span class="tabular-nums flex items-center gap-2">
            <!-- 失败数 -->
            <span v-if="totalFailed > 0" class="text-red-400 font-medium flex items-center gap-1">
              {{ totalFailed }} 失败
            </span>
            <!-- 进行/等待数 -->
            <span v-if="totalPending + processingCount > 0" class="text-primary-300">
              {{ processingCount + totalPending }} 任务
            </span>
          </span>
        </div>

        <!-- 总体进度条 -->
        <div class="h-1 bg-white/5 rounded-full overflow-hidden">
          <div
            class="h-full rounded-full transition-all duration-500 ease-out"
            :class="getProgressBarClass"
            :style="{ width: `${overallProgress}%` }"
          ></div>
        </div>
      </div>
    </template>

    <!-- 收起状态 -->
    <template v-else>
      <div class="flex flex-col items-center gap-1">
        <div class="relative">
          <SparklesIcon class="h-4 w-4 text-gray-500 group-hover:text-primary-400 transition-colors" />
          <!-- 状态指示点 -->
          <span
            v-if="hasActiveTasks"
            class="absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full bg-primary-500 animate-pulse border border-gray-900"
          ></span>
          <span
            v-else-if="hasFailedTasks"
            class="absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full bg-red-500 border border-gray-900"
          ></span>
        </div>

        <!-- 简单的数量指示 -->
        <span v-if="processingCount > 0" class="text-[10px] text-primary-400 font-bold">
          {{ processingCount }}
        </span>
      </div>
    </template>

    <!-- 详情弹窗 -->
    <AIQueueListModal
        :visible="modalVisible"
        :trigger-rect="triggerRect"
        @close="modalVisible = false"
    />
  </div>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'
import {ArrowRightIcon, SparklesIcon} from '@heroicons/vue/24/outline'
import {useNotificationStore} from '@/stores/notification.ts'
import AIQueueListModal from '@/components/widgets/AIQueueListModal.vue'

defineProps<{
  collapsed: boolean
}>()

const notificationStore = useNotificationStore()
const containerRef = ref<HTMLElement | null>(null)
const modalVisible = ref(false)
const triggerRect = ref<DOMRect | null>(null)

// 打开弹窗
function openModal() {
  // 获取点击时的触发区域位置
  if (containerRef.value) {
    triggerRect.value = containerRef.value.getBoundingClientRect()
    modalVisible.value = true
  }
}

// 计算属性
const totalPending = computed(() => notificationStore.aiQueueStatus?.total_pending || 0)
const totalFailed = computed(() => notificationStore.aiQueueStatus?.total_failed || 0)
// 计算正在处理中的队列数量
const processingCount = computed(() => {
  const queues = notificationStore.aiQueueStatus?.queues || []
  return queues.filter(q => q.status === 'processing').length
})

const hasActiveTasks = computed(() => processingCount.value > 0 || totalPending.value > 0)
const hasFailedTasks = computed(() => totalFailed.value > 0)

// 总体进度计算
const overallProgress = computed(() => {
  const total = totalPending.value + processingCount.value + totalFailed.value
  if (total === 0) return 0 // 空闲时进度为0
  if (processingCount.value > 0) {
    // 简单的动画效果模拟：如果有正在处理的，显示至少 10%
    return Math.max(10, Math.round((processingCount.value / (totalPending.value + processingCount.value)) * 100))
  }
  return 0
})

const getProgressBarClass = computed(() => {
  if (hasFailedTasks.value) {
    return 'bg-red-500'
  }
  if (hasActiveTasks.value) {
    return 'bg-primary-500 animate-pulse'
  }
  return 'bg-gray-700'
})

</script>