<template>
  <div
    ref="containerRef"
    class="group cursor-pointer hover:bg-white/5 -mx-2 px-2 py-2 rounded-lg transition-colors"
    @click="openModal"
  >
    <!-- 展开状态 -->
    <template v-if="!collapsed">
      <div class="flex items-center justify-between text-xs text-gray-500 font-mono tracking-wider mb-2">
        <span>存储空间</span>
        <ArrowRightIcon class="h-3 w-3 opacity-0 group-hover:opacity-100 transition-opacity" />
      </div>

      <!-- 总体使用情况摘要 -->
      <div class="space-y-1.5">
        <div class="flex items-center justify-between text-xs">
          <span class="text-gray-300">
             {{ activeProviderCount }} 个存储区
          </span>
          <span class="text-gray-400 tabular-nums">
            {{ formatBytes(totalUsed) }} / {{ formatBytes(totalCapacity) }}
          </span>
        </div>
        <!-- 进度条 -->
        <div class="h-1 bg-white/5 rounded-full overflow-hidden">
          <div
              class="h-full rounded-full transition-all duration-500 ease-out"
              :class="getProgressBarClass(totalPercent)"
              :style="{ width: `${totalPercent}%` }"
          ></div>
        </div>
      </div>
    </template>

    <!-- 收起状态 - 显示汇总信息 -->
    <template v-else>
      <div class="flex flex-col items-center gap-1">
        <!-- 简单的环形进度或小进度条 -->
         <div class="h-8 w-1 bg-white/5 rounded-full overflow-hidden flex flex-col justify-end">
           <div
              class="w-full rounded-full transition-all duration-500 ease-out"
              :class="getProgressBarClass(totalPercent)"
              :style="{ height: `${totalPercent}%` }"
           ></div>
         </div>
         <span class="text-[10px] text-gray-500">{{ Math.round(totalPercent) }}%</span>
      </div>
    </template>

    <!-- 详情弹窗 -->
    <StorageDetailModal
        :visible="modalVisible"
        :trigger-rect="triggerRect"
        @close="modalVisible = false"
    />
  </div>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'
import {ArrowRightIcon} from '@heroicons/vue/24/outline'
import {useNotificationStore} from '@/stores/notification.ts'
import StorageDetailModal from '@/components/widgets/storage/StorageDetailModal.vue'

defineProps<{
  collapsed: boolean
}>()

const notificationStore = useNotificationStore()
const containerRef = ref<HTMLElement | null>(null)
const modalVisible = ref(false)
const triggerRect = ref<DOMRect | null>(null)

// 打开弹窗
function openModal() {
  // 获取点击时的触发区域位置
  if (containerRef.value) {
    triggerRect.value = containerRef.value.getBoundingClientRect()
    modalVisible.value = true
  }
}

// 计算属性
const activeProviderCount = computed(() => {
  return notificationStore.storageStats?.providers.filter(p => p.is_active).length || 0
})

const totalUsed = computed(() => {
  return notificationStore.storageStats?.providers.reduce((acc, p) => acc + p.used_bytes, 0) || 0
})

const totalCapacity = computed(() => {
  return notificationStore.storageStats?.providers.reduce((acc, p) => acc + p.total_bytes, 0) || 0
})

const totalPercent = computed(() => {
  if (totalCapacity.value === 0) return 0
  return Math.min(100, (totalUsed.value / totalCapacity.value) * 100)
})

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

</script>