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
              <SparklesIcon class="h-4 w-4 text-primary-400" />
              AI 处理队列
            </h3>
            <button
              @click="$emit('close')"
              class="text-white/40 hover:text-white transition-colors p-1 hover:bg-white/10 rounded"
            >
              <XMarkIcon class="h-4 w-4" />
            </button>
          </div>

          <!-- 内容区域 -->
          <div class="p-2 space-y-2 flex-1 overflow-y-auto min-h-0 custom-scrollbar">
            <template v-if="aiStore.queues.length > 0">
              <div
                v-for="queue in aiStore.queues"
                :key="queue.id"
                class="glass-item group"
              >
                <div class="flex items-center justify-between mb-2">
                  <div class="flex items-center gap-2">
                    <div class="w-2 h-2 rounded-full shadow-[0_0_5px_currentColor]" :class="getQueueStatusDotClass(queue)"></div>
                    <span :title="getQueueDisplayName(queue)" class="text-xs font-medium text-gray-200 truncate max-w-[200px]">
                      {{ getQueueDisplayName(queue) }}
                    </span>
                  </div>

                  <!-- 详情按钮 (仅在有失败或处理中显示，或者hover时显示) -->
                   <button
                      v-if="queue.failed_count > 0"
                      @click.stop="openQueueDetail(queue)"
                      class="text-[10px] px-1.5 py-0.5 bg-red-500/20 text-red-300 rounded hover:bg-red-500/30 transition-colors border border-red-500/20"
                  >
                    {{ queue.failed_count }} 失败
                  </button>
                   <span v-else class="text-[10px] text-white/40 font-mono">
                     {{ queue.pending_count }} 任务
                   </span>
                </div>

                <!-- 统计数字 -->
                <div class="grid grid-cols-2 gap-1 mb-2 text-[10px] text-white/50 bg-black/20 rounded p-1.5 border border-white/5">
                  <div class="text-center">
                     <span class="font-bold" :class="queue.pending_count > 0 ? 'text-yellow-400' : ''">{{ queue.pending_count }}</span> 待处理
                  </div>
                  <div class="text-center border-l border-white/10">
                     <span class="font-bold" :class="queue.failed_count > 0 ? 'text-red-400' : ''">{{ queue.failed_count }}</span> 失败
                  </div>
                </div>
              </div>
            </template>
            <div v-else class="py-8 text-center flex flex-col items-center justify-center text-white/30 gap-2">
              <SparklesIcon class="h-8 w-8 opacity-20" />
              <span class="text-xs">暂无 AI 任务</span>
            </div>
          </div>
        </LiquidGlassCard>
      </div>
    </Transition>

    <!-- 详情弹窗 -->
    <AIQueueDetailModal
        v-if="selectedQueue"
        :queue="selectedQueue"
        :visible="!!selectedQueue"
        @close="closeQueueDetail"
    />
  </Teleport>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'
import {SparklesIcon, XMarkIcon} from '@heroicons/vue/24/outline'
import {useAIStore} from '@/stores/ai.ts'
import type {AIQueueInfo} from '@/types/ai.ts'
import {getQueueDisplayName} from '@/types/ai.ts'
import {onClickOutside} from '@vueuse/core'
import AIQueueDetailModal from '@/components/widgets/AIQueueDetailModal.vue'
import LiquidGlassCard from '@/components/common/LiquidGlassCard.vue'

const props = defineProps<{
  visible: boolean
  triggerRect: DOMRect | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const aiStore = useAIStore()
const modalRef = ref<HTMLElement | null>(null)

// 详情弹窗状态
const selectedQueue = ref<AIQueueInfo | null>(null)

// 点击外部关闭
onClickOutside(modalRef, () => {
  if (props.visible && !selectedQueue.value) { // 如果打开了详情弹窗，不关闭列表弹窗
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

function getQueueStatusDotClass(queue: AIQueueInfo): string {
  if (queue.status === 'processing') {
    return 'bg-primary-500 animate-pulse text-primary-500'
  }
  if (queue.pending_count > 0) {
    return 'bg-yellow-500 text-yellow-500'
  }
  if (queue.failed_count > 0) {
    return 'bg-red-500 text-red-500'
  }
  return 'bg-green-500 text-green-500'
}

function openQueueDetail(queue: AIQueueInfo) {
  selectedQueue.value = queue
}

function closeQueueDetail() {
  selectedQueue.value = null
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