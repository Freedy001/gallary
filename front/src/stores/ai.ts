import {defineStore} from 'pinia'
import {computed, ref} from 'vue'
import {aiApi} from '@/api/ai'
import {useNotificationStore} from './notification'
import type {AIQueueDetail, AIQueueInfo} from '@/types/ai'

export const useAIStore = defineStore('ai', () => {
  // ================== State ==================
  const queueDetail = ref<AIQueueDetail | null>(null)
  const loading = ref(false)

  // 从 notification store 获取 queueStatus
  const notificationStore = useNotificationStore()
  const queueStatus = computed(() => notificationStore.aiQueueStatus)
  // ================== Computed ==================

  // 所有队列
  const queues = computed((): AIQueueInfo[] => {
    return queueStatus.value?.queues || []
  })
  // ================== Actions ==================

  // 获取队列详情
  async function fetchQueueDetail(queueId: number, page = 1, pageSize = 20) {
    try {
      loading.value = true
      const response = await aiApi.getQueueDetail(queueId, page, pageSize)
      if (response.data) {
        queueDetail.value = response.data
      }
      return response.data
    } catch (err) {
      throw err
    } finally {
      loading.value = false
    }
  }

  // 重试队列所有失败图片
  async function retryQueueFailedImages(queueId: number) {
    try {
      loading.value = true
      await aiApi.retryQueueFailedImages(queueId)
      // 如果当前正在查看该队列的详情，也刷新详情
      if (queueDetail.value?.queue.id === queueId) {
        await fetchQueueDetail(queueId)
      }
    } catch (err) {
      throw err
    } finally {
      loading.value = false
    }
  }

  // 重试单个任务项
  async function retryTaskItem(taskItemId: number) {
    try {
      loading.value = true
      await aiApi.retryTaskImage(taskItemId)
      // 如果当前正在查看详情，也刷新详情
      if (queueDetail.value) {
        await fetchQueueDetail(queueDetail.value.queue.id, queueDetail.value.page, queueDetail.value.page_size)
      }
    } catch (err) {
      throw err
    } finally {
      loading.value = false
    }
  }

  // 忽略单个任务项
  async function ignoreTaskItem(taskItemId: number) {
    try {
      loading.value = true
      await aiApi.ignoreTaskImage(taskItemId)
      // 如果当前正在查看详情，也刷新详情
      if (queueDetail.value) {
        await fetchQueueDetail(queueDetail.value.queue.id, queueDetail.value.page, queueDetail.value.page_size)
      }
    } catch (err) {
      throw err
    } finally {
      loading.value = false
    }
  }


  return {
    // State
    queueStatus,
    queueDetail,
    loading,

    // Computed
    queues,

    // Actions
    fetchQueueDetail,
    retryQueueFailedImages,
    retryTaskItem,
    ignoreTaskItem,
  }
})
