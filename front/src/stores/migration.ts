import {defineStore} from 'pinia'
import {computed, ref} from 'vue'
import {useNotificationStore} from './notification'
import type {MigrationHistoryResponse,} from '@/types/migration'

export const useMigrationStore = defineStore('migration', () => {
  // ================== State ==================
  const loading = ref(false)
  const historyData = ref<MigrationHistoryResponse | null>(null)

  // 从 notification store 获取迁移状态
  const notificationStore = useNotificationStore()
  // ================== Computed ==================

  // 所有活跃任务
  const tasks = computed(() => notificationStore.migrationStatus?.tasks || [])

  // 是否有活跃任务
  const hasTasks = computed(() => tasks.value.length > 0)

  // 运行中任务数
  const runningCount = computed(() => notificationStore.migrationStatus?.total_running || 0)

  // 暂停任务数
  const pausedCount = computed(() => notificationStore.migrationStatus?.total_paused || 0)

  // 总体进度（所有任务的平均进度）
  const overallProgress = computed(() => {
    const task = tasks.value
    if (task.length === 0) return 0
    const totalProgress = task.reduce((sum, task) => sum + task.progress_percent, 0)
    return Math.round(totalProgress / task.length)
  })

  // ================== Actions ==================

  return {
    // State
    loading,
    migrationStatus: notificationStore.migrationStatus,
    historyData,

    // Computed
    tasks: tasks,
    hasTasks: hasTasks,
    runningCount: runningCount,
    pausedCount: pausedCount,
    overallProgress: overallProgress,
  }
})
