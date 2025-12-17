<template>
  <Teleport to="body">
    <Transition name="popover">
      <div
        v-if="visible"
        ref="modalRef"
        class="fixed z-[40] w-80 flex flex-col"
        :style="positionStyle"
      >
        <LiquidGlassCard
            class="w-full h-full flex flex-col"
            :hover-effect="false"
            content-class="p-0 flex flex-col h-full"
        >
          <!-- 头部 -->
          <div class="flex items-center justify-between px-4 py-3 border-b border-white/10 bg-white/5">
            <h3 class="text-sm font-medium text-white flex items-center gap-2">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-4 h-4 text-primary-400">
                <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 0v3.75m-16.5-3.75v3.75m16.5 0v3.75C20.25 16.153 16.556 18 12 18s-8.25-1.847-8.25-4.125v-3.75m16.5 0c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125" />
              </svg>
              存储空间详情
            </h3>
            <button
              @click="$emit('close')"
              class="text-white/40 hover:text-white transition-colors p-1 hover:bg-white/10 rounded"
            >
              <XMarkIcon class="h-4 w-4" />
            </button>
          </div>

          <!-- 内容区域 -->
          <div class="p-4 space-y-4 flex-1 overflow-y-auto min-h-0 custom-scrollbar">
            <template v-if="notificationStore.storageStats && notificationStore.storageStats.providers.length > 0">
              <div
                v-for="provider in notificationStore.storageStats.providers"
                :key="provider.id"
                class="glass-item"
              >
                <div class="flex items-center justify-between mb-2">
                  <div class="flex items-center gap-2">
                    <div
                      class="w-2 h-2 rounded-full shadow-[0_0_5px_currentColor]"
                      :class="provider.is_active ? 'bg-green-500 text-green-500' : 'bg-gray-600 text-gray-600'"
                    ></div>
                    <span class="text-xs font-medium text-gray-200 truncate max-w-[120px]" :title="getStorageDriverName(provider.id)">
                      {{ getStorageDriverName(provider.id) }}
                    </span>
                  </div>
                  <span class="text-[10px] text-white/50 font-mono">
                    {{ formatBytes(provider.used_bytes) }} / {{ formatBytes(provider.total_bytes) }}
                  </span>
                </div>

                <!-- 进度条 -->
                <div class="h-1.5 bg-black/40 rounded-full overflow-hidden mb-1">
                  <div
                    class="h-full rounded-full transition-all duration-500 ease-out relative overflow-hidden"
                    :class="getProgressBarClass(getUsagePercent(provider))"
                    :style="{ width: `${getUsagePercent(provider)}%` }"
                  >
                    <!-- 光泽效果 -->
                    <div class="absolute inset-0 bg-gradient-to-b from-white/20 to-transparent"></div>
                  </div>
                </div>
              </div>
            </template>
            <div v-else class="py-8 text-center flex flex-col items-center justify-center text-white/30 gap-2">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-8 h-8 opacity-20">
                <path stroke-linecap="round" stroke-linejoin="round" d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 0v3.75m-16.5-3.75v3.75m16.5 0v3.75C20.25 16.153 16.556 18 12 18s-8.25-1.847-8.25-4.125v-3.75m16.5 0c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125" />
              </svg>
              <span class="text-xs">暂无存储设备信息</span>
            </div>
          </div>
        </LiquidGlassCard>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { XMarkIcon } from '@heroicons/vue/24/outline'
import { useNotificationStore } from '@/stores/notification.ts'
import { getStorageDriverName } from '@/api/storage.ts'
import type { ProviderStats } from '@/api/storage.ts'
import { onClickOutside } from '@vueuse/core'
import LiquidGlassCard from '@/components/common/LiquidGlassCard.vue'

// ... (keep existing props, emits, and logic) ...
const props = defineProps<{
  visible: boolean
  triggerRect: DOMRect | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const notificationStore = useNotificationStore()
const modalRef = ref<HTMLElement | null>(null)

// 点击外部关闭
onClickOutside(modalRef, () => {
  if (props.visible) {
    emit('close')
  }
})

// 计算位置
const positionStyle = computed(() => {
  if (!props.triggerRect) return {}

  const { left, width, bottom, top } = props.triggerRect
  const gap = 12 // 间距
  const padding = 20 // 屏幕边缘最小间距
  const x = left + width + gap
  const windowHeight = window.innerHeight

  // 判断触发器在屏幕的位置（上半部分还是下半部分）
  const isLowerHalf = top > windowHeight / 2

  if (isLowerHalf) {
    // 下半部分：底部对齐
    // 计算底部距离
    const bottomOffset = windowHeight - bottom
    // 计算可用高度（从底部位置向上到顶部留出padding）
    const availableHeight = bottom - padding

    return {
      left: `${x}px`,
      bottom: `${bottomOffset}px`,
      maxHeight: `${availableHeight}px`,
      transformOrigin: 'left bottom'
    }
  } else {
    // 上半部分：顶部对齐
    // 计算顶部距离
    const topOffset = top
    // 计算可用高度（从顶部位置向下到底部留出padding）
    const availableHeight = windowHeight - top - padding

    return {
      left: `${x}px`,
      top: `${topOffset}px`,
      maxHeight: `${availableHeight}px`,
      transformOrigin: 'left top'
    }
  }
})

function getUsagePercent(provider: ProviderStats): number {
  if (!provider || provider.total_bytes === 0) return 0
  return Math.round((provider.used_bytes / provider.total_bytes) * 100)
}

function getProgressBarClass(percent: number): string {
  if (percent >= 90) {
    return 'bg-gradient-to-r from-red-500 to-red-600 shadow-[0_0_8px_rgba(239,68,68,0.3)]'
  } else if (percent >= 70) {
    return 'bg-gradient-to-r from-yellow-500 to-orange-500 shadow-[0_0_8px_rgba(234,179,8,0.3)]'
  }
  return 'bg-gradient-to-r from-primary-500 to-primary-600 shadow-[0_0_8px_rgba(139,92,246,0.3)]'
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}
</script>

<style scoped>
.glass-item {
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 0.75rem;
  padding: 0.75rem;
  transition: all 0.2s ease;
}

.glass-item:hover {
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.15);
  transform: translateY(-1px);
}

.custom-scrollbar::-webkit-scrollbar {
  width: 4px;
}

.custom-scrollbar::-webkit-scrollbar-track {
  background: rgba(255, 255, 255, 0.02);
  border-radius: 2px;
}

.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 2px;
}

.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.2);
}

.popover-enter-active,
.popover-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.popover-enter-from,
.popover-leave-to {
  opacity: 0;
  transform: scale(0.95) translateX(-10px);
}
</style>