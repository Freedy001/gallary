import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { aiApi } from '@/api/ai'
import { useNotificationStore } from './notification'
import type { AIConfig, AIQueueDetail, AIQueueInfo } from '@/types/ai'

export const useAIStore = defineStore('ai', () => {
  // ================== State ==================
  const config = ref<AIConfig>({
    models: []
  })
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
  // 获取 AI 配置
  async function fetchConfig() {
    try {
      const response = await aiApi.getSettings()
      if (response.data) {
        config.value = {
          models: response.data.models || []
        }
      }
    } catch (err) {
      throw err
    }
  }

  // 更新 AI 配置
  async function updateConfig(newConfig: AIConfig) {
    try {
      await aiApi.updateSettings(newConfig)
      config.value = newConfig
    } catch (err) {
      throw err
    }
  }

  // 测试连接
  async function testConnection(id: string) {
    try {
      loading.value = true
      const response = await aiApi.testConnection({ id })
      return response.data?.message || '连接成功'
    } catch (err) {
      throw err
    } finally {
      loading.value = false
    }
  }


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

  // 重试单张图片
  async function retryTaskImage(taskImageId: number) {
    try {
      loading.value = true
      await aiApi.retryTaskImage(taskImageId)
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

  // 忽略单张图片
  async function ignoreTaskImage(taskImageId: number) {
    try {
      loading.value = true
      await aiApi.ignoreTaskImage(taskImageId)
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
    config,
    queueStatus,
    queueDetail,
    loading,

    // Computed
    queues,

    // Actions
    fetchConfig,
    updateConfig,
    testConnection,
    fetchQueueDetail,
    retryQueueFailedImages,
    retryTaskImage,
    ignoreTaskImage,
  }
})
