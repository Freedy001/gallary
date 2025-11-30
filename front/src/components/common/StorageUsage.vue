<template>
  <div class="space-y-2">
    <!-- 展开状态 -->
    <template v-if="!collapsed">
      <div class="flex items-center justify-between text-xs text-gray-500 font-mono tracking-wider">
        <span>存储空间</span>
        <span v-if="stats" class="text-gray-400">
          {{ formatBytes(stats.used_bytes) }} / {{ formatBytes(stats.total_bytes) }}
        </span>
        <span v-else class="text-gray-600">--</span>
      </div>
      <!-- 进度条 -->
      <div class="h-1.5 bg-white/5 rounded-full overflow-hidden">
        <div
            class="h-full rounded-full transition-all duration-500 ease-out"
            :class="progressBarClass"
            :style="{ width: `${usagePercent}%` }"
        ></div>
      </div>
    </template>

    <!-- 收起状态 -->
    <template v-else>
      <div class="flex flex-col items-center gap-1">
        <div class="w-10 h-1 bg-white/5 rounded-full overflow-hidden">
          <div
              class="h-full rounded-full transition-all duration-500 ease-out"
              :class="progressBarClass"
              :style="{ width: `${usagePercent}%` }"
          ></div>
        </div>
        <span class="text-[10px] text-gray-500">{{ usagePercent }}%</span>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import {ref, computed, onMounted} from 'vue'
import {storageApi} from "@/api/storage.ts";

interface StorageStats {
  used_bytes: number
  total_bytes: number
}

defineProps<{
  collapsed: boolean
}>()

const stats = ref<StorageStats | null>(null)

// 使用率百分比
const usagePercent = computed(() => {
  if (!stats.value || stats.value.total_bytes === 0) return 0
  return Math.round((stats.value.used_bytes / stats.value.total_bytes) * 100)
})

// 进度条样式（根据使用率变化颜色）
const progressBarClass = computed(() => {
  const percent = usagePercent.value
  if (percent >= 90) {
    return 'bg-gradient-to-r from-red-500 to-red-600 shadow-[0_0_10px_rgba(239,68,68,0.4)]'
  } else if (percent >= 70) {
    return 'bg-gradient-to-r from-yellow-500 to-orange-500 shadow-[0_0_10px_rgba(234,179,8,0.3)]'
  }
  return 'bg-gradient-to-r from-primary-500 to-primary-600 shadow-[0_0_10px_rgba(139,92,246,0.3)]'
})

// 格式化字节数
function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

// 获取存储统计
async function fetchStats() {
  try {
    stats.value = (await storageApi.getStorageStats()).data
  } catch (error) {
    console.error('获取存储统计失败:', error)
  }
}

onMounted(() => {
  fetchStats()
})
</script>
