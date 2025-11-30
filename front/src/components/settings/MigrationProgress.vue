<template>
  <div v-if="activeMigration || showCompleted" class="rounded-xl bg-white/[0.03] border border-white/10 p-5">
    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center gap-3">
        <div :class="statusIconClass">
          <component :is="statusIcon" class="h-5 w-5" />
        </div>
        <div>
          <h3 class="text-sm font-medium text-white">存储迁移</h3>
          <p class="text-xs text-gray-500 mt-0.5">{{ statusText }}</p>
        </div>
      </div>

      <!-- 状态标签 -->
      <span :class="statusBadgeClass">
        {{ statusLabel }}
      </span>
    </div>

    <!-- 迁移任务信息 -->
    <div v-if="activeMigration" class="space-y-4">
      <!-- 进度条 -->
      <div v-if="activeMigration.status === 'running'" class="space-y-2">
        <div class="flex justify-between text-xs text-gray-400">
          <span>迁移进度</span>
          <span>{{ activeMigration.processed_files }} / {{ activeMigration.total_files }}</span>
        </div>
        <div class="h-2 bg-white/10 rounded-full overflow-hidden">
          <div
            class="h-full bg-gradient-to-r from-primary-500 to-primary-400 rounded-full transition-all duration-300"
            :style="{ width: progressPercent + '%' }"
          ></div>
        </div>
        <div class="text-xs text-gray-500 text-right">{{ progressPercent.toFixed(1) }}%</div>
      </div>

      <!-- 路径信息 -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3 text-xs">
        <div class="p-3 rounded-lg bg-white/[0.02] border border-white/5">
          <div class="text-gray-500 mb-1">原路径</div>
          <div class="text-gray-300 font-mono break-all">{{ activeMigration.old_base_path }}</div>
        </div>
        <div class="p-3 rounded-lg bg-white/[0.02] border border-white/5">
          <div class="text-gray-500 mb-1">新路径</div>
          <div class="text-gray-300 font-mono break-all">{{ activeMigration.new_base_path }}</div>
        </div>
      </div>

      <!-- 错误信息 -->
      <div v-if="activeMigration.error_message" class="p-3 rounded-lg bg-red-500/10 border border-red-500/20">
        <div class="flex items-start gap-2">
          <ExclamationTriangleIcon class="h-4 w-4 text-red-400 flex-shrink-0 mt-0.5" />
          <div class="text-sm text-red-300">{{ activeMigration.error_message }}</div>
        </div>
      </div>

      <!-- 时间信息 -->
      <div class="flex flex-wrap gap-4 text-xs text-gray-500">
        <div v-if="activeMigration.started_at">
          <span class="text-gray-600">开始时间:</span>
          <span class="ml-1">{{ formatTime(activeMigration.started_at) }}</span>
        </div>
        <div v-if="activeMigration.completed_at">
          <span class="text-gray-600">完成时间:</span>
          <span class="ml-1">{{ formatTime(activeMigration.completed_at) }}</span>
        </div>
      </div>

      <!-- 操作按钮 -->
      <div class="flex gap-3 pt-2">
        <button
          v-if="activeMigration.status === 'running' || activeMigration.status === 'pending'"
          @click="handleCancel"
          :disabled="cancelling"
          class="px-4 py-2 rounded-lg bg-red-500/20 text-red-400 hover:bg-red-500/30 ring-1 ring-red-500/30 transition-all duration-300 text-sm disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {{ cancelling ? '取消中...' : '取消迁移' }}
        </button>
        <button
          v-if="isFinished"
          @click="dismiss"
          class="px-4 py-2 rounded-lg bg-white/5 text-gray-400 hover:bg-white/10 ring-1 ring-white/10 transition-all duration-300 text-sm"
        >
          关闭
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { migrationApi, type MigrationTask, type MigrationStatus } from '@/api/migration'
import { useDialogStore } from '@/stores/dialog'
import {
  ArrowPathIcon,
  CheckCircleIcon,
  XCircleIcon,
  ExclamationTriangleIcon,
  ClockIcon,
} from '@heroicons/vue/24/outline'

const dialogStore = useDialogStore()

const activeMigration = ref<MigrationTask | null>(null)
const showCompleted = ref(false)
const cancelling = ref(false)
let pollingTimer: ReturnType<typeof setInterval> | null = null

// 进度百分比
const progressPercent = computed(() => {
  if (!activeMigration.value || activeMigration.value.total_files === 0) return 0
  return (activeMigration.value.processed_files / activeMigration.value.total_files) * 100
})

// 是否已完成（成功、失败、取消、回滚）
const isFinished = computed(() => {
  if (!activeMigration.value) return false
  return ['completed', 'failed', 'cancelled', 'rolled_back'].includes(activeMigration.value.status)
})

// 状态文本
const statusText = computed(() => {
  if (!activeMigration.value) return ''
  const statusMap: Record<MigrationStatus, string> = {
    pending: '准备开始迁移文件...',
    running: '正在迁移文件，请勿关闭页面',
    completed: '所有文件已成功迁移到新位置',
    failed: '迁移过程中发生错误',
    rolled_back: '迁移已回滚，文件恢复原状',
    cancelled: '迁移已被用户取消',
  }
  return statusMap[activeMigration.value.status] || ''
})

// 状态标签
const statusLabel = computed(() => {
  if (!activeMigration.value) return ''
  const labelMap: Record<MigrationStatus, string> = {
    pending: '等待中',
    running: '迁移中',
    completed: '已完成',
    failed: '失败',
    rolled_back: '已回滚',
    cancelled: '已取消',
  }
  return labelMap[activeMigration.value.status] || ''
})

// 状态标签样式
const statusBadgeClass = computed(() => {
  if (!activeMigration.value) return ''
  const classMap: Record<MigrationStatus, string> = {
    pending: 'px-2.5 py-1 rounded-full text-xs font-medium bg-yellow-500/20 text-yellow-400 ring-1 ring-yellow-500/30',
    running: 'px-2.5 py-1 rounded-full text-xs font-medium bg-blue-500/20 text-blue-400 ring-1 ring-blue-500/30 animate-pulse',
    completed: 'px-2.5 py-1 rounded-full text-xs font-medium bg-green-500/20 text-green-400 ring-1 ring-green-500/30',
    failed: 'px-2.5 py-1 rounded-full text-xs font-medium bg-red-500/20 text-red-400 ring-1 ring-red-500/30',
    rolled_back: 'px-2.5 py-1 rounded-full text-xs font-medium bg-orange-500/20 text-orange-400 ring-1 ring-orange-500/30',
    cancelled: 'px-2.5 py-1 rounded-full text-xs font-medium bg-gray-500/20 text-gray-400 ring-1 ring-gray-500/30',
  }
  return classMap[activeMigration.value.status] || ''
})

// 状态图标
const statusIcon = computed(() => {
  if (!activeMigration.value) return ClockIcon
  const iconMap: Record<MigrationStatus, any> = {
    pending: ClockIcon,
    running: ArrowPathIcon,
    completed: CheckCircleIcon,
    failed: XCircleIcon,
    rolled_back: ExclamationTriangleIcon,
    cancelled: XCircleIcon,
  }
  return iconMap[activeMigration.value.status] || ClockIcon
})

// 状态图标样式
const statusIconClass = computed(() => {
  if (!activeMigration.value) return 'p-2 rounded-lg bg-gray-500/20 text-gray-400'
  const classMap: Record<MigrationStatus, string> = {
    pending: 'p-2 rounded-lg bg-yellow-500/20 text-yellow-400',
    running: 'p-2 rounded-lg bg-blue-500/20 text-blue-400 animate-spin',
    completed: 'p-2 rounded-lg bg-green-500/20 text-green-400',
    failed: 'p-2 rounded-lg bg-red-500/20 text-red-400',
    rolled_back: 'p-2 rounded-lg bg-orange-500/20 text-orange-400',
    cancelled: 'p-2 rounded-lg bg-gray-500/20 text-gray-400',
  }
  return classMap[activeMigration.value.status] || 'p-2 rounded-lg bg-gray-500/20 text-gray-400'
})

// 格式化时间
function formatTime(timeStr: string): string {
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

// 获取活跃迁移任务
async function fetchActiveMigration() {
  try {
    const resp = await migrationApi.getActive()
    activeMigration.value = resp.data

    if (activeMigration.value) {
      showCompleted.value = true

      // 如果正在进行中，启动轮询
      if (!isFinished.value && !pollingTimer) {
        startPolling()
      }

      // 如果已完成，停止轮询
      if (isFinished.value && pollingTimer) {
        stopPolling()
      }
    }
  } catch (error) {
    console.error('获取迁移任务失败:', error)
  }
}

// 开始轮询
function startPolling() {
  stopPolling()
  pollingTimer = setInterval(fetchActiveMigration, 2000)
}

// 停止轮询
function stopPolling() {
  if (pollingTimer) {
    clearInterval(pollingTimer)
    pollingTimer = null
  }
}

// 取消迁移
async function handleCancel() {
  if (!activeMigration.value) return

  const confirmed = await dialogStore.confirm({
    title: '取消迁移',
    message: '确定要取消迁移吗？已复制的文件将被删除，原文件保持不变。',
    type: 'warning'
  })

  if (!confirmed) return

  cancelling.value = true
  try {
    await migrationApi.cancel(activeMigration.value.id)
    await fetchActiveMigration()
    await dialogStore.alert({
      title: '已取消',
      message: '迁移任务已取消',
      type: 'info'
    })
  } catch (error: any) {
    await dialogStore.alert({
      title: '错误',
      message: error.message || '取消迁移失败',
      type: 'error'
    })
  } finally {
    cancelling.value = false
  }
}

// 关闭完成提示
function dismiss() {
  showCompleted.value = false
  activeMigration.value = null
}

// 暴露刷新方法给父组件
defineExpose({
  refresh: fetchActiveMigration,
})

onMounted(() => {
  fetchActiveMigration()
})

onUnmounted(() => {
  stopPolling()
})
</script>
