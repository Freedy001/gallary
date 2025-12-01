import { defineStore } from 'pinia'
import { ref } from 'vue'
import { storageApi, type StorageStats } from '@/api/storage'
import { imageApi } from '@/api/image'

export const useStorageStore = defineStore('storage', () => {
  // 存储统计数据
  const stats = ref<StorageStats | null>(null)
  // 加载状态
  const loading = ref(false)
  // 上次加载时间
  const lastFetchTime = ref<number>(0)
  // 缓存有效期（5分钟）
  const cacheValidDuration = 5 * 60 * 1000

  // 图片总数（全部影像，不包括回收站）
  const totalImages = ref<number>(0)
  const totalImagesFetchTime = ref<number>(0)

  // 获取存储统计（带缓存）
  async function fetchStats(force = false) {
    const now = Date.now()

    // 如果缓存有效且不是强制刷新，直接返回
    if (!force && stats.value && (now - lastFetchTime.value) < cacheValidDuration) {
      return stats.value
    }

    // 避免重复请求
    if (loading.value) {
      return stats.value
    }

    loading.value = true
    try {
      const response = await storageApi.getStorageStats()
      stats.value = response.data
      lastFetchTime.value = now
      return stats.value
    } catch (error) {
      console.error('获取存储统计失败:', error)
      return stats.value
    } finally {
      loading.value = false
    }
  }

  // 获取图片总数（带缓存）
  async function fetchTotalImages(force = false) {
    const now = Date.now()

    // 如果缓存有效且不是强制刷新，直接返回
    if (!force && totalImages.value > 0 && (now - totalImagesFetchTime.value) < cacheValidDuration) {
      return totalImages.value
    }

    try {
      // 只请求第一页，page_size=1，仅获取 total
      const response = await imageApi.getList(1, 1)
      totalImages.value = response.data.total
      totalImagesFetchTime.value = now
      return totalImages.value
    } catch (error) {
      console.error('获取图片总数失败:', error)
      return totalImages.value
    }
  }

  // 更新图片总数（上传/删除后调用）
  function updateTotalImages(delta: number) {
    totalImages.value = Math.max(0, totalImages.value + delta)
  }

  // 强制刷新
  async function refreshStats() {
    return fetchStats(true)
  }

  // 强制刷新图片总数
  async function refreshTotalImages() {
    return fetchTotalImages(true)
  }

  // 清除缓存
  function clearCache() {
    stats.value = null
    lastFetchTime.value = 0
    totalImagesFetchTime.value = 0
  }

  return {
    stats,
    loading,
    totalImages,
    fetchStats,
    fetchTotalImages,
    updateTotalImages,
    refreshStats,
    refreshTotalImages,
    clearCache,
  }
})
