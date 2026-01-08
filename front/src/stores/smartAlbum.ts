import {defineStore} from 'pinia'
import {computed, ref} from 'vue'
import {wsService} from '@/services/websocket'
import type {SmartAlbumProgressVO} from '@/types/smart-album'
import {useDialogStore} from '@/stores/dialog'

export const useSmartAlbumStore = defineStore('smartAlbum', () => {
  const dialogStore = useDialogStore()

  // State
  const currentTaskId = ref<number | null>(null)
  const currentProgress = ref<SmartAlbumProgressVO | null>(null)
  const result = ref<SmartAlbumProgressVO | null>(null)
  const errorMessage = ref<string | null>(null)

  // Computed
  const taskInProgress = computed(() => {
    if (!currentProgress.value) return false
    const status = currentProgress.value.status
    return status === 'pending' || status === 'collecting' || status === 'clustering' || status === 'creating'
  })

  // Actions
  function setTaskId(id: number) {
    currentTaskId.value = id
    // Reset state when starting new task
    currentProgress.value = {
      task_id: id,
      status: 'pending',
      progress: 0,
      message: '等待处理...'
    }
    result.value = null
    errorMessage.value = null
  }

  function handleProgressMessage(data: SmartAlbumProgressVO) {
    // Only handle current task progress
    if (currentTaskId.value && data.task_id === currentTaskId.value) {
      currentProgress.value = data

      if (data.status === 'completed') {
        result.value = data
        dialogStore.notify({
          title: '成功',
          message: `成功创建 ${data.cluster_count || 0} 个智能相册`,
          type: 'success'
        })
      } else if (data.status === 'failed') {
        errorMessage.value = data.error || '任务执行失败'
        dialogStore.notify({
          title: '错误',
          message: data.error || '任务执行失败',
          type: 'error'
        })
      }
    }
  }

  function resetState() {
    currentTaskId.value = null
    currentProgress.value = null
    result.value = null
    errorMessage.value = null
  }

  // Subscribe to websocket updates
  wsService.subscribe<SmartAlbumProgressVO>('smart_album_progress', handleProgressMessage)

  return {
    currentTaskId,
    currentProgress,
    result,
    errorMessage,
    taskInProgress,
    setTaskId,
    resetState
  }
})
