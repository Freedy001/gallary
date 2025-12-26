<template>
  <Teleport to="body">
    <Transition name="modal-overlay">
      <div
        v-if="modelValue"
        class="fixed inset-0 z-50 flex items-center justify-center overflow-y-auto bg-black/60 backdrop-blur-sm"
        @click.self="handleClose"
      >
        <Transition name="modal-content" appear>
          <LiquidGlassCard
            v-if="modelValue"
            :class="[
              'relative w-full mx-auto',
              sizeClasses,
            ]"
            :hover-effect="false"
            content-class="p-0"
            @click.stop
          >
            <!-- 头部 -->
            <div v-if="title || $slots.header" class="flex items-center justify-between border-b border-white/10 px-6 py-4">
              <slot name="header">
                <h3 class="text-xl font-semibold text-white">{{ title }}</h3>
              </slot>

              <button
                v-if="closable"
                @click="handleClose"
                class="rounded-lg p-1 text-white/50 hover:bg-white/10 hover:text-white transition-colors"
              >
                <XMarkIcon class="h-5 w-5" />
              </button>
            </div>

            <!-- 内容 -->
            <div class="px-6 py-4 max-h-[80vh] overflow-y-auto custom-scrollbar">
              <slot />
            </div>

            <!-- 底部 -->
            <div v-if="$slots.footer" class="border-t border-white/10 px-6 py-4">
              <slot name="footer" />
            </div>
          </LiquidGlassCard>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { XMarkIcon } from '@heroicons/vue/24/outline'
import LiquidGlassCard from '@/components/common/LiquidGlassCard.vue'

interface Props {
  modelValue: boolean
  title?: string
  size?: 'sm' | 'md' | 'lg' | 'xl' | 'full'
  closable?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  size: 'md',
  closable: true,
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  close: []
}>()

const sizeClasses = computed(() => {
  const sizes = {
    sm: 'max-w-sm',
    md: 'max-w-md',
    lg: 'max-w-lg',
    xl: 'max-w-xl',
    full: 'max-w-full',
  }
  return sizes[props.size]
})

function handleClose() {
  if (props.closable) {
    emit('update:modelValue', false)
    emit('close')
  }
}
</script>

<style scoped>
/* 自定义滚动条样式 */
.custom-scrollbar::-webkit-scrollbar {
  width: 8px;
}

.custom-scrollbar::-webkit-scrollbar-track {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 4px;
}

.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 4px;
  transition: background 0.2s ease;
}

.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.3);
}

/* 背景遮罩层动画 - 保持 backdrop-filter 稳定 */
.modal-overlay-enter-active {
  transition: opacity 0.3s ease;
}

.modal-overlay-leave-active {
  transition: opacity 0.2s ease;
}

.modal-overlay-enter-from,
.modal-overlay-leave-to {
  opacity: 0;
}

/* 内容区域动画 */
.modal-content-enter-active {
  transition: transform 0.3s ease, opacity 0.3s ease;
}

.modal-content-leave-active {
  transition: transform 0.2s ease, opacity 0.2s ease;
}

.modal-content-enter-from {
  transform: scale(0.95) translateY(10px);
  opacity: 0;
}

.modal-content-leave-to {
  transform: scale(0.95) translateY(-10px);
  opacity: 0;
}
</style>
