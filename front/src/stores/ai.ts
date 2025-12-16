import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { aiApi } from '@/api/ai'
import type { AIConfig, AIQueueStatus, AIQueueDetail, AIQueueInfo, ModelConfig } from '@/types/ai'
import { getEnabledModels } from '@/types/ai'

export const useAIStore = defineStore('ai', () => {
  // ================== State ==================
  const config = ref<AIConfig>({
    models: []
  })
  const queueStatus = ref<AIQueueStatus | null>(null)
  const queueDetail = ref<AIQueueDetail | null>(null)
  const loading = ref(false)
  const configLoading = ref(false)
  const error = ref<string | null>(null)

  // 轮询控制
  let pollingInterval: ReturnType<typeof setInterval> | ReturnType<typeof setTimeout> | null = null
  const isPolling = ref(false)

  // ================== Computed ==================

  // 是否有活跃任务（待处理或处理中）
  const hasActiveTasks = computed(() => {
    if (!queueStatus.value) return false
    return queueStatus.value.total_pending > 0 || queueStatus.value.total_processing > 0
  })

  // 是否有失败任务
  const hasFailedTasks = computed(() => {
    if (!queueStatus.value) return false
    return queueStatus.value.total_failed > 0
  })

  // 所有队列
  const queues = computed((): AIQueueInfo[] => {
    return queueStatus.value?.queues || []
  })

  // 有失败图片的队列
  const queuesWithFailures = computed((): AIQueueInfo[] => {
    return queues.value.filter(q => q.failed_count > 0)
  })

  // 获取所有启用的模型
  const enabledModels = computed((): ModelConfig[] => {
    return getEnabledModels(config.value)
  })

  // ================== Actions ==================

  // 获取 AI 配置
  async function fetchConfig() {
    try {
      configLoading.value = true
      error.value = null
      const response = await aiApi.getSettings()
      if (response.data) {
        config.value = {
          models: response.data.models || []
        }
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : '获取 AI 配置失败'
      throw err
    } finally {
      configLoading.value = false
    }
  }

  // 更新 AI 配置
  async function updateConfig(newConfig: AIConfig) {
    try {
      configLoading.value = true
      error.value = null
      await aiApi.updateSettings(newConfig)
      config.value = newConfig
    } catch (err) {
      error.value = err instanceof Error ? err.message : '更新 AI 配置失败'
      throw err
    } finally {
      configLoading.value = false
    }
  }

  // 测试连接
  async function testConnection(id: string) {
    try {
      loading.value = true
      error.value = null
      const response = await aiApi.testConnection({ id })
      return response.data?.message || '连接成功'
    } catch (err) {
      error.value = err instanceof Error ? err.message : '连接测试失败'
      throw err
    } finally {
      loading.value = false
    }
  }

  // 获取队列状态
  async function fetchQueueStatus() {
    try {
      const response = await aiApi.getQueueStatus()
      if (response.data) {
        queueStatus.value = response.data
      }
    } catch (err) {
      console.error('获取 AI 队列状态失败:', err)
    }
  }

  // 获取队列详情
  async function fetchQueueDetail(queueId: number, page = 1, pageSize = 20) {
    try {
      loading.value = true
      error.value = null
      const response = await aiApi.getQueueDetail(queueId, page, pageSize)
      if (response.data) {
        queueDetail.value = response.data
      }
      return response.data
    } catch (err) {
      error.value = err instanceof Error ? err.message : '获取队列详情失败'
      throw err
    } finally {
      loading.value = false
    }
  }

  // 重试队列所有失败图片
  async function retryQueueFailedImages(queueId: number) {
    try {
      loading.value = true
      error.value = null
      await aiApi.retryQueueFailedImages(queueId)
      await fetchQueueStatus()
      // 如果当前正在查看该队列的详情，也刷新详情
      if (queueDetail.value?.queue.id === queueId) {
        await fetchQueueDetail(queueId)
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : '重试失败'
      throw err
    } finally {
      loading.value = false
    }
  }

  // 重试单张图片
  async function retryTaskImage(taskImageId: number) {
    try {
      loading.value = true
      error.value = null
      await aiApi.retryTaskImage(taskImageId)
      await fetchQueueStatus()
      // 如果当前正在查看详情，也刷新详情
      if (queueDetail.value) {
        await fetchQueueDetail(queueDetail.value.queue.id, queueDetail.value.page, queueDetail.value.page_size)
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : '重试失败'
      throw err
    } finally {
      loading.value = false
    }
  }

  // 忽略单张图片
  async function ignoreTaskImage(taskImageId: number) {
    try {
      loading.value = true
      error.value = null
      await aiApi.ignoreTaskImage(taskImageId)
      await fetchQueueStatus()
      // 如果当前正在查看详情，也刷新详情
      if (queueDetail.value) {
        await fetchQueueDetail(queueDetail.value.queue.id, queueDetail.value.page, queueDetail.value.page_size)
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : '操作失败'
      throw err
    } finally {
      loading.value = false
    }
  }

  // 清除队列详情
  function clearQueueDetail() {
    queueDetail.value = null
  }

  // ================== 轮询控制 ==================

  function startPolling(intervalMs = 5000) {
    if (isPolling.value) return

    isPolling.value = true
    // 立即执行一次
    fetchQueueStatus()

    pollingInterval = setInterval(() => {
      fetchQueueStatus()
    }, intervalMs)
  }

  function stopPolling() {
    if (pollingInterval) {
      clearInterval(pollingInterval as ReturnType<typeof setInterval>)
      clearTimeout(pollingInterval as ReturnType<typeof setTimeout>)
      pollingInterval = null
    }
    isPolling.value = false
  }

  // 智能轮询：有活动任务时轮询，没有时停止
  function smartPolling(intervalMs = 3000) {
    if (isPolling.value) return

    const poll = async () => {
      await fetchQueueStatus()
      if (hasActiveTasks.value) {
        pollingInterval = setTimeout(poll, intervalMs)
      } else {
        isPolling.value = false
      }
    }

    isPolling.value = true
    poll()
  }

  return {
    // State
    config,
    queueStatus,
    queueDetail,
    loading,
    configLoading,
    error,
    isPolling,

    // Computed
    hasActiveTasks,
    hasFailedTasks,
    queues,
    queuesWithFailures,
    enabledModels,

    // Actions
    fetchConfig,
    updateConfig,
    testConnection,
    fetchQueueStatus,
    fetchQueueDetail,
    retryQueueFailedImages,
    retryTaskImage,
    ignoreTaskImage,
    clearQueueDetail,
    startPolling,
    stopPolling,
    smartPolling,
  }
})
