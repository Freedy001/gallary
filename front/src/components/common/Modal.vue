<template>
  <Teleport to="body">
    <Transition name="fade">
      <div
        v-if="modelValue"
        class="fixed inset-0 z-50 flex items-center justify-center overflow-y-auto bg-black bg-opacity-50 p-4"
        @click.self="handleClose"
      >
        <Transition name="modal">
          <div
            v-if="modelValue"
            :class="[
              'relative w-full rounded-lg bg-white shadow-xl',
              sizeClasses,
            ]"
            @click.stop
          >
            <!-- 头部 -->
            <div v-if="title || $slots.header" class="flex items-center justify-between border-b border-gray-200 px-6 py-4">
              <slot name="header">
                <h3 class="text-lg font-semibold text-gray-900">{{ title }}</h3>
              </slot>

              <button
                v-if="closable"
                @click="handleClose"
                class="rounded-lg p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
              >
                <XMarkIcon class="h-5 w-5" />
              </button>
            </div>

            <!-- 内容 -->
            <div class="px-6 py-4">
              <slot />
            </div>

            <!-- 底部 -->
            <div v-if="$slots.footer" class="border-t border-gray-200 px-6 py-4">
              <slot name="footer" />
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { XMarkIcon } from '@heroicons/vue/24/outline'

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
.modal-enter-active,
.modal-leave-active {
  transition: transform 0.3s ease, opacity 0.3s ease;
}

.modal-enter-from {
  transform: scale(0.95);
  opacity: 0;
}

.modal-leave-to {
  transform: scale(0.95);
  opacity: 0;
}
</style>
