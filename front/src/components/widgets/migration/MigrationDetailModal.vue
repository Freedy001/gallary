<template>
  <Teleport to="body">
    <Transition name="popover">
      <div
        v-if="visible"
        ref="modalRef"
        :style="positionStyle"
        class="fixed z-[40] w-96 flex flex-col"
      >
        <LiquidGlassCard
            :hover-effect="false"
            class="w-full h-full flex flex-col"
            content-class="p-0 flex flex-col h-full"
        >
          <!-- 头部 -->
          <div class="flex items-center justify-between px-4 py-3 border-b border-white/10 bg-white/5">
            <h3 class="text-sm font-medium text-white flex items-center gap-2">
              <ArrowsRightLeftIcon class="h-4 w-4 text-blue-400" />
              存储迁移任务
            </h3>
            <button
              class="text-white/40 hover:text-white transition-colors p-1 hover:bg-white/10 rounded"
              @click="$emit('close')"
            >
              <XMarkIcon class="h-4 w-4" />
            </button>
          </div>

          <!-- 任务列表视图 -->
          <div class="p-2 space-y-2 flex-1 overflow-y-auto min-h-0 custom-scrollbar">
            <template v-if="migrationStore.tasks.length > 0">
              <div
                v-for="task in migrationStore.tasks"
                :key="task.task_id"
                class="glass-item group"
              >
                <div class="flex items-center justify-between mb-2">
                  <div class="flex items-center gap-2">
                    <div :class="getTaskStatusDotClass(task)" class="w-2 h-2 rounded-full shadow-[0_0_5px_currentColor]"></div>
                    <span class="text-xs font-medium text-gray-200">
                      {{ getMigrationTypeLabel(task.migration_type) }}
                    </span>
                  </div>

                  <!-- 状态标签 -->
                  <span :class="getStatusBadgeClass(task.status)" class="text-[10px] px-1.5 py-0.5 rounded">
                    {{ getStatusLabel(task.status) }}
                  </span>
                </div>

                <!-- 存储路径 -->
                <div class="text-[10px] text-white/50 mb-2 flex items-center gap-1">
                  <span :title="task.source_storage_id" class="truncate max-w-[120px]">{{ getStorageName(task.source_storage_id) }}</span>
                  <ArrowRightIcon class="h-3 w-3 flex-shrink-0" />
                  <span :title="task.target_storage_id" class="truncate max-w-[120px]">{{ getStorageName(task.target_storage_id) }}</span>
                </div>

                <!-- 进度条 -->
                <div class="mb-2">
                  <div class="flex justify-between text-[10px] text-white/50 mb-1">
                    <span>{{ task.processed_files }} / {{ task.total_files }}</span>
                    <span>{{ Math.round(task.progress_percent) }}%</span>
                  </div>
                  <div class="h-1.5 bg-white/5 rounded-full overflow-hidden">
                    <div
                        :class="getProgressBarClass(task)"
                        :style="{ width: `${task.progress_percent}%` }"
                        class="h-full rounded-full transition-all duration-500 ease-out"
                    ></div>
                  </div>
                  <!-- 速度和剩余时间 -->
                  <div v-if="task.status === 'running' && task.speed > 0" class="flex justify-between text-[10px] text-white/40 mt-1">
                    <span>{{ formatSpeed(task.speed) }}</span>
                    <span>剩余 {{ formatRemainingTime(task.remaining_seconds) }}</span>
                  </div>
                </div>

                <!-- 失败数 -->
                <div v-if="task.failed_files > 0" class="text-[10px] mb-2">
                  <button
                      class="text-red-400 hover:text-red-300 transition-colors underline"
                      @click.stop="viewFailedFiles(task.task_id)"
                  >
                    {{ task.failed_files }} 个文件迁移失败 (查看详情)
                  </button>
                </div>

                <!-- 操作按钮 -->
                <div class="flex gap-2 pt-2 border-t border-white/5">
                  <template v-if="task.status === 'running'">
                    <button
                        :disabled="migrationStore.loading"
                        class="flex-1 text-[10px] px-2 py-1 bg-yellow-500/20 text-yellow-300 rounded hover:bg-yellow-500/30 transition-colors border border-yellow-500/20 disabled:opacity-50"
                        @click="handlePause(task.task_id)"
                    >
                      暂停
                    </button>
                  </template>
                  <template v-else-if="task.status === 'paused'">
                    <button
                        :disabled="migrationStore.loading"
                        class="flex-1 text-[10px] px-2 py-1 bg-blue-500/20 text-blue-300 rounded hover:bg-blue-500/30 transition-colors border border-blue-500/20 disabled:opacity-50"
                        @click="handleResume(task.task_id)"
                    >
                      恢复
                    </button>
                  </template>
                  <template v-else-if="task.status === 'completed' && task.failed_files > 0">
                    <button
                        :disabled="migrationStore.loading"
                        class="flex-1 text-[10px] px-2 py-1 bg-blue-500/20 text-blue-300 rounded hover:bg-blue-500/30 transition-colors border border-blue-500/20 disabled:opacity-50"
                        @click="handleRetry(task.task_id)"
                    >
                      重试失败
                    </button>
                    <button
                        :disabled="migrationStore.loading"
                        class="flex-1 text-[10px] px-2 py-1 bg-gray-500/20 text-gray-300 rounded hover:bg-gray-500/30 transition-colors border border-gray-500/20 disabled:opacity-50"
                        @click="handleDismiss(task.task_id)"
                    >
                      忽略
                    </button>
                  </template>
                  <button
                      v-if="task.status === 'running' || task.status === 'paused' || task.status === 'pending'"
                      :disabled="migrationStore.loading"
                      class="flex-1 text-[10px] px-2 py-1 bg-red-500/20 text-red-300 rounded hover:bg-red-500/30 transition-colors border border-red-500/20 disabled:opacity-50"
                      @click="handleCancel(task.task_id)"
                  >
                    取消
                  </button>
                </div>
              </div>
            </template>
            <div v-else class="py-8 text-center flex flex-col items-center justify-center text-white/30 gap-2">
              <ArrowsRightLeftIcon class="h-8 w-8 opacity-20" />
              <span class="text-xs">暂无迁移任务</span>
            </div>
          </div>
        </LiquidGlassCard>
      </div>
    </Transition>
  </Teleport>

  <!-- 失败文件详情弹窗 -->
  <MigrationFailedFilesModal
    :task-id="viewingFailedTaskId"
    :visible="!!viewingFailedTaskId"
    @close="viewingFailedTaskId = null"
  />
</template>

<script lang="ts" setup>
import {computed, ref, watch} from 'vue'
import {ArrowRightIcon, ArrowsRightLeftIcon, XMarkIcon} from '@heroicons/vue/24/outline'
import {useMigrationStore} from '@/stores/migration.ts'
import {storageMigrationApi} from '@/api/storageMigration'
import {onClickOutside} from '@vueuse/core'
import LiquidGlassCard from '@/components/common/LiquidGlassCard.vue'
import MigrationFailedFilesModal from '@/components/widgets/migration/MigrationFailedFilesModal.vue'
import type {MigrationProgressVO, MigrationStatus, MigrationType} from '@/types/migration.ts'
import {getStorageDriverName, parseStorageId} from '@/api/storage.ts'

const props = defineProps<{
  visible: boolean
  triggerRect: DOMRect | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const migrationStore = useMigrationStore()
const modalRef = ref<HTMLElement | null>(null)

// 当前查看失败文件的任务 ID
const viewingFailedTaskId = ref<number | null>(null)

// 关闭弹窗时重置状态
watch(() => props.visible, (visible) => {
  if (!visible) {
    viewingFailedTaskId.value = null
  }
})

// 监听外部点击关闭弹窗
onClickOutside(modalRef, (event) => {
  if (props.visible) {
    // 如果正在查看失败文件详情，不关闭弹窗
    if (viewingFailedTaskId.value) {
      return
    }

    event.stopPropagation()
    emit('close')
  }
})

// 计算位置
const positionStyle = computed(() => {
  if (!props.triggerRect) return {}

  const { left, width, bottom, top } = props.triggerRect
  const gap = 12
  const padding = 20
  const x = left + width + gap
  const windowHeight = window.innerHeight

  const isLowerHalf = top > windowHeight / 2

  if (isLowerHalf) {
    const bottomOffset = windowHeight - bottom
    const availableHeight = bottom - padding

    return {
      left: `${x}px`,
      bottom: `${bottomOffset}px`,
      maxHeight: `${availableHeight}px`,
      transformOrigin: 'left bottom'
    }
  } else {
    const topOffset = top
    const availableHeight = windowHeight - top - padding

    return {
      left: `${x}px`,
      top: `${topOffset}px`,
      maxHeight: `${availableHeight}px`,
      transformOrigin: 'left top'
    }
  }
})

function getMigrationTypeLabel(type: MigrationType): string {
  return type === 'original' ? '原图迁移' : '缩略图迁移'
}

function getStorageName(storageId: string): string {
  const { accountId } = parseStorageId(storageId)
  const driverName = getStorageDriverName(storageId)
  return accountId ? `${driverName}` : driverName
}

function getStatusLabel(status: MigrationStatus): string {
  const labels: Record<MigrationStatus, string> = {
    pending: '等待中',
    running: '运行中',
    paused: '已暂停',
    completed: '已完成',
    failed: '失败',
    cancelled: '已取消'
  }
  return labels[status] || status
}

function getStatusBadgeClass(status: MigrationStatus): string {
  const classes: Record<MigrationStatus, string> = {
    pending: 'bg-gray-500/20 text-gray-300 border border-gray-500/20',
    running: 'bg-blue-500/20 text-blue-300 border border-blue-500/20',
    paused: 'bg-yellow-500/20 text-yellow-300 border border-yellow-500/20',
    completed: 'bg-green-500/20 text-green-300 border border-green-500/20',
    failed: 'bg-red-500/20 text-red-300 border border-red-500/20',
    cancelled: 'bg-gray-500/20 text-gray-300 border border-gray-500/20'
  }
  return classes[status] || ''
}

function getTaskStatusDotClass(task: MigrationProgressVO): string {
  if (task.status === 'running') {
    return 'bg-blue-500 animate-pulse text-blue-500'
  }
  if (task.status === 'paused') {
    return 'bg-yellow-500 text-yellow-500'
  }
  if (task.status === 'pending') {
    return 'bg-gray-500 text-gray-500'
  }
  if (task.failed_files > 0) {
    return 'bg-red-500 text-red-500'
  }
  return 'bg-green-500 text-green-500'
}

function getProgressBarClass(task: MigrationProgressVO): string {
  if (task.status === 'running') {
    return 'bg-blue-500 animate-pulse'
  }
  if (task.status === 'paused') {
    return 'bg-yellow-500'
  }
  return 'bg-gray-500'
}

// 格式化传输速度
function formatSpeed(bytesPerSecond: number): string {
  if (bytesPerSecond < 1024) {
    return `${bytesPerSecond} B/s`
  } else if (bytesPerSecond < 1024 * 1024) {
    return `${(bytesPerSecond / 1024).toFixed(1)} KB/s`
  } else if (bytesPerSecond < 1024 * 1024 * 1024) {
    return `${(bytesPerSecond / (1024 * 1024)).toFixed(1)} MB/s`
  } else {
    return `${(bytesPerSecond / (1024 * 1024 * 1024)).toFixed(1)} GB/s`
  }
}

// 格式化剩余时间
function formatRemainingTime(seconds: number): string {
  if (seconds <= 0) {
    return '计算中...'
  }
  if (seconds < 60) {
    return `${seconds}秒`
  } else if (seconds < 3600) {
    const minutes = Math.floor(seconds / 60)
    const secs = seconds % 60
    return secs > 0 ? `${minutes}分${secs}秒` : `${minutes}分钟`
  } else {
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    return minutes > 0 ? `${hours}小时${minutes}分` : `${hours}小时`
  }
}

async function handlePause(taskId: number) {
  try {
    migrationStore.loading = true
    await storageMigrationApi.pauseMigration(taskId)
  } catch (err) {
    console.error('暂停迁移失败', err)
  } finally {
    migrationStore.loading = false
  }
}

async function handleResume(taskId: number) {
  try {
    migrationStore.loading = true
    await storageMigrationApi.resumeMigration(taskId)
  } catch (err) {
    console.error('恢复迁移失败', err)
  } finally {
    migrationStore.loading = false
  }
}

async function handleCancel(taskId: number) {
  try {
    migrationStore.loading = true
    await storageMigrationApi.dismissFailedFiles(taskId)
  } catch (err) {
    console.error('取消迁移失败', err)
  } finally {
    migrationStore.loading = false
  }
}

async function handleRetry(taskId: number) {
  try {
    migrationStore.loading = true
    await storageMigrationApi.retryFailedFiles(taskId)
  } catch (err) {
    console.error('重试失败文件失败', err)
  } finally {
    migrationStore.loading = false
  }
}

async function handleDismiss(taskId: number) {
  try {
    migrationStore.loading = true
    await storageMigrationApi.dismissFailedFiles(taskId)
  } catch (err) {
    console.error('忽略失败文件失败', err)
  } finally {
    migrationStore.loading = false
  }
}

async function viewFailedFiles(taskId: number) {
  // 只需要设置 taskId，MigrationFailedFilesModal 会自己加载数据
  viewingFailedTaskId.value = taskId
}
</script>

<style scoped>
.glass-item {
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 0.75rem;
  padding: 0.75rem;
  transition: all 0.2s ease;
}

.glass-item:hover {
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.15);
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

.popover-enter-active,
.popover-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.popover-enter-from,
.popover-leave-to {
  opacity: 0;
  transform: scale(0.95) translateX(-10px);
}
</style>
