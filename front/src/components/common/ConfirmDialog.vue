<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="dialogStore.state.visible"
        class="fixed inset-0 z-[100] flex items-center justify-center overflow-y-auto px-4 py-6 sm:px-0"
        role="dialog"
        aria-modal="true"
      >
        <!-- Backdrop -->
        <div
          class="fixed inset-0 bg-black/60 backdrop-blur-sm transition-opacity"
          @click="handleBackdropClick"
        ></div>

        <!-- Modal Panel -->
        <div
            class="relative w-full max-w-md -translate-y-20 overflow-hidden rounded-2xl border border-white/10 bg-[#121212] p-6 text-left align-middle shadow-[0_0_40px_rgba(0,0,0,0.5)] transition-all"
        >
          <!-- Glow Effect -->
          <div
            class="absolute -top-20 -right-20 h-40 w-40 rounded-full blur-3xl opacity-20 pointer-events-none"
            :class="glowColor"
          ></div>

          <div class="relative flex gap-4">
            <!-- Icon -->
            <div
              class="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full ring-1 ring-white/10"
              :class="iconBgColor"
            >
              <component :is="currentIcon" class="h-6 w-6" :class="iconColor" />
            </div>

            <!-- Content -->
            <div class="flex-1 pt-1">
              <h3 class="text-lg font-medium leading-6 text-white">
                {{ dialogStore.state.title }}
              </h3>
              <div class="mt-2">
                <p class="text-sm text-gray-400 leading-relaxed">
                  {{ dialogStore.state.message }}
                </p>
              </div>
            </div>
          </div>

          <!-- Actions -->
          <div class="mt-8 flex justify-end gap-3">
            <button
              v-if="dialogStore.state.cancelText"
              @click="dialogStore.handleCancel"
              class="rounded-xl px-4 py-2.5 text-sm font-medium text-gray-400 transition-colors hover:bg-white/5 hover:text-white focus:outline-none focus:ring-2 focus:ring-white/10"
            >
              {{ dialogStore.state.cancelText }}
            </button>

            <button
              @click="dialogStore.handleConfirm"
              class="rounded-xl px-5 py-2.5 text-sm font-medium text-white shadow-lg transition-all hover:scale-105 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-[#121212]"
              :class="confirmBtnClass"
            >
              {{ dialogStore.state.confirmText }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useDialogStore } from '@/stores/dialog'
import {
  ExclamationTriangleIcon,
  InformationCircleIcon,
  CheckCircleIcon,
  XCircleIcon
} from '@heroicons/vue/24/outline'

const dialogStore = useDialogStore()

const currentIcon = computed(() => {
  switch (dialogStore.state.type) {
    case 'success': return CheckCircleIcon
    case 'error': return XCircleIcon
    case 'warning': return ExclamationTriangleIcon
    case 'confirm': return InformationCircleIcon
    default: return InformationCircleIcon
  }
})

const glowColor = computed(() => {
  switch (dialogStore.state.type) {
    case 'success': return 'bg-green-500'
    case 'error': return 'bg-red-500'
    case 'warning': return 'bg-yellow-500'
    case 'confirm': return 'bg-primary-500'
    default: return 'bg-primary-500'
  }
})

const iconBgColor = computed(() => {
  switch (dialogStore.state.type) {
    case 'success': return 'bg-green-500/10'
    case 'error': return 'bg-red-500/10'
    case 'warning': return 'bg-yellow-500/10'
    case 'confirm': return 'bg-primary-500/10'
    default: return 'bg-primary-500/10'
  }
})

const iconColor = computed(() => {
  switch (dialogStore.state.type) {
    case 'success': return 'text-green-400'
    case 'error': return 'text-red-400'
    case 'warning': return 'text-yellow-400'
    case 'confirm': return 'text-primary-400'
    default: return 'text-primary-400'
  }
})

const confirmBtnClass = computed(() => {
  switch (dialogStore.state.type) {
    case 'error':
      return 'bg-red-500 hover:bg-red-600 hover:shadow-red-500/20 focus:ring-red-500'
    case 'warning':
      return 'bg-yellow-600 hover:bg-yellow-700 hover:shadow-yellow-500/20 focus:ring-yellow-600'
    case 'success':
      return 'bg-green-600 hover:bg-green-700 hover:shadow-green-500/20 focus:ring-green-600'
    default:
      return 'bg-primary-600 hover:bg-primary-500 hover:shadow-primary-500/20 focus:ring-primary-500'
  }
})

function handleBackdropClick() {
  // Only close on backdrop click if it's not a confirm dialog (optional choice)
  // For now, let's make it cancel
  dialogStore.handleCancel()
}
</script>
