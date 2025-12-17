import { defineStore } from 'pinia'
import { ref } from 'vue'
import { wsService } from '@/services/websocket'
import type { AIQueueStatus } from '@/types/ai'
import type { StorageStats } from '@/api/storage'

export const useNotificationStore = defineStore('notification', () => {
  // ================== State ==================

  // AI 队列状态
  const aiQueueStatus = ref<AIQueueStatus | null>(null)

  // 存储统计
  const storageStats = ref<StorageStats | null>(null)

  // 图片总数
  const imageCount = ref<number>(0)

  wsService.connect({
    onConnected: () => {
      console.log('[Notification] WebSocket 已连接')
    },

    onDisconnected: () => {
      // WebSocket 断开连接
    },

    onReconnecting: () => {
      // WebSocket 重连中
    },

    onError: (error) => {
      console.error('[Notification] WebSocket 错误', error)
    }
  })

  // 订阅 AI 队列状态更新
  wsService.subscribe<AIQueueStatus>('ai_queue_status', (data) => {
    aiQueueStatus.value = data
  })

  // 订阅存储统计更新
  wsService.subscribe<StorageStats>('storage_stats', (data) => {
    storageStats.value = data
  })

  // 订阅图片总数更新
  wsService.subscribe<number>('image_count', (count) => {
    imageCount.value = count
  })

  // ================== Actions ==================
  /**
   * 断开连接
   */
  function disconnect() {
    wsService.disconnect()
  }

  return {
    // State
    aiQueueStatus,
    storageStats,
    imageCount,

    // Actions
    disconnect,
  }
})
