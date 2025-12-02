<template>
  <div class="space-y-3">
    <!-- 展开状态 -->
    <template v-if="!collapsed">
      <div class="text-xs text-gray-500 font-mono tracking-wider mb-2">存储空间</div>

      <!-- 多提供者列表 -->
      <template v-if="storageStore.stats && storageStore.stats.providers.length > 0">
        <div
            v-for="provider in storageStore.stats.providers"
            :key="provider.id"
            class="space-y-1.5"
        >
          <div class="flex items-center justify-between text-xs">
            <span class="flex items-center gap-1.5">
              <span
                  class="w-1.5 h-1.5 rounded-full"
                  :class="provider.is_active ? 'bg-green-500' : 'bg-gray-600'"
              ></span>
              <span :class="provider.is_active ? 'text-gray-300' : 'text-gray-500'">
                {{ getProviderLabel(provider.id) }}
              </span>
            </span>
            <span class="text-gray-400 tabular-nums">
              {{ formatBytes(provider.used_bytes) }} / {{ formatBytes(provider.total_bytes) }}
            </span>
          </div>
          <!-- 进度条 -->
          <div class="h-1 bg-white/5 rounded-full overflow-hidden">
            <div
                class="h-full rounded-full transition-all duration-500 ease-out"
                :class="getProgressBarClass(getUsagePercent(provider))"
                :style="{ width: `${getUsagePercent(provider)}%` }"
            ></div>
          </div>
        </div>
      </template>

      <!-- 无数据 -->
      <div v-else class="text-xs text-gray-600">暂无存储信息</div>
    </template>

    <!-- 收起状态 - 显示汇总信息 -->
    <template v-else>
      <div class="flex flex-col items-center gap-1">
        <!-- 多个小进度条 -->
        <div class="flex flex-col gap-1 w-10">
          <template v-if="storageStore.stats && storageStore.stats.providers.length > 0">
            <div
                v-for="provider in storageStore.stats.providers"
                :key="provider.id"
                class="h-1 bg-white/5 rounded-full overflow-hidden"
                :title="getProviderLabel(provider.id)"
            >
              <div
                  class="h-full rounded-full transition-all duration-500 ease-out"
                  :class="getProgressBarClass(getUsagePercent(provider))"
                  :style="{ width: `${getUsagePercent(provider)}%` }"
              ></div>
            </div>
          </template>
          <div v-else class="h-1 bg-white/5 rounded-full"></div>
        </div>
        <span class="text-[10px] text-gray-500">{{ storageStore.stats?.providers.length || 0 }}</span>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useStorageStore } from '@/stores/storage'
import type { ProviderStats, StorageId } from '@/api/storage'
import { getStorageDriverName } from '@/api/storage'

defineProps<{
  collapsed: boolean
}>()

const storageStore = useStorageStore()

// 获取提供者显示名称
function getProviderLabel(id: StorageId): string {
  return getStorageDriverName(id)
}

// 计算使用率百分比
function getUsagePercent(provider: ProviderStats): number {
  if (!provider || provider.total_bytes === 0) return 0
  return Math.round((provider.used_bytes / provider.total_bytes) * 100)
}

// 进度条样式（根据使用率变化颜色）
function getProgressBarClass(percent: number): string {
  if (percent >= 90) {
    return 'bg-gradient-to-r from-red-500 to-red-600 shadow-[0_0_10px_rgba(239,68,68,0.4)]'
  } else if (percent >= 70) {
    return 'bg-gradient-to-r from-yellow-500 to-orange-500 shadow-[0_0_10px_rgba(234,179,8,0.3)]'
  }
  return 'bg-gradient-to-r from-primary-500 to-primary-600 shadow-[0_0_10px_rgba(139,92,246,0.3)]'
}

// 格式化字节数
function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

onMounted(() => {
  // 使用 store 的带缓存请求，不会重复请求
  storageStore.fetchStats()
})
</script>
